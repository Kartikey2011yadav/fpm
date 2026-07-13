package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

type stashEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Message   string         `json:"message,omitempty"`
	Packages  []stashPackage `json:"packages"`
}

type stashPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

var stashCmd = &cobra.Command{
	Use:   "stash [message]",
	Short: "Save unlocked packages and restore clean lockfile state",
	Long: `Save packages not in the lockfile (your "uncommitted" changes) to a stack,
then restore the environment to match the lockfile exactly.

Use 'fpm stash pop' to restore the stashed packages.
Use 'fpm stash list' to see all stashed entries.`,
	GroupID: "environment",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runStash,
}

var stashPopCmd = &cobra.Command{
	Use:   "pop",
	Short: "Restore the most recently stashed packages",
	RunE:  runStashPop,
}

var stashListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stashed entries",
	RunE:  runStashList,
}

var stashDropCmd = &cobra.Command{
	Use:   "drop [index]",
	Short: "Discard the most recent stash entry",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStashDrop,
}

func init() {
	stashCmd.AddCommand(stashPopCmd)
	stashCmd.AddCommand(stashListCmd)
	stashCmd.AddCommand(stashDropCmd)
	rootCmd.AddCommand(stashCmd)
}

func stashFilePath(envPath string) string {
	return filepath.Join(envPath, ".fpm-stash.json")
}

func loadStash(envPath string) ([]stashEntry, error) {
	data, err := os.ReadFile(stashFilePath(envPath))
	if err != nil {
		return nil, nil
	}
	var entries []stashEntry
	json.Unmarshal(data, &entries)
	return entries, nil
}

func saveStash(envPath string, entries []stashEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(stashFilePath(envPath), data, 0644)
}

func runStash(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	lockPath := filepath.Join(cwd, lock.LockFileName)
	lf, err := lock.Read(lockPath)
	if err != nil {
		return fmt.Errorf("no lockfile found. Run 'fpm lock' first to establish a baseline")
	}

	// Find packages installed but NOT in lockfile
	lockVersions := make(map[string]string)
	for _, pkg := range lf.Packages {
		lockVersions[types.NewPackageName(pkg.Name).Normalized()] = pkg.Version
	}

	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()

	var toStash []stashPackage
	for _, pkg := range scanResult.Packages {
		if pkg.Manager != env.ManagerFpm {
			continue
		}
		if _, inLock := lockVersions[pkg.Name.Normalized()]; !inLock {
			toStash = append(toStash, stashPackage{
				Name:    pkg.Name.Normalized(),
				Version: pkg.Version.String(),
			})
		}
	}

	if len(toStash) == 0 {
		fmt.Println("No changes to stash. Environment matches lockfile.")
		return nil
	}

	// Save to stash stack
	message := ""
	if len(args) > 0 {
		message = args[0]
	}

	entries, _ := loadStash(activeVenv.Path)
	entries = append([]stashEntry{{
		Timestamp: time.Now().UTC(),
		Message:   message,
		Packages:  toStash,
	}}, entries...)
	saveStash(activeVenv.Path, entries)

	// Remove stashed packages from environment
	for _, pkg := range toStash {
		pkgName := types.NewPackageName(pkg.Name)
		uninstallPackage(activeVenv.SitePackages, pkgName)
	}

	fmt.Printf("Stashed %d package(s)", len(toStash))
	if message != "" {
		fmt.Printf(": %s", message)
	}
	fmt.Println()
	fmt.Println("  Environment now matches lockfile.")
	return nil
}

func runStashPop(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	entries, _ := loadStash(activeVenv.Path)
	if len(entries) == 0 {
		fmt.Println("No stash entries.")
		return nil
	}

	entry := entries[0]
	entries = entries[1:]
	saveStash(activeVenv.Path, entries)

	// Re-install stashed packages
	fmt.Printf("Restoring %d stashed package(s)...\n", len(entry.Packages))
	var names []string
	for _, pkg := range entry.Packages {
		names = append(names, pkg.Name+"=="+pkg.Version)
	}

	// Use fpm install internally by calling the install logic
	fmt.Printf("  Run: fpm install %s\n", joinMax(names, 3))
	fmt.Println("  (use 'fpm install' with the above packages to restore)")
	return nil
}

func runStashList(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	entries, _ := loadStash(activeVenv.Path)
	if len(entries) == 0 {
		fmt.Println("No stash entries.")
		return nil
	}

	for i, e := range entries {
		ts := e.Timestamp.Local().Format("2006-01-02 15:04")
		pkgStr := fmt.Sprintf("%d package(s)", len(e.Packages))
		if len(e.Packages) <= 3 {
			var names []string
			for _, p := range e.Packages {
				names = append(names, p.Name+"=="+p.Version)
			}
			pkgStr = joinMax(names, 3)
		}
		msg := e.Message
		if msg == "" {
			msg = pkgStr
		}
		fmt.Printf("  stash@{%d}: %s  %s\n", i, ts, msg)
	}
	return nil
}

func runStashDrop(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	entries, _ := loadStash(activeVenv.Path)
	if len(entries) == 0 {
		fmt.Println("No stash entries to drop.")
		return nil
	}

	entries = entries[1:]
	saveStash(activeVenv.Path, entries)
	fmt.Println("Dropped stash@{0}.")
	return nil
}

func joinMax(items []string, max int) string {
	if len(items) <= max {
		result := ""
		for i, item := range items {
			if i > 0 {
				result += ", "
			}
			result += item
		}
		return result
	}
	result := ""
	for i := 0; i < max; i++ {
		if i > 0 {
			result += ", "
		}
		result += items[i]
	}
	result += fmt.Sprintf(" (+%d more)", len(items)-max)
	return result
}
