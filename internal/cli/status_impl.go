package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

func runStatus(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		fmt.Println("No virtual environment detected.")
		fmt.Println("  Run 'fpm init' to create a project or 'fpm venv' to create a venv.")
		return nil
	}

	fmt.Printf("On environment: %s\n", activeVenv.Path)

	// Read lockfile
	lockPath := filepath.Join(cwd, lock.LockFileName)
	lf, err := lock.Read(lockPath)
	if err != nil {
		fmt.Println("No lockfile found.")
		fmt.Println("  Run 'fpm lock' to generate one.")

		// Still show installed packages
		sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
		scanner := env.NewScanner(sitePackagesDirs)
		scanResult, _ := scanner.Scan()
		fpmCount := 0
		for _, pkg := range scanResult.Packages {
			if pkg.Manager == env.ManagerFpm {
				fpmCount++
			}
		}
		if fpmCount > 0 {
			fmt.Printf("\n  %d package(s) installed (no lockfile to compare against)\n", fpmCount)
		}
		return nil
	}

	// Scan current environment
	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()

	// Build maps for comparison
	lockVersions := make(map[string]string) // normalized name → version
	for _, pkg := range lf.Packages {
		lockVersions[types.NewPackageName(pkg.Name).Normalized()] = pkg.Version
	}

	installedVersions := make(map[string]string) // normalized name → version
	for _, pkg := range scanResult.Packages {
		if pkg.Manager == env.ManagerFpm {
			installedVersions[pkg.Name.Normalized()] = pkg.Version.String()
		}
	}

	// Compute diffs
	var added []statusEntry   // installed but not in lockfile
	var removed []statusEntry // in lockfile but not installed
	var changed []statusEntry // version differs

	for name, installedVer := range installedVersions {
		lockVer, inLock := lockVersions[name]
		if !inLock {
			added = append(added, statusEntry{name: name, version: installedVer})
		} else if installedVer != lockVer {
			changed = append(changed, statusEntry{name: name, version: installedVer, lockVersion: lockVer})
		}
	}

	for name, lockVer := range lockVersions {
		if _, installed := installedVersions[name]; !installed {
			removed = append(removed, statusEntry{name: name, version: lockVer})
		}
	}

	// Display
	if len(added) == 0 && len(removed) == 0 && len(changed) == 0 {
		fmt.Printf("Lockfile: %s\n", lock.LockFileName)
		fmt.Println("\n  Environment matches lockfile. Nothing to do.")
		return nil
	}

	fmt.Printf("Lockfile: %s\n\n", lock.LockFileName)
	dirty := false

	if len(added) > 0 {
		dirty = true
		fmt.Println("  Packages not in lockfile:")
		fmt.Println("    (use \"fpm lock\" to update lockfile)")
		fmt.Println()
		for _, e := range added {
			fmt.Printf("      \033[32madded:\033[0m   %s %s\n", e.name, e.version)
		}
		fmt.Println()
	}

	if len(changed) > 0 {
		dirty = true
		fmt.Println("  Version changes:")
		fmt.Println("    (use \"fpm lock\" to update, or \"fpm sync\" to revert)")
		fmt.Println()
		for _, e := range changed {
			fmt.Printf("      \033[33mchanged:\033[0m %s %s → %s\n", e.name, e.lockVersion, e.version)
		}
		fmt.Println()
	}

	if len(removed) > 0 {
		dirty = true
		fmt.Println("  Lockfile packages not installed:")
		fmt.Println("    (use \"fpm sync\" to install)")
		fmt.Println()
		for _, e := range removed {
			fmt.Printf("      \033[31mmissing:\033[0m %s %s\n", e.name, e.version)
		}
		fmt.Println()
	}

	if dirty {
		cmd.SilenceErrors = true
		return fmt.Errorf("")
	}
	return nil
}

type statusEntry struct {
	name        string
	version     string
	lockVersion string
}
