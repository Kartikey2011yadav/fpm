package env

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/pep440"
	"github.com/kartikeyyadav/fpm/pkg/types"
)

type PackageManager int

const (
	ManagerUnknown PackageManager = iota
	ManagerPip
	ManagerConda
	ManagerSystem
	ManagerFpm
	ManagerUv
	ManagerPoetry
	ManagerPdm
)

func (m PackageManager) String() string {
	switch m {
	case ManagerPip:
		return "pip"
	case ManagerConda:
		return "conda"
	case ManagerSystem:
		return "system"
	case ManagerFpm:
		return "fpm"
	case ManagerUv:
		return "uv"
	case ManagerPoetry:
		return "poetry"
	case ManagerPdm:
		return "pdm"
	default:
		return "unknown"
	}
}

type InstalledPackage struct {
	Name     types.PackageName
	Version  pep440.Version
	Location string
	Manager  PackageManager
	DistInfo string
}

type ScanResult struct {
	Packages []InstalledPackage
	byName   map[string][]InstalledPackage
}

func (s *ScanResult) FindByName(name types.PackageName) []InstalledPackage {
	if s.byName == nil {
		s.buildIndex()
	}
	return s.byName[name.Normalized()]
}

func (s *ScanResult) FindExact(name types.PackageName, version pep440.Version) *InstalledPackage {
	for _, pkg := range s.FindByName(name) {
		if pkg.Version.Equal(version) {
			return &pkg
		}
	}
	return nil
}

func (s *ScanResult) buildIndex() {
	s.byName = make(map[string][]InstalledPackage)
	for _, pkg := range s.Packages {
		key := pkg.Name.Normalized()
		s.byName[key] = append(s.byName[key], pkg)
	}
}

type Scanner struct {
	sitePackagesDirs []string
}

func NewScanner(dirs []string) *Scanner {
	return &Scanner{sitePackagesDirs: dirs}
}

func (s *Scanner) Scan() (*ScanResult, error) {
	result := &ScanResult{}

	for _, dir := range s.sitePackagesDirs {
		packages, err := scanSitePackages(dir)
		if err != nil {
			continue
		}
		result.Packages = append(result.Packages, packages...)
	}

	return result, nil
}

func scanSitePackages(dir string) ([]InstalledPackage, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var packages []InstalledPackage

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		distInfoPath := filepath.Join(dir, name)

		if strings.HasSuffix(name, ".dist-info") {
			pkg, err := parseDistInfo(distInfoPath, dir)
			if err != nil {
				continue
			}
			packages = append(packages, *pkg)
		}
	}

	return packages, nil
}

func parseDistInfo(distInfoPath, sitePackagesDir string) (*InstalledPackage, error) {
	metadataPath := filepath.Join(distInfoPath, "METADATA")
	data, err := os.ReadFile(metadataPath)
	if err != nil {
		// Try PKG-INFO (older format)
		metadataPath = filepath.Join(distInfoPath, "PKG-INFO")
		data, err = os.ReadFile(metadataPath)
		if err != nil {
			return nil, err
		}
	}

	name, version := parseMetadataFields(string(data))
	if name == "" {
		return nil, os.ErrNotExist
	}

	ver, err := pep440.Parse(version)
	if err != nil {
		return nil, err
	}

	manager := detectManager(distInfoPath)

	return &InstalledPackage{
		Name:     types.NewPackageName(name),
		Version:  ver,
		Location: sitePackagesDir,
		Manager:  manager,
		DistInfo: distInfoPath,
	}, nil
}

func parseMetadataFields(content string) (name, version string) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			break // End of headers
		}
		if strings.HasPrefix(line, "Name: ") {
			name = strings.TrimPrefix(line, "Name: ")
		} else if strings.HasPrefix(line, "Version: ") {
			version = strings.TrimPrefix(line, "Version: ")
		}
		if name != "" && version != "" {
			break
		}
	}
	return
}

func detectManager(distInfoPath string) PackageManager {
	// Check for INSTALLER file
	installerPath := filepath.Join(distInfoPath, "INSTALLER")
	data, err := os.ReadFile(installerPath)
	if err != nil {
		return detectManagerFromPath(distInfoPath)
	}

	installer := strings.TrimSpace(string(data))
	switch strings.ToLower(installer) {
	case "pip":
		return ManagerPip
	case "conda":
		return ManagerConda
	case "fpm":
		return ManagerFpm
	case "uv":
		return ManagerUv
	case "poetry":
		return ManagerPoetry
	case "pdm":
		return ManagerPdm
	default:
		return detectManagerFromPath(distInfoPath)
	}
}

func detectManagerFromPath(distInfoPath string) PackageManager {
	lower := strings.ToLower(distInfoPath)
	switch {
	case strings.Contains(lower, "conda"):
		return ManagerConda
	case strings.Contains(lower, "/usr/lib/python"):
		return ManagerSystem
	case strings.Contains(lower, "/usr/local/lib/python"):
		return ManagerSystem
	default:
		return ManagerUnknown
	}
}

func FindSitePackagesDirs(sysPaths []string) []string {
	var dirs []string
	seen := make(map[string]bool)

	for _, p := range sysPaths {
		if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
			abs, err := filepath.Abs(p)
			if err != nil {
				abs = p
			}
			if !seen[abs] {
				if info, err := os.Stat(abs); err == nil && info.IsDir() {
					seen[abs] = true
					dirs = append(dirs, abs)
				}
			}
		}
	}

	return dirs
}
