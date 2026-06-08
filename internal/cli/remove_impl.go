package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

func init() {
	removeCmd.RunE = runRemove
}

func runRemove(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	// Determine target site-packages
	var targetSitePackages string
	activeVenv, _ := venv.Detect(cwd)

	if activeVenv != nil && !flagSystem {
		targetSitePackages = activeVenv.SitePackages
	} else if flagSystem {
		finder := python.NewFinder()
		interp, err := finder.FindBest("")
		if err != nil {
			return fmt.Errorf("no Python found")
		}
		targetSitePackages = interp.SitePackages
		if targetSitePackages == "" {
			for _, p := range interp.SysPaths {
				if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
					targetSitePackages = p
					break
				}
			}
		}
	} else {
		// No venv and no --system: try pyproject.toml only (backward compat)
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found. Use --system to remove from system Python")
		}
		for _, name := range args {
			pyproject.RemoveDependency(name)
			fmt.Printf("  \033[32m✓\033[0m Removed %s from dependencies\n", name)
		}
		workspace.WritePyProjectToml(cwd, pyproject)
		fmt.Println("\n  Run 'fpm sync' to apply changes to your environment.")
		return nil
	}

	// Uninstall packages from site-packages
	removed := 0
	for _, name := range args {
		pkgName := types.NewPackageName(name)
		if err := uninstallPackage(targetSitePackages, pkgName); err != nil {
			fmt.Printf("  \033[31m✗\033[0m %s: %v\n", name, err)
			continue
		}
		fmt.Printf("  \033[32m✓\033[0m Removed %s\n", name)
		removed++
	}

	// Also update pyproject.toml if in a project
	if activeVenv != nil && !flagSystem {
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err == nil {
			for _, name := range args {
				pyproject.RemoveDependency(name)
			}
			workspace.WritePyProjectToml(cwd, pyproject)
		}
	}

	if removed > 0 {
		fmt.Printf("\n  %d package(s) removed.\n", removed)
	}
	return nil
}

func uninstallPackage(sitePackages string, name types.PackageName) error {
	// Find the dist-info directory
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", sitePackages, err)
	}

	normalized := strings.ReplaceAll(name.Normalized(), "-", "_")
	var distInfoDir string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}
		entryNorm := strings.ToLower(strings.ReplaceAll(entry.Name(), "-", "_"))
		if strings.HasPrefix(entryNorm, normalized+"_") || strings.HasPrefix(entryNorm, normalized+"-") {
			distInfoDir = filepath.Join(sitePackages, entry.Name())
			break
		}
	}

	if distInfoDir == "" {
		return fmt.Errorf("package not found in %s", sitePackages)
	}

	// Check INSTALLER — only remove fpm-installed packages
	installerPath := filepath.Join(distInfoDir, "INSTALLER")
	if data, err := os.ReadFile(installerPath); err == nil {
		mgr := strings.TrimSpace(string(data))
		if mgr != "fpm" && mgr != "" {
			scanner := env.NewScanner([]string{sitePackages})
			result, _ := scanner.Scan()
			found := result.FindByName(name)
			if len(found) > 0 {
				return fmt.Errorf("installed by %s, not fpm (use %s to remove)", found[0].Manager, found[0].Manager)
			}
		}
	}

	// Read RECORD file to get list of installed files
	recordPath := filepath.Join(distInfoDir, "RECORD")
	files, err := parseRecord(recordPath)
	if err != nil {
		// No RECORD: just remove dist-info and top-level package dir
		os.RemoveAll(distInfoDir)
		pkgDir := filepath.Join(sitePackages, normalized)
		os.RemoveAll(pkgDir)
		return nil
	}

	// Remove all files listed in RECORD
	for _, file := range files {
		path := filepath.Join(sitePackages, file)
		os.Remove(path)
	}

	// Remove empty directories
	os.RemoveAll(distInfoDir)
	pkgDir := filepath.Join(sitePackages, normalized)
	os.RemoveAll(pkgDir)

	return nil
}

func parseRecord(recordPath string) ([]string, error) {
	f, err := os.Open(recordPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) > 0 && parts[0] != "" {
			files = append(files, parts[0])
		}
	}
	return files, nil
}
