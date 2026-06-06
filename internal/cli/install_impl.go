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
		cfg = config.DefaultConfig()
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

	if len(requirements) == 0 {
		return fmt.Errorf("no packages specified")
	}

	// Find the active virtual environment
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found. Run 'fpm init' or 'fpm venv' first")
	}

	// Ensure site-packages directory exists
	if activeVenv.SitePackages != "" {
		os.MkdirAll(activeVenv.SitePackages, 0755)
	}

	// Scan existing installations
	var scanResult *env.ScanResult
	if activeVenv.SitePackages != "" {
		scanner := env.NewScanner([]string{activeVenv.SitePackages})
		scanResult, _ = scanner.Scan()
	}
	if scanResult == nil {
		scanResult = &env.ScanResult{}
	}

	fmt.Printf("Resolving %d package(s)...\n", len(requirements))

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
	}).Resolve(requirements)
	if err != nil {
		return fmt.Errorf("resolution failed: %w", err)
	}

	if len(res.Packages) == 0 {
		fmt.Println("Nothing to install (no packages resolved).")
		return nil
	}

	// Download and install
	pkgCache := cache.New(cfg.Cache.Dir)
	pkgCache.Init()
	refTracker := cache.NewRefTracker(pkgCache)

	ctx := context.Background()
	installed := 0
	for _, pkg := range res.Packages {
		if pkg.URL == "" {
			fmt.Printf("  %s %s — resolved (no wheel URL available)\n", pkg.Name.Raw(), pkg.Version.String())
			continue
		}

		// Download wheel
		wheelFilename := filepath.Base(pkg.URL)
		wheelPath := filepath.Join(pkgCache.WheelsDir(), wheelFilename)

		if _, err := os.Stat(wheelPath); err != nil {
			fmt.Printf("  Downloading %s %s...\n", pkg.Name.Raw(), pkg.Version.String())
			dlFile := client.SimpleFile{URL: pkg.URL, Filename: wheelFilename}
			if err := pypiClient.DownloadWheel(ctx, dlFile, wheelPath); err != nil {
				fmt.Fprintf(os.Stderr, "  error downloading %s: %v\n", pkg.Name.Raw(), err)
				continue
			}
		} else {
			fmt.Printf("  Using cached %s %s\n", pkg.Name.Raw(), pkg.Version.String())
		}

		// Store in CAS
		casKey, err := pkgCache.Store(wheelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error caching %s: %v\n", pkg.Name.Raw(), err)
			continue
		}

		// Link to site-packages
		casPath, _ := pkgCache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, activeVenv.SitePackages, fs.LinkModeAuto); err != nil {
			fmt.Fprintf(os.Stderr, "  error installing %s: %v\n", pkg.Name.Raw(), err)
			continue
		}

		// Track reference
		refTracker.AddReference(activeVenv.Path, casKey, pkg.Name.Normalized(), pkg.Version.String())
		installed++
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

	fmt.Printf("\nDone: %d package(s) installed.\n", installed)
	return nil
}
