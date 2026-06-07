package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/resolver"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	fpmErrors "github.com/kartikeyyadav/fpm/pkg/errors"
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

	// Determine target: venv (local) or system (global)
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)

	var targetSitePackages string
	var targetBinDir string
	var envPath string

	if activeVenv != nil && !flagSystem {
		// Install into local venv
		targetSitePackages = activeVenv.SitePackages
		targetBinDir = activeVenv.BinDir
		envPath = activeVenv.Path
	} else if activeVenv == nil && !flagSystem {
		// No venv and no --system flag: error (like uv)
		return &fpmErrors.FpmError{
			Message:  "No virtual environment found",
			ExitCode: fpmErrors.ExitFailure,
			Hint:     "Run `fpm venv` to create an environment, or pass `--system` (`-s`) to\ninstall into the system Python.",
		}
	} else {
		// Install globally (system site-packages) — like pip without a venv
		finder := python.NewFinder()
		interp, findErr := finder.FindBest("")
		if findErr != nil {
			return fmt.Errorf("no Python found. Install Python or create a venv with 'fpm init'")
		}
		targetSitePackages = interp.SitePackages
		if targetSitePackages == "" && len(interp.SysPaths) > 0 {
			// Use first writable site-packages from sys.path
			for _, p := range interp.SysPaths {
				if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
					targetSitePackages = p
					break
				}
			}
		}
		targetBinDir = interp.BinDir()
		envPath = "global:" + interp.Path
		if flagSystem && activeVenv != nil {
			fmt.Printf("  \033[33m●\033[0m Installing to system Python (--system flag)\n")
			fmt.Printf("    \033[2mTarget: %s\033[0m\n\n", targetSitePackages)
		}
	}

	if targetSitePackages == "" {
		return fmt.Errorf("cannot determine site-packages directory. Run 'fpm init' to create a project")
	}

	// Ensure site-packages directory exists
	os.MkdirAll(targetSitePackages, 0755)

	// Scan existing installations
	scanner := env.NewScanner([]string{targetSitePackages})
	scanResult, _ := scanner.Scan()
	if scanResult == nil {
		scanResult = &env.ScanResult{}
	}

	_ = targetBinDir

	fmt.Printf("\033[1m⠋ Resolving %d package(s)...\033[0m\n\n", len(requirements))

	// Resolve dependencies
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

	res, err := resolver.New(resolver.ResolverOptions{
		Client:     pypiClient,
		Scanner:    scanResult,
		Immutables: cfg.Immutable.Packages,
	}).Resolve(requirements)
	if err != nil {
		return wrapResolutionError(err, args)
	}

	if len(res.Packages) == 0 {
		fmt.Println("Nothing to install (no packages resolved).")
		return nil
	}

	// Cross-manager conflict check (ask/install/skip policy)
	crossChecker := env.NewCrossManagerChecker(scanResult, env.CrossManagerPolicy(cfg.Tool.CrossManagerPolicy))
	var filteredPackages []resolver.ResolvedPackage
	for _, pkg := range res.Packages {
		result := crossChecker.Check(pkg.Name, pkg.Version)
		switch result.Action {
		case env.ActionSkip:
			fmt.Printf("  \033[33m●\033[0m %s\n", result.Message)
			continue
		case env.ActionAbort:
			return fmt.Errorf("%s", result.Message)
		case env.ActionInstall:
			if result.Message != "" {
				fmt.Printf("  \033[33m●\033[0m %s\n", result.Message)
			}
		}
		filteredPackages = append(filteredPackages, pkg)
	}
	res.Packages = filteredPackages

	if len(res.Packages) == 0 {
		fmt.Println("Nothing to install (all packages already available).")
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
			fmt.Printf("  \033[34m↓\033[0m \033[1m%s\033[0m \033[2m%s\033[0m\n", pkg.Name.Raw(), pkg.Version.String())
			dlFile := client.SimpleFile{URL: pkg.URL, Filename: wheelFilename}
			if err := pypiClient.DownloadWheel(ctx, dlFile, wheelPath); err != nil {
				fmt.Fprintf(os.Stderr, "  \033[31m✗\033[0m %s: %v\n", pkg.Name.Raw(), err)
				continue
			}
		} else {
			fmt.Printf("  \033[33m●\033[0m \033[1m%s\033[0m \033[2m%s (cached)\033[0m\n", pkg.Name.Raw(), pkg.Version.String())
		}

		// Store in CAS
		casKey, err := pkgCache.Store(wheelPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "  error caching %s: %v\n", pkg.Name.Raw(), err)
			continue
		}

		// Link to site-packages
		casPath, _ := pkgCache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, targetSitePackages, fs.LinkModeAuto); err != nil {
			fmt.Fprintf(os.Stderr, "  error installing %s: %v\n", pkg.Name.Raw(), err)
			continue
		}

		// Write INSTALLER marker so fpm is recognized as the manager
		entries, _ := os.ReadDir(targetSitePackages)
		pkgNorm := strings.ReplaceAll(pkg.Name.Normalized(), "-", "_")
		for _, entry := range entries {
			name := entry.Name()
			if !strings.HasSuffix(name, ".dist-info") {
				continue
			}
			entryNorm := strings.ToLower(strings.ReplaceAll(name, "-", "_"))
			if strings.HasPrefix(entryNorm, pkgNorm+"_") || strings.HasPrefix(entryNorm, pkgNorm+"-") {
				installerPath := filepath.Join(targetSitePackages, name, "INSTALLER")
				os.WriteFile(installerPath, []byte("fpm\n"), 0644)
				break
			}
		}

		// Track reference
		refTracker.AddReference(envPath, casKey, pkg.Name.Normalized(), pkg.Version.String())
		installed++
		fmt.Printf("  \033[32m✓\033[0m \033[1m%s\033[0m \033[36m%s\033[0m\n", pkg.Name.Raw(), pkg.Version.String())
	}

	// Update pyproject.toml and lockfile only when in a project (not global installs)
	if activeVenv != nil && !flagSystem {
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err == nil {
			for _, arg := range args {
				pyproject.AddDependency(arg)
			}
			workspace.WritePyProjectToml(cwd, pyproject)
		}

		lf := lock.Generate(res, "")
		lf.Write(filepath.Join(cwd, lock.LockFileName))
	}

	fmt.Printf("\n\033[32m🚀 Done: %d package(s) installed.\033[0m\n", installed)
	return nil
}

func wrapResolutionError(err error, args []string) error {
	msg := err.Error()

	// Detect 404 (package not found)
	if strings.Contains(msg, "HTTP 404") {
		pkgName := ""
		if len(args) == 1 {
			pkgName = args[0]
		}

		fpmErr := &fpmErrors.FpmError{
			Message:  fmt.Sprintf("Package %q not found on PyPI", pkgName),
			Cause:    err,
			ExitCode: fpmErrors.ExitFailure,
		}

		if pkgName != "" {
			if suggestion := client.SuggestPackage(pkgName); suggestion != "" {
				fpmErr.Hint = fmt.Sprintf("Did you mean %q?\nRun: fpm install %s", suggestion, suggestion)
			} else {
				fpmErr.Hint = "Check the package name at https://pypi.org"
			}
		}
		return fpmErr
	}

	// Detect TLS/certificate errors
	if strings.Contains(msg, "x509") || strings.Contains(msg, "certificate") || strings.Contains(msg, "tls:") {
		return &fpmErrors.FpmError{
			Message:  "TLS certificate verification failed",
			Cause:    err,
			ExitCode: fpmErrors.ExitFailure,
			Hint:     "Your network may be intercepting HTTPS connections (VPN/proxy).\nTry: fpm install --allow-insecure-host pypi.org --allow-insecure-host files.pythonhosted.org <package>\n Or: export FPM_INSECURE=1",
		}
	}

	// Detect network errors
	if strings.Contains(msg, "dial tcp") || strings.Contains(msg, "no such host") || strings.Contains(msg, "connection refused") {
		return &fpmErrors.FpmError{
			Message:  "Network connection failed",
			Cause:    err,
			ExitCode: fpmErrors.ExitFailure,
			Hint:     "Check your internet connection and proxy settings.",
		}
	}

	// Default: just wrap with cleaner message
	return &fpmErrors.FpmError{
		Message:  "Dependency resolution failed",
		Cause:    err,
		ExitCode: fpmErrors.ExitFailure,
	}
}
