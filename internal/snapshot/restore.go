package snapshot

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type RestoreResult struct {
	Restored     int
	Downloaded   int
	Warnings     []string
	Errors       []string
	DriftEntries []DriftLogEntry
}

type DriftLogEntry struct {
	Package    string
	Manager    string
	Expected   string
	Actual     string
	Status     DriftStatus
}

type DriftStatus int

const (
	DriftMatch   DriftStatus = iota // Package matches snapshot
	DriftChanged                     // Version changed in external manager
	DriftMissing                     // Package missing from environment
	DriftNew                         // New package not in snapshot
)

func (d DriftStatus) Symbol() string {
	switch d {
	case DriftMatch:
		return "✓"
	case DriftChanged:
		return "~"
	case DriftMissing:
		return "✗"
	case DriftNew:
		return "+"
	default:
		return "?"
	}
}

type RestoreOptions struct {
	Cache        *cache.Cache
	RefTracker   *cache.RefTracker
	PyPIClient   *client.RegistryClient
	SitePackages string
	EnvPath      string
	AutoDownload bool
}

func Restore(snap *Snapshot, currentScan *env.ScanResult, opts RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{}

	// Separate fpm-managed (restorable) from external (track drift)
	var fpmPackages []SnapshotPackage
	var externalPackages []SnapshotPackage

	for _, pkg := range snap.Packages {
		if pkg.Manager == "fpm" {
			fpmPackages = append(fpmPackages, pkg)
		} else {
			externalPackages = append(externalPackages, pkg)
		}
	}

	// Clean fpm packages from site-packages before restore
	CleanSitePackages(opts.SitePackages)

	// Restore fpm packages via CAS re-linking
	for _, pkg := range fpmPackages {
		restored := false

		// Try CAS first
		if pkg.CASKey != "" {
			casKey := cache.CASKey{Algorithm: "sha256", Digest: pkg.CASKey}
			if opts.Cache.Has(casKey) {
				casPath, err := opts.Cache.Retrieve(casKey)
				if err == nil {
					if err := fs.LinkDir(casPath, opts.SitePackages, fs.LinkModeAuto); err == nil {
						opts.RefTracker.AddReference(opts.EnvPath, casKey, pkg.Name, pkg.Version)
						result.Restored++
						restored = true
					}
				}
			}
		}

		// Fallback: auto-download if CAS entry is missing
		if !restored && opts.AutoDownload && opts.PyPIClient != nil {
			err := redownloadPackage(pkg, opts)
			if err != nil {
				result.Errors = append(result.Errors,
					fmt.Sprintf("%s %s: CAS entry missing and re-download failed: %v", pkg.Name, pkg.Version, err))
			} else {
				result.Downloaded++
				restored = true
			}
		}

		if !restored && pkg.CASKey == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s %s: no CAS key recorded, attempting re-download", pkg.Name, pkg.Version))
			if opts.PyPIClient != nil {
				if err := redownloadPackage(pkg, opts); err == nil {
					result.Downloaded++
				} else {
					result.Errors = append(result.Errors,
						fmt.Sprintf("%s %s: could not restore: %v", pkg.Name, pkg.Version, err))
				}
			}
		}
	}

	// Detailed drift log for external packages
	currentMap := make(map[string]env.InstalledPackage)
	for _, pkg := range currentScan.Packages {
		currentMap[pkg.Name.Normalized()] = pkg
	}

	for _, pkg := range externalPackages {
		current, exists := currentMap[pkg.Name]
		if !exists {
			result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
				Package:  pkg.Name,
				Manager:  pkg.Manager,
				Expected: pkg.Version,
				Actual:   "(missing)",
				Status:   DriftMissing,
			})
		} else if current.Version.String() != pkg.Version {
			result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
				Package:  pkg.Name,
				Manager:  pkg.Manager,
				Expected: pkg.Version,
				Actual:   current.Version.String(),
				Status:   DriftChanged,
			})
		} else {
			result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
				Package:  pkg.Name,
				Manager:  pkg.Manager,
				Expected: pkg.Version,
				Actual:   current.Version.String(),
				Status:   DriftMatch,
			})
		}
	}

	// Detect new packages not in snapshot
	snapshotNames := make(map[string]bool)
	for _, p := range snap.Packages {
		snapshotNames[p.Name] = true
	}
	for _, pkg := range currentScan.Packages {
		if !snapshotNames[pkg.Name.Normalized()] {
			result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
				Package:  pkg.Name.Normalized(),
				Manager:  pkg.Manager.String(),
				Expected: "(not in snapshot)",
				Actual:   pkg.Version.String(),
				Status:   DriftNew,
			})
		}
	}

	// Update current pointer
	store := NewStore(opts.EnvPath)
	store.SetCurrent(snap.ID)

	return result, nil
}

func redownloadPackage(pkg SnapshotPackage, opts RestoreOptions) error {
	ctx := context.Background()

	// Fetch package versions from PyPI
	pkgName := types.NewPackageName(pkg.Name)
	detail, err := opts.PyPIClient.FetchPackageVersions(ctx, pkgName)
	if err != nil {
		return fmt.Errorf("failed to fetch from PyPI: %w", err)
	}

	// Find the exact version's wheel
	for _, file := range detail.Files {
		if !strings.HasSuffix(file.Filename, ".whl") {
			continue
		}
		if !strings.Contains(file.Filename, pkg.Version) {
			continue
		}

		// Download wheel
		wheelPath := filepath.Join(opts.Cache.WheelsDir(), file.Filename)
		if _, err := os.Stat(wheelPath); err != nil {
			if err := opts.PyPIClient.DownloadWheel(ctx, file, wheelPath); err != nil {
				continue
			}
		}

		// Store in CAS
		casKey, err := opts.Cache.Store(wheelPath)
		if err != nil {
			return err
		}

		// Link to site-packages
		casPath, _ := opts.Cache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, opts.SitePackages, fs.LinkModeAuto); err != nil {
			return err
		}

		// Track reference
		opts.RefTracker.AddReference(opts.EnvPath, casKey, pkg.Name, pkg.Version)
		return nil
	}

	return fmt.Errorf("no compatible wheel found for %s %s", pkg.Name, pkg.Version)
}

func (r *RestoreResult) PrintLog() {
	if r.Restored > 0 {
		fmt.Printf("  ✓ Restored %d packages from cache\n", r.Restored)
	}
	if r.Downloaded > 0 {
		fmt.Printf("  ↓ Re-downloaded %d packages (missing from cache)\n", r.Downloaded)
	}

	// Print drift log
	hasDrift := false
	for _, entry := range r.DriftEntries {
		if entry.Status != DriftMatch {
			hasDrift = true
			break
		}
	}

	if hasDrift {
		fmt.Println("\n  External package status:")
		for _, entry := range r.DriftEntries {
			switch entry.Status {
			case DriftChanged:
				fmt.Printf("    %s %s (%s): expected %s, found %s\n",
					entry.Status.Symbol(), entry.Package, entry.Manager, entry.Expected, entry.Actual)
			case DriftMissing:
				fmt.Printf("    %s %s (%s): was %s, now missing\n",
					entry.Status.Symbol(), entry.Package, entry.Manager, entry.Expected)
			case DriftNew:
				fmt.Printf("    %s %s (%s): %s (not in snapshot)\n",
					entry.Status.Symbol(), entry.Package, entry.Manager, entry.Actual)
			}
		}
	}

	if len(r.Errors) > 0 {
		fmt.Println("\n  Errors:")
		for _, e := range r.Errors {
			fmt.Printf("    ✗ %s\n", e)
		}
	}

	if len(r.Warnings) > 0 {
		fmt.Println("\n  Warnings:")
		for _, w := range r.Warnings {
			fmt.Printf("    ⚠ %s\n", w)
		}
	}
}

// CleanSitePackages removes all fpm-installed files before restore.
func CleanSitePackages(sitePackages string) error {
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".dist-info") {
			continue
		}
		installerPath := filepath.Join(sitePackages, name, "INSTALLER")
		data, err := os.ReadFile(installerPath)
		if err != nil {
			continue
		}
		installer := strings.TrimSpace(string(data))
		if installer == "fpm" {
			os.RemoveAll(filepath.Join(sitePackages, name))
			pkgDir := distInfoToPackageDir(name)
			if pkgDir != "" {
				os.RemoveAll(filepath.Join(sitePackages, pkgDir))
			}
		}
	}

	return nil
}

func distInfoToPackageDir(distInfoName string) string {
	// "numpy-1.24.0.dist-info" -> "numpy"
	idx := strings.Index(distInfoName, "-")
	if idx < 0 {
		return ""
	}
	return distInfoName[:idx]
}
