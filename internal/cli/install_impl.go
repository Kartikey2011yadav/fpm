package cli

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/depgraph"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/journal"
	"github.com/kartikeyyadav/fpm/internal/lock"
	fpmlog "github.com/kartikeyyadav/fpm/internal/log"
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

	fpmlog.Info("install: %d package(s) requested", len(requirements))
	for _, r := range requirements {
		fpmlog.Debug("  requirement: %s", r.Name.Raw())
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

	// Pre-flight: check write permissions before attempting install
	if err := fs.CheckWritable(targetSitePackages); err != nil {
		var permErr *fs.PermissionError
		if errors.As(err, &permErr) {
			return fpmErrors.WithHint(fpmErrors.New(permErr.Error()), permErr.Hint())
		}
		return err
	}

	// Acquire venv-level lock to prevent concurrent install corruption
	venvLock, lockErr := fs.LockFile(filepath.Join(targetSitePackages, ".fpm"))
	if lockErr != nil {
		if fs.IsPermissionError(lockErr) {
			return fpmErrors.WithHint(
				fpmErrors.New("permission denied: cannot lock environment"),
				"Try: sudo fpm install ...\n          Or use a virtual environment (fpm venv).")
		}
		return fmt.Errorf("could not acquire environment lock (another fpm process may be running): %w", lockErr)
	}
	defer fs.UnlockFile(venvLock)

	// Ensure site-packages directory exists
	os.MkdirAll(targetSitePackages, 0755)

	// Scan existing installations
	scanner := env.NewScanner([]string{targetSitePackages})
	scanResult, _ := scanner.Scan()
	if scanResult == nil {
		scanResult = &env.ScanResult{}
	}

	_ = targetBinDir

	resolveStart := time.Now()
	fmt.Printf("\033[1m  Resolving %d package(s)...\033[0m", len(requirements))

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
		fmt.Println()
		return wrapResolutionError(err, args)
	}

	resolveElapsed := time.Since(resolveStart)
	fmt.Printf(" \033[32mdone\033[0m \033[2m(%dms)\033[0m\n", resolveElapsed.Milliseconds())

	if len(res.Packages) == 0 {
		fmt.Println("  Nothing to install (no packages resolved).")
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

	// Download and install (transactional: stage first, then swap atomically)
	pkgCache := cache.New(cfg.Cache.Dir)
	pkgCache.Init()
	refTracker := cache.NewRefTracker(pkgCache)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	startTime := time.Now()

	// Create a temporary staging directory for atomic install
	stagingSuffix := strconv.Itoa(os.Getpid()) + "-" + strconv.FormatInt(rand.Int63(), 36)
	stagingDir := filepath.Join(os.TempDir(), "fpm-install-"+stagingSuffix)
	if err := os.MkdirAll(stagingDir, 0755); err != nil {
		return fmt.Errorf("failed to create staging directory: %w", err)
	}
	defer os.RemoveAll(stagingDir) // cleanup on any exit path

	// Track successfully staged packages for lockfile accuracy
	type stagedPkg struct {
		pkg    resolver.ResolvedPackage
		casKey cache.CASKey
	}
	var stagedPackages []stagedPkg

	fmt.Println()
	for _, pkg := range res.Packages {
		if ctx.Err() != nil {
			fmt.Printf("\n  Interrupted. Cleaning up...\n")
			os.RemoveAll(stagingDir)
			return fmt.Errorf("install interrupted by signal")
		}

		if pkg.URL == "" {
			continue
		}

		// Download wheel
		wheelFilename := filepath.Base(pkg.URL)
		wheelPath := filepath.Join(pkgCache.WheelsDir(), wheelFilename)

		if _, err := os.Stat(wheelPath); err != nil {
			// Download needed — show progress
			dlStart := time.Now()
			fmt.Printf("  \033[34m↓\033[0m \033[1m%-25s\033[0m \033[2m%s\033[0m", pkg.Name.Raw(), pkg.Version.String())
			dlFile := client.SimpleFile{URL: pkg.URL, Filename: wheelFilename}
			if err := pypiClient.DownloadWheel(ctx, dlFile, wheelPath); err != nil {
				fmt.Printf(" \033[31m✗ failed\033[0m\n")
				os.RemoveAll(stagingDir)
				return fmt.Errorf("failed to download %s: %w", pkg.Name.Raw(), err)
			}
			elapsed := time.Since(dlStart)
			fmt.Printf(" \033[32m✓\033[0m \033[2m%dms\033[0m\n", elapsed.Milliseconds())

			// Verify SHA256 hash after download
			if pkg.Hash != "" {
				fileHash, hashErr := cache.HashFile(wheelPath)
				if hashErr != nil {
					os.Remove(wheelPath)
					os.RemoveAll(stagingDir)
					return fmt.Errorf("hash computation error for %s: %w", pkg.Name.Raw(), hashErr)
				}
				if fileHash != pkg.Hash {
					os.Remove(wheelPath)
					os.RemoveAll(stagingDir)
					return fmt.Errorf("hash mismatch for %s: expected %s, got %s", pkg.Name.Raw(), pkg.Hash, fileHash)
				}
			}
		} else {
			// Already cached — verify hash integrity
			if pkg.Hash != "" {
				fileHash, hashErr := cache.HashFile(wheelPath)
				if hashErr != nil {
					os.Remove(wheelPath)
					os.RemoveAll(stagingDir)
					return fmt.Errorf("hash computation error for cached %s: %w", pkg.Name.Raw(), hashErr)
				}
				if fileHash != pkg.Hash {
					os.Remove(wheelPath)
					os.RemoveAll(stagingDir)
					return fmt.Errorf("hash mismatch for cached %s: expected %s, got %s", pkg.Name.Raw(), pkg.Hash, fileHash)
				}
			}
			fmt.Printf("  \033[32m✓\033[0m \033[1m%-25s\033[0m \033[36m%s\033[0m \033[2m(cached)\033[0m\n", pkg.Name.Raw(), pkg.Version.String())
		}

		// Store in CAS
		casKey, err := pkgCache.Store(wheelPath)
		if err != nil {
			os.RemoveAll(stagingDir)
			return fmt.Errorf("failed to store %s in cache: %w", pkg.Name.Raw(), err)
		}

		// Link CAS -> staging directory (NOT directly to site-packages)
		casPath, _ := pkgCache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, stagingDir, fs.LinkModeAuto); err != nil {
			os.RemoveAll(stagingDir)
			return fmt.Errorf("failed to stage %s: %w", pkg.Name.Raw(), err)
		}

		stagedPackages = append(stagedPackages, stagedPkg{pkg: pkg, casKey: casKey})
	}

	// All packages staged successfully — now atomically swap into site-packages
	stagingEntries, err := os.ReadDir(stagingDir)
	if err != nil {
		os.RemoveAll(stagingDir)
		return fmt.Errorf("failed to read staging directory: %w", err)
	}

	for _, entry := range stagingEntries {
		src := filepath.Join(stagingDir, entry.Name())
		dst := filepath.Join(targetSitePackages, entry.Name())
		// Remove existing entry in site-packages to allow clean link
		os.RemoveAll(dst)
		if entry.IsDir() {
			if err := fs.LinkDir(src, dst, fs.LinkModeAuto); err != nil {
				os.RemoveAll(stagingDir)
				return fmt.Errorf("failed to install %s into site-packages: %w", entry.Name(), err)
			}
		} else {
			if err := fs.LinkFile(src, dst, fs.LinkModeAuto); err != nil {
				os.RemoveAll(stagingDir)
				return fmt.Errorf("failed to install %s into site-packages: %w", entry.Name(), err)
			}
		}
	}

	// Swap succeeded — write INSTALLER markers
	for _, sp := range stagedPackages {
		pkgNorm := strings.ReplaceAll(sp.pkg.Name.Normalized(), "-", "_")
		spEntries, _ := os.ReadDir(targetSitePackages)
		for _, entry := range spEntries {
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
		refTracker.AddReference(envPath, sp.casKey, sp.pkg.Name.Normalized(), sp.pkg.Version.String())
	}

	installed := len(stagedPackages)
	totalTime := time.Since(startTime)

	// Update dependency graph — track requested vs transitive
	graph := depgraph.Load(envPath)
	requestedNames := make(map[string]bool)
	for _, req := range requirements {
		requestedNames[req.Name.Normalized()] = true
	}
	for _, sp := range stagedPackages {
		var deps []string
		for _, d := range sp.pkg.Deps {
			deps = append(deps, d.Name.Normalized())
		}
		if requestedNames[sp.pkg.Name.Normalized()] {
			graph.AddRequested(sp.pkg.Name.Normalized(), sp.pkg.Version.String(), deps)
		} else {
			graph.AddTransitive(sp.pkg.Name.Normalized(), sp.pkg.Version.String(), deps)
		}
	}
	graph.Save(envPath)

	// Update pyproject.toml and lockfile only after successful installation
	if activeVenv != nil && !flagSystem && installed > 0 {
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err == nil {
			for _, arg := range args {
				pyproject.AddDependency(arg)
			}
			workspace.WritePyProjectToml(cwd, pyproject)
		}

		// Generate lockfile based only on successfully installed packages
		installedRes := &resolver.Resolution{
			Packages: make([]resolver.ResolvedPackage, 0, len(stagedPackages)),
		}
		for _, sp := range stagedPackages {
			installedRes.Packages = append(installedRes.Packages, sp.pkg)
		}
		lf := lock.Generate(installedRes, "")
		lf.Write(filepath.Join(cwd, lock.LockFileName))
	}

	// Record operation in journal
	var installedNames []string
	for _, sp := range stagedPackages {
		installedNames = append(installedNames, sp.pkg.Name.Normalized()+"=="+sp.pkg.Version.String())
	}
	journal.Record(envPath, journal.OpInstall, installedNames, "")

	fmt.Printf("\n  \033[32m✓ Installed %d package(s)\033[0m \033[2min %dms\033[0m\n", installed, totalTime.Milliseconds())
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
