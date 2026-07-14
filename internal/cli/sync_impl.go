package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

func init() {
	syncCmd.RunE = runSync
}

func runSync(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	cfg, _ := config.LoadFromCwd()

	// Read lockfile
	lockPath := filepath.Join(cwd, lock.LockFileName)
	lf, err := lock.Read(lockPath)
	if err != nil {
		return fmt.Errorf("no lockfile found. Run 'fpm lock' first")
	}

	// Find venv
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found.\nRun 'fpm venv' to create one, or 'fpm init' to start a new project")
	}

	// Pre-flight: check write permissions
	if err := fs.CheckWritable(activeVenv.SitePackages); err != nil {
		var permErr *fs.PermissionError
		if errors.As(err, &permErr) {
			return fmt.Errorf("%s\n\n  hint: %s", permErr.Error(), permErr.Hint())
		}
		return err
	}

	// Acquire venv-level lock to prevent concurrent sync corruption
	venvLock, lockErr := fs.LockFile(filepath.Join(activeVenv.SitePackages, ".fpm"))
	if lockErr != nil {
		if fs.IsPermissionError(lockErr) {
			return fmt.Errorf("permission denied: cannot lock environment\n\n  hint: Try: sudo fpm sync")
		}
		return fmt.Errorf("could not acquire environment lock (another fpm process may be running): %w", lockErr)
	}
	defer fs.UnlockFile(venvLock)

	// Scan current environment
	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()

	// Compare with lockfile
	lockVersions := lf.Versions()
	var toInstall []lock.LockedPackage

	for _, pkg := range lf.Packages {
		version, err := pep440.Parse(pkg.Version)
		if err != nil {
			continue
		}

		installed := scanResult.FindExact(types.NewPackageName(pkg.Name), version)
		if installed == nil {
			toInstall = append(toInstall, pkg)
		}
	}

	// Find fpm-managed packages not in lockfile
	var toRemove []env.InstalledPackage
	for _, installed := range scanResult.Packages {
		if installed.Manager != env.ManagerFpm {
			continue
		}
		if _, exists := lockVersions[installed.Name.Normalized()]; !exists {
			toRemove = append(toRemove, installed)
		}
	}

	if len(toInstall) == 0 && len(toRemove) == 0 {
		fmt.Println("Everything is up to date.")
		return nil
	}

	// Install missing packages
	if len(toInstall) > 0 {
		pkgCache := cache.New(cfg.Cache.Dir)
		pkgCache.Init()
		refTracker := cache.NewRefTracker(pkgCache)

		networkCfg := cfg.Network
		if len(flagAllowInsecureHost) > 0 {
			networkCfg.AllowInsecureHost = append(networkCfg.AllowInsecureHost, flagAllowInsecureHost...)
		}
		pypiClient := client.New(client.ClientOptions{
			Indexes:     cfg.Indexes,
			CacheDir:    filepath.Join(cfg.Cache.Dir, "http"),
			Concurrency: cfg.Tool.Concurrency,
			Network:     networkCfg,
		})

		ctx := context.Background()
		for _, pkg := range toInstall {
			url := pkg.Source.URL
			if url == "" {
				fmt.Printf("  Skip %s %s (no source URL)\n", pkg.Name, pkg.Version)
				continue
			}

			wheelFilename := filepath.Base(url)
			wheelPath := filepath.Join(pkgCache.WheelsDir(), wheelFilename)

			if _, err := os.Stat(wheelPath); err != nil {
				fmt.Printf("  Downloading %s %s...\n", pkg.Name, pkg.Version)
				file := client.SimpleFile{URL: url, Filename: wheelFilename}
				if err := pypiClient.DownloadWheel(ctx, file, wheelPath); err != nil {
					fmt.Printf("  Warning: failed to download %s: %v\n", pkg.Name, err)
					continue
				}
			}

			casKey, err := pkgCache.Store(wheelPath)
			if err != nil {
				fmt.Printf("  Warning: failed to cache %s: %v\n", pkg.Name, err)
				continue
			}

			casPath, _ := pkgCache.Retrieve(casKey)
			if err := fs.LinkDir(casPath, activeVenv.SitePackages, fs.LinkModeAuto); err != nil {
				fmt.Printf("  Warning: failed to install %s: %v\n", pkg.Name, err)
				continue
			}

			refTracker.AddReference(activeVenv.Path, casKey, pkg.Name, pkg.Version)
			fmt.Printf("  Installed %s %s\n", pkg.Name, pkg.Version)
		}
	}

	for _, pkg := range toRemove {
		pkgName := types.NewPackageName(pkg.Name.Raw())
		if err := uninstallPackage(activeVenv.SitePackages, pkgName); err != nil {
			fmt.Printf("  Warning: failed to remove %s: %v\n", pkg.Name.Raw(), err)
			continue
		}
		fmt.Printf("  Removed %s %s\n", pkg.Name.Raw(), pkg.Version.String())
	}

	fmt.Printf("\nSynced: %d installed, %d removed.\n", len(toInstall), len(toRemove))
	return nil
}
