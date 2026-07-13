package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/client"
	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/internal/journal"
	"github.com/kartikeyyadav/fpm/internal/snapshot"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

var cherryPickCmd = &cobra.Command{
	Use:   "cherry-pick <snapshot-id> <package>",
	Short: "Restore a single package from a snapshot",
	Long: `Install a specific package version from a snapshot without doing a full restore.

Useful for reverting a single package to a known-good version.`,
	GroupID: "environment",
	Args:    cobra.ExactArgs(2),
	RunE:    runCherryPick,
}

func init() {
	rootCmd.AddCommand(cherryPickCmd)
}

func runCherryPick(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	snapshotID := args[0]
	pkgName := types.NewPackageName(args[1])

	activeVenv, _ := venv.Detect(cwd)
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	cfg, _ := config.LoadFromCwd()

	// Load snapshot (try tag resolution first)
	store := snapshot.NewStore(activeVenv.Path)
	tags, _ := loadTags(activeVenv.Path)
	if resolvedID, ok := tags[snapshotID]; ok {
		snapshotID = resolvedID
	}

	snap, err := store.Get(snapshotID)
	if err != nil {
		return fmt.Errorf("snapshot %q not found", snapshotID)
	}

	// Find the package in the snapshot
	var targetPkg *snapshot.SnapshotPackage
	for i := range snap.Packages {
		if types.NewPackageName(snap.Packages[i].Name).Normalized() == pkgName.Normalized() {
			targetPkg = &snap.Packages[i]
			break
		}
	}
	if targetPkg == nil {
		return fmt.Errorf("package %q not found in snapshot %s", args[1], snapshotID)
	}

	// Try CAS first (instant restore)
	pkgCache := cache.New(cfg.Cache.Dir)
	if targetPkg.CASKey != "" {
		casKey := cache.CASKey{Algorithm: "sha256", Digest: targetPkg.CASKey}
		if pkgCache.Has(casKey) {
			casPath, _ := pkgCache.Retrieve(casKey)
			// Remove current version if installed
			uninstallPackage(activeVenv.SitePackages, pkgName)
			// Link from CAS
			if err := fs.LinkDir(casPath, activeVenv.SitePackages, fs.LinkModeAuto); err != nil {
				return fmt.Errorf("failed to link from cache: %w", err)
			}
			journal.Record(activeVenv.Path, journal.OpInstall,
				[]string{targetPkg.Name + "==" + targetPkg.Version},
				"cherry-pick from "+snapshotID)
			fmt.Printf("Cherry-picked %s==%s from snapshot %s (from cache)\n", targetPkg.Name, targetPkg.Version, snapshotID)
			return nil
		}
	}

	// Fallback: download from PyPI
	fmt.Printf("Downloading %s==%s...\n", targetPkg.Name, targetPkg.Version)

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
	detail, err := pypiClient.FetchPackageVersions(ctx, pkgName)
	if err != nil {
		return fmt.Errorf("failed to fetch %s: %w", pkgName.Raw(), err)
	}

	// Find matching version file
	var matchFile *client.SimpleFile
	for i := range detail.Files {
		f := &detail.Files[i]
		if containsVersion(f.Filename, targetPkg.Version) {
			matchFile = f
			break
		}
	}
	if matchFile == nil {
		return fmt.Errorf("version %s of %s not found on PyPI", targetPkg.Version, targetPkg.Name)
	}

	wheelPath := filepath.Join(pkgCache.WheelsDir(), matchFile.Filename)
	if err := pypiClient.DownloadWheel(ctx, *matchFile, wheelPath); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	casKey, err := pkgCache.Store(wheelPath)
	if err != nil {
		return fmt.Errorf("failed to cache: %w", err)
	}

	casPath, _ := pkgCache.Retrieve(casKey)
	uninstallPackage(activeVenv.SitePackages, pkgName)
	if err := fs.LinkDir(casPath, activeVenv.SitePackages, fs.LinkModeAuto); err != nil {
		return fmt.Errorf("failed to install: %w", err)
	}

	journal.Record(activeVenv.Path, journal.OpInstall,
		[]string{targetPkg.Name + "==" + targetPkg.Version},
		"cherry-pick from "+snapshotID)
	fmt.Printf("Cherry-picked %s==%s from snapshot %s\n", targetPkg.Name, targetPkg.Version, snapshotID)
	return nil
}

func containsVersion(filename, version string) bool {
	return len(filename) > 0 && len(version) > 0 &&
		(filepath.Base(filename) != "" &&
			(len(filename) > len(version) &&
				(filename[0] != '.' && filepath.Ext(filename) != ""))) &&
		(contains(filename, version) || contains(filename, version+"-"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
