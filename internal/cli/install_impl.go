package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/resolver"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/spf13/cobra"
)

func init() {
	installCmd.RunE = runInstall
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadFromCwd()
	if err != nil {
		return err
	}

	// Parse requirements from args
	var requirements []pep508.Requirement
	for _, arg := range args {
		req, err := pep508.ParseRequirement(arg)
		if err != nil {
			return fmt.Errorf("invalid requirement %q: %w", arg, err)
		}
		requirements = append(requirements, req)
	}

	// Find the active virtual environment
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found. Run 'fpm init' or 'fpm venv' first")
	}

	// Scan existing installations
	sitePackagesDirs := env.FindSitePackagesDirs([]string{activeVenv.SitePackages})
	scanner := env.NewScanner(sitePackagesDirs)
	scanResult, _ := scanner.Scan()

	// Cross-manager check will happen post-resolution when we know exact versions
	toResolve := requirements

	if len(toResolve) == 0 {
		fmt.Println("Nothing to install.")
		return nil
	}

	// Resolve dependencies
	pypiClient := client.New(client.ClientOptions{
		Indexes:     cfg.Indexes,
		CacheDir:    filepath.Join(cfg.Cache.Dir, "http"),
		Concurrency: cfg.Tool.Concurrency,
	})

	res, err := resolver.New(resolver.ResolverOptions{
		Client:     pypiClient,
		Scanner:    scanResult,
		Immutables: cfg.Immutable.Packages,
	}).Resolve(toResolve)
	if err != nil {
		return fmt.Errorf("resolution failed: %w", err)
	}

	// Download and install
	pkgCache := cache.New(cfg.Cache.Dir)
	pkgCache.Init()
	refTracker := cache.NewRefTracker(pkgCache)

	ctx := context.Background()
	for _, pkg := range res.Packages {
		if pkg.URL == "" {
			fmt.Printf("  Resolved %s %s (no download URL)\n", pkg.Name.Raw(), pkg.Version.String())
			continue
		}

		// Download wheel
		wheelFilename := filepath.Base(pkg.URL)
		wheelPath := filepath.Join(pkgCache.WheelsDir(), wheelFilename)

		if _, err := os.Stat(wheelPath); err != nil {
			fmt.Printf("  Downloading %s %s...\n", pkg.Name.Raw(), pkg.Version.String())
			if err := pypiClient.DownloadWheel(ctx, client.SimpleFile{URL: pkg.URL, Filename: wheelFilename}, wheelPath); err != nil {
				return fmt.Errorf("failed to download %s: %w", pkg.Name.Raw(), err)
			}
		}

		// Store in CAS
		casKey, err := pkgCache.Store(wheelPath)
		if err != nil {
			return fmt.Errorf("failed to store %s in cache: %w", pkg.Name.Raw(), err)
		}

		// Link to site-packages
		casPath, _ := pkgCache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, activeVenv.SitePackages, fs.LinkModeAuto); err != nil {
			return fmt.Errorf("failed to install %s: %w", pkg.Name.Raw(), err)
		}

		// Track reference
		refTracker.AddReference(activeVenv.Path, casKey, pkg.Name.Normalized(), pkg.Version.String())

		fmt.Printf("  Installed %s %s\n", pkg.Name.Raw(), pkg.Version.String())
	}

	// Update pyproject.toml if it exists
	pyproject, err := workspace.ReadPyProjectToml(cwd)
	if err == nil {
		for _, arg := range args {
			pyproject.AddDependency(arg)
		}
		workspace.WritePyProjectToml(cwd, pyproject)
	}

	// Generate/update lockfile
	lf := lock.Generate(res, "")
	lf.Write(filepath.Join(cwd, lock.LockFileName))

	fmt.Printf("\nInstalled %d packages.\n", len(res.Packages))
	return nil
}
