package snapshot

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
	Cache           *cache.Cache
	RefTracker      *cache.RefTracker
	PyPIClient      *client.RegistryClient
	SitePackages    string
	EnvPath         string
	AutoDownload    bool
	RestoreExternal bool
	ProjectDir      string
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
						writeInstallerFile(opts.SitePackages, pkg.Name, pkg.Version)
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

	// Build maps for comparison
	currentMap := make(map[string]env.InstalledPackage)
	for _, pkg := range currentScan.Packages {
		currentMap[pkg.Name.Normalized()] = pkg
	}

	snapshotNames := make(map[string]bool)
	for _, p := range snap.Packages {
		snapshotNames[p.Name] = true
	}

	// Handle external packages
	if opts.RestoreExternal {
		// Remove packages that exist now but were NOT in the snapshot
		for _, pkg := range currentScan.Packages {
			if !snapshotNames[pkg.Name.Normalized()] && pkg.Manager != env.ManagerFpm {
				if err := removePackageFromSitePackages(pkg.Name.Normalized(), opts.SitePackages); err == nil {
					result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
						Package:  pkg.Name.Normalized(),
						Manager:  pkg.Manager.String(),
						Expected: "(not in snapshot)",
						Actual:   pkg.Version.String(),
						Status:   DriftNew,
					})
				}
			}
		}

		// Restore external packages that are missing or have wrong version
		for _, pkg := range externalPackages {
			current, exists := currentMap[pkg.Name]
			if !exists {
				// Missing — reinstall
				if err := reinstallExternalPackage(pkg, opts.SitePackages); err != nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("%s %s (%s): reinstall failed: %v", pkg.Name, pkg.Version, pkg.Manager, err))
				} else {
					result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
						Package: pkg.Name, Manager: pkg.Manager,
						Expected: pkg.Version, Actual: "(restored)", Status: DriftMissing,
					})
				}
			} else if current.Version.String() != pkg.Version {
				// Wrong version — remove and reinstall correct version
				removePackageFromSitePackages(pkg.Name, opts.SitePackages)
				if err := reinstallExternalPackage(pkg, opts.SitePackages); err != nil {
					result.Errors = append(result.Errors,
						fmt.Sprintf("%s: revert %s→%s failed: %v", pkg.Name, current.Version.String(), pkg.Version, err))
				} else {
					result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
						Package: pkg.Name, Manager: pkg.Manager,
						Expected: pkg.Version, Actual: current.Version.String(), Status: DriftChanged,
					})
				}
			} else {
				result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
					Package: pkg.Name, Manager: pkg.Manager,
					Expected: pkg.Version, Actual: current.Version.String(), Status: DriftMatch,
				})
			}
		}
	} else {
		// Legacy: just log drift without restoring
		for _, pkg := range externalPackages {
			current, exists := currentMap[pkg.Name]
			if !exists {
				result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
					Package: pkg.Name, Manager: pkg.Manager,
					Expected: pkg.Version, Actual: "(missing)", Status: DriftMissing,
				})
			} else if current.Version.String() != pkg.Version {
				result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
					Package: pkg.Name, Manager: pkg.Manager,
					Expected: pkg.Version, Actual: current.Version.String(), Status: DriftChanged,
				})
			} else {
				result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
					Package: pkg.Name, Manager: pkg.Manager,
					Expected: pkg.Version, Actual: current.Version.String(), Status: DriftMatch,
				})
			}
		}
		for _, pkg := range currentScan.Packages {
			if !snapshotNames[pkg.Name.Normalized()] {
				result.DriftEntries = append(result.DriftEntries, DriftLogEntry{
					Package: pkg.Name.Normalized(), Manager: pkg.Manager.String(),
					Expected: "(not in snapshot)", Actual: pkg.Version.String(), Status: DriftNew,
				})
			}
		}
	}

	// Restore fpm.toml if snapshot captured it
	if opts.ProjectDir != "" && snap.FpmToml != "" {
		os.WriteFile(filepath.Join(opts.ProjectDir, "fpm.toml"), []byte(snap.FpmToml), 0644)
	} else if opts.ProjectDir != "" && snap.FpmToml == "" {
		// Snapshot had no fpm.toml — remove current one to match snapshot state
		os.Remove(filepath.Join(opts.ProjectDir, "fpm.toml"))
	}

	// Update current pointer
	store := NewStore(opts.EnvPath)
	store.SetCurrent(snap.ID)

	return result, nil
}

func redownloadPackage(pkg SnapshotPackage, opts RestoreOptions) error {
	ctx := context.Background()

	pkgName := types.NewPackageName(pkg.Name)
	detail, err := opts.PyPIClient.FetchPackageVersions(ctx, pkgName)
	if err != nil {
		return fmt.Errorf("failed to fetch from PyPI: %w", err)
	}

	for _, file := range detail.Files {
		if !strings.HasSuffix(file.Filename, ".whl") {
			continue
		}
		if !strings.Contains(file.Filename, pkg.Version) {
			continue
		}

		wheelPath := filepath.Join(opts.Cache.WheelsDir(), file.Filename)
		if _, err := os.Stat(wheelPath); err != nil {
			if err := opts.PyPIClient.DownloadWheel(ctx, file, wheelPath); err != nil {
				continue
			}
		}

		casKey, err := opts.Cache.Store(wheelPath)
		if err != nil {
			return err
		}

		casPath, _ := opts.Cache.Retrieve(casKey)
		if err := fs.LinkDir(casPath, opts.SitePackages, fs.LinkModeAuto); err != nil {
			return err
		}

		// Write INSTALLER file to mark as fpm-managed
		writeInstallerFile(opts.SitePackages, pkg.Name, pkg.Version)

		opts.RefTracker.AddReference(opts.EnvPath, casKey, pkg.Name, pkg.Version)
		return nil
	}

	return fmt.Errorf("no compatible wheel found for %s %s", pkg.Name, pkg.Version)
}

func writeInstallerFile(sitePackages, name, version string) {
	entries, _ := os.ReadDir(sitePackages)
	normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(name), "-", "_"), ".", "_")
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}
		entryName := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(entry.Name()), "-", "_"), ".", "_")
		if strings.HasPrefix(entryName, normalized+"_") {
			installerPath := filepath.Join(sitePackages, entry.Name(), "INSTALLER")
			os.WriteFile(installerPath, []byte("fpm\n"), 0644)
			return
		}
	}
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

func removePackageFromSitePackages(name string, sitePackages string) error {
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return err
	}
	normalized := strings.ReplaceAll(strings.ToLower(name), "-", "_")
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		entryLower := strings.ToLower(entry.Name())
		entryNorm := strings.ReplaceAll(entryLower, "-", "_")
		if strings.HasSuffix(entryLower, ".dist-info") {
			pkg := distInfoToPackageDir(entryNorm)
			if pkg == normalized {
				os.RemoveAll(filepath.Join(sitePackages, entry.Name()))
			}
		} else if entryNorm == normalized {
			os.RemoveAll(filepath.Join(sitePackages, entry.Name()))
		}
	}
	return nil
}

func reinstallExternalPackage(pkg SnapshotPackage, sitePackages string) error {
	cmd := exec.Command("pip", "install", "--target", sitePackages,
		"--no-deps", "--quiet", fmt.Sprintf("%s==%s", pkg.Name, pkg.Version))
	cmd.Env = append(os.Environ(), "PIP_DISABLE_PIP_VERSION_CHECK=1")
	return cmd.Run()
}
