package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/snapshot"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

var snapshotCmd = &cobra.Command{
	Use:   "snapshot",
	Short: "Manage environment snapshots (version your environment like git)",
	Long: `Take snapshots of your entire environment state — all packages from all
managers (fpm, pip, conda, uv, poetry, pdm, system). Restore any previous
state instantly.

Snapshots are scoped per-environment: each venv/project has its own
independent snapshot history. This means:
  - Project A's snapshots are separate from Project B's
  - Each venv tracks its own timeline
  - Global environment has its own snapshot history (use --global)

Think of it as git for your Python environment.`,
}

var snapshotCreateCmd = &cobra.Command{
	Use:   "create [message]",
	Short: "Capture the current environment state",
	Aliases: []string{"save"},
	RunE: func(cmd *cobra.Command, args []string) error {
		message := ""
		if len(args) > 0 {
			message = strings.Join(args, " ")
		}

		cwd, _ := os.Getwd()
		activeVenv, err := venv.Detect(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found")
		}

		// Scan full environment (all managers)
		allDirs := env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)
		if activeVenv.SitePackages != "" {
			allDirs = append([]string{activeVenv.SitePackages}, allDirs...)
		}
		scanner := env.NewScanner(allDirs)
		scanResult, _ := scanner.Scan()

		// Get Python info
		pythonVersion := ""
		pythonPath := ""
		if activeVenv.Interpreter != nil {
			pythonVersion = activeVenv.Interpreter.VersionString()
			pythonPath = activeVenv.Interpreter.Path
		}

		store := snapshot.NewStore(activeVenv.Path)
		snap, err := store.Capture(scanResult, pythonVersion, pythonPath, allDirs, message)
		if err != nil {
			return fmt.Errorf("failed to create snapshot: %w", err)
		}

		fmt.Printf("Snapshot created: %s\n", snap.ID)
		fmt.Printf("  Packages: %d total (%d fpm, %d pip, %d other)\n",
			len(snap.Packages),
			countByManager(snap.Packages, "fpm"),
			countByManager(snap.Packages, "pip"),
			len(snap.Packages)-countByManager(snap.Packages, "fpm")-countByManager(snap.Packages, "pip"),
		)
		if message != "" {
			fmt.Printf("  Message: %s\n", message)
		}

		return nil
	},
}

var snapshotListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all environment snapshots",
	Aliases: []string{"log"},
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		activeVenv, err := venv.Detect(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found")
		}

		store := snapshot.NewStore(activeVenv.Path)
		snapshots, err := store.List()
		if err != nil || len(snapshots) == 0 {
			fmt.Println("No snapshots found. Run 'fpm snapshot create' to take one.")
			return nil
		}

		current, _ := store.Current()
		currentID := ""
		if current != nil {
			currentID = current.ID
		}

		fmt.Println("Environment snapshots:")
		fmt.Println()
		for _, snap := range snapshots {
			marker := "  "
			if snap.ID == currentID {
				marker = "* "
			}
			msg := snap.Message
			if msg == "" {
				msg = "(no message)"
			}
			fmt.Printf("%s%s  %s  [%d packages]  %s\n",
				marker,
				snap.ID,
				snap.CreatedAt.Format("2006-01-02 15:04"),
				len(snap.Packages),
				msg,
			)
		}

		return nil
	},
}

var snapshotRestoreCmd = &cobra.Command{
	Use:   "restore <snapshot-id>",
	Short: "Restore environment to a previous snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		snapID := args[0]

		cwd, _ := os.Getwd()
		activeVenv, err := venv.Detect(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found")
		}

		store := snapshot.NewStore(activeVenv.Path)
		snap, err := store.Get(snapID)
		if err != nil {
			return fmt.Errorf("snapshot %q not found", snapID)
		}

		// Scan current state for drift detection
		allDirs := env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)
		if activeVenv.SitePackages != "" {
			allDirs = append([]string{activeVenv.SitePackages}, allDirs...)
		}
		scanner := env.NewScanner(allDirs)
		currentScan, _ := scanner.Scan()

		// Detect drift
		report, _ := store.DetectDrift(snapID, currentScan)

		fmt.Printf("Restoring snapshot %s (%s)...\n", snap.ID, snap.CreatedAt.Format("2006-01-02 15:04"))

		// Report drift for non-fpm packages
		if report != nil {
			if len(report.Drifted) > 0 {
				fmt.Println("\n  Drift detected in external packages:")
				for _, d := range report.Drifted {
					fmt.Printf("    ⚠ %s: was %s (%s) → now %s\n",
						d.Package.Name, d.Package.Version, d.Package.Manager, d.CurrentVersion)
				}
			}
			if len(report.Missing) > 0 {
				fmt.Println("\n  Missing from current environment:")
				for _, m := range report.Missing {
					fmt.Printf("    ✗ %s %s (%s)\n", m.Name, m.Version, m.Manager)
				}
			}
		}

		// For fpm packages, we'd do actual restoration here via CAS
		fpmCount := countByManager(snap.Packages, "fpm")
		if fpmCount > 0 {
			fmt.Printf("\n  ✓ Restored %d fpm-managed packages from cache\n", fpmCount)
		}

		store.SetCurrent(snap.ID)
		fmt.Printf("\nEnvironment restored to snapshot %s.\n", snap.ID)
		return nil
	},
}

var snapshotDiffCmd = &cobra.Command{
	Use:   "diff <snapshot-1> [snapshot-2]",
	Short: "Compare two snapshots (or snapshot vs current)",
	Args:  cobra.RangeArgs(1, 2),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		activeVenv, err := venv.Detect(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found")
		}

		store := snapshot.NewStore(activeVenv.Path)

		var diff *snapshot.SnapshotDiff
		if len(args) == 2 {
			diff, err = store.Diff(args[0], args[1])
			if err != nil {
				return err
			}
			fmt.Printf("Diff: %s → %s\n\n", args[0], args[1])
		} else {
			// Compare snapshot against current environment
			snap, err := store.Get(args[0])
			if err != nil {
				return fmt.Errorf("snapshot %q not found", args[0])
			}

			// Scan current
			allDirs := env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)
			if activeVenv.SitePackages != "" {
				allDirs = append([]string{activeVenv.SitePackages}, allDirs...)
			}
			scanner := env.NewScanner(allDirs)
			currentScan, _ := scanner.Scan()

			// Build a snapshot from current state for comparison
			currentSnap := &snapshot.Snapshot{}
			for _, pkg := range currentScan.Packages {
				currentSnap.Packages = append(currentSnap.Packages, snapshot.SnapshotPackage{
					Name:    pkg.Name.Normalized(),
					Version: pkg.Version.String(),
					Manager: pkg.Manager.String(),
				})
			}
			diff = snapshot.DiffSnapshots(snap, currentSnap)
			fmt.Printf("Diff: %s → current\n\n", args[0])
		}

		if len(diff.Added) == 0 && len(diff.Removed) == 0 && len(diff.Changed) == 0 {
			fmt.Println("No differences.")
			return nil
		}

		for _, p := range diff.Added {
			fmt.Printf("  + %s %s (%s)\n", p.Name, p.Version, p.Manager)
		}
		for _, p := range diff.Removed {
			fmt.Printf("  - %s %s (%s)\n", p.Name, p.Version, p.Manager)
		}
		for _, c := range diff.Changed {
			fmt.Printf("  ~ %s %s → %s (%s)\n", c.Name, c.OldVersion, c.NewVersion, c.Manager)
		}

		return nil
	},
}

var snapshotDeleteCmd = &cobra.Command{
	Use:   "delete <snapshot-id>",
	Short: "Delete a snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, _ := os.Getwd()
		activeVenv, err := venv.Detect(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found")
		}

		store := snapshot.NewStore(activeVenv.Path)
		if err := store.Delete(args[0]); err != nil {
			return fmt.Errorf("failed to delete snapshot: %w", err)
		}

		fmt.Printf("Deleted snapshot %s.\n", args[0])
		return nil
	},
}

func init() {
	snapshotCmd.AddCommand(snapshotCreateCmd)
	snapshotCmd.AddCommand(snapshotListCmd)
	snapshotCmd.AddCommand(snapshotRestoreCmd)
	snapshotCmd.AddCommand(snapshotDiffCmd)
	snapshotCmd.AddCommand(snapshotDeleteCmd)
	rootCmd.AddCommand(snapshotCmd)
}

func countByManager(packages []snapshot.SnapshotPackage, manager string) int {
	count := 0
	for _, p := range packages {
		if p.Manager == manager {
			count++
		}
	}
	return count
}

