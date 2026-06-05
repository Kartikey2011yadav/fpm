package snapshot

import (
	"fmt"
	"os"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/fs"
)

type RestoreResult struct {
	Restored  int
	Warnings  []string
	Errors    []string
}

type RestoreOptions struct {
	Cache       *cache.Cache
	RefTracker  *cache.RefTracker
	SitePackages string
	EnvPath     string
}

func Restore(snap *Snapshot, currentScan *env.ScanResult, opts RestoreOptions) (*RestoreResult, error) {
	result := &RestoreResult{}

	// Separate fpm-managed (restorable) from external (warn only)
	var fpmPackages []SnapshotPackage
	var externalPackages []SnapshotPackage

	for _, pkg := range snap.Packages {
		if pkg.Manager == "fpm" {
			fpmPackages = append(fpmPackages, pkg)
		} else {
			externalPackages = append(externalPackages, pkg)
		}
	}

	// Restore fpm packages via CAS re-linking
	for _, pkg := range fpmPackages {
		if pkg.CASKey == "" {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s %s: no CAS key recorded, cannot restore precisely", pkg.Name, pkg.Version))
			continue
		}

		casKey := cache.CASKey{Algorithm: "sha256", Digest: pkg.CASKey}

		// Verify CAS entry still exists
		if !opts.Cache.Has(casKey) {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s %s: CAS entry missing (was it garbage collected?)", pkg.Name, pkg.Version))
			continue
		}

		// Link from CAS to site-packages
		casPath, err := opts.Cache.Retrieve(casKey)
		if err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s %s: failed to retrieve from CAS: %v", pkg.Name, pkg.Version, err))
			continue
		}

		if err := fs.LinkDir(casPath, opts.SitePackages, fs.LinkModeAuto); err != nil {
			result.Errors = append(result.Errors,
				fmt.Sprintf("%s %s: failed to link: %v", pkg.Name, pkg.Version, err))
			continue
		}

		// Update reference tracking
		opts.RefTracker.AddReference(opts.EnvPath, casKey, pkg.Name, pkg.Version)
		result.Restored++
	}

	// Check external packages for drift
	currentMap := make(map[string]env.InstalledPackage)
	for _, pkg := range currentScan.Packages {
		currentMap[pkg.Name.Normalized()] = pkg
	}

	for _, pkg := range externalPackages {
		current, exists := currentMap[pkg.Name]
		if !exists {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s %s (%s): was present in snapshot but is now missing",
					pkg.Name, pkg.Version, pkg.Manager))
		} else if current.Version.String() != pkg.Version {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("%s: was %s (%s) in snapshot, now %s — cannot auto-restore external packages",
					pkg.Name, pkg.Version, pkg.Manager, current.Version.String()))
		}
	}

	// Update current pointer
	store := NewStore(opts.EnvPath)
	store.SetCurrent(snap.ID)

	return result, nil
}

// CleanSitePackages removes all fpm-installed files before restore.
// Only removes packages that have INSTALLER=fpm in their dist-info.
func CleanSitePackages(sitePackages string) error {
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return nil
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Only remove dist-info dirs that belong to fpm
		name := entry.Name()
		if !isDistInfoDir(name) {
			continue
		}
		installerPath := fmt.Sprintf("%s/%s/INSTALLER", sitePackages, name)
		data, err := os.ReadFile(installerPath)
		if err != nil {
			continue
		}
		if string(data) == "fpm\n" || string(data) == "fpm" {
			// This is fpm-managed, safe to remove
			os.RemoveAll(fmt.Sprintf("%s/%s", sitePackages, name))
			// Also remove the package directory itself
			pkgDir := distInfoToPackageDir(name)
			if pkgDir != "" {
				os.RemoveAll(fmt.Sprintf("%s/%s", sitePackages, pkgDir))
			}
		}
	}

	return nil
}

func isDistInfoDir(name string) bool {
	return len(name) > 10 && name[len(name)-10:] == ".dist-info"
}

func distInfoToPackageDir(distInfoName string) string {
	// "numpy-1.24.0.dist-info" -> "numpy"
	parts := splitFirst(distInfoName, "-")
	if parts == "" {
		return ""
	}
	return parts
}

func splitFirst(s, sep string) string {
	for i := 0; i < len(s)-len(sep)+1; i++ {
		if s[i:i+len(sep)] == sep {
			return s[:i]
		}
	}
	return s
}
