package installer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/cache"
	"github.com/kartikeyyadav/fpm/internal/fs"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type InstallPlan struct {
	Package      types.PackageName
	Version      string
	CASKey       cache.CASKey
	TargetDir    string
	LinkMode     fs.LinkMode
	ConsoleScripts []ConsoleScript
}

type ConsoleScript struct {
	Name   string
	Module string
	Func   string
}

type InstallResult struct {
	InstalledFiles int
	DistInfoDir    string
}

func Install(plan InstallPlan, pkgCache *cache.Cache) (*InstallResult, error) {
	casPath, err := pkgCache.Retrieve(plan.CASKey)
	if err != nil {
		return nil, fmt.Errorf("CAS entry not found: %w", err)
	}

	result := &InstallResult{}

	// Walk CAS directory and link each file into target site-packages
	err = filepath.Walk(casPath, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(casPath, path)
		if err != nil {
			return err
		}
		dst := filepath.Join(plan.TargetDir, rel)

		if info.IsDir() {
			return os.MkdirAll(dst, info.Mode())
		}

		// Track dist-info directory
		if strings.Contains(rel, ".dist-info") && result.DistInfoDir == "" {
			parts := strings.SplitN(rel, string(os.PathSeparator), 2)
			result.DistInfoDir = filepath.Join(plan.TargetDir, parts[0])
		}

		if err := fs.LinkFile(path, dst, plan.LinkMode); err != nil {
			return fmt.Errorf("failed to link %s: %w", rel, err)
		}
		result.InstalledFiles++
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}

	// Write INSTALLER file to mark as fpm-managed
	if result.DistInfoDir != "" {
		installerPath := filepath.Join(result.DistInfoDir, "INSTALLER")
		os.WriteFile(installerPath, []byte("fpm\n"), 0644)
	}

	return result, nil
}

func WriteInstallerMarker(distInfoDir string) error {
	return os.WriteFile(filepath.Join(distInfoDir, "INSTALLER"), []byte("fpm\n"), 0644)
}
