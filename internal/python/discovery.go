package python

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/pep440"
)

type Finder struct {
	managedDir string
}

func NewFinder() *Finder {
	return &Finder{
		managedDir: config.PythonInstallDir(),
	}
}

func (f *Finder) FindAll() ([]*Interpreter, error) {
	var interpreters []*Interpreter

	// 1. Check fpm-managed Pythons
	managed, _ := f.findManaged()
	interpreters = append(interpreters, managed...)

	// 2. Search PATH
	pathFound, _ := f.findInPath()
	interpreters = append(interpreters, pathFound...)

	// Deduplicate by resolved path
	seen := make(map[string]bool)
	var unique []*Interpreter
	for _, interp := range interpreters {
		resolved, err := filepath.EvalSymlinks(interp.Path)
		if err != nil {
			resolved = interp.Path
		}
		if !seen[resolved] {
			seen[resolved] = true
			unique = append(unique, interp)
		}
	}

	// Sort by version descending
	sort.Slice(unique, func(i, j int) bool {
		return pep440.Compare(unique[i].Version, unique[j].Version) > 0
	})

	return unique, nil
}

func (f *Finder) FindBest(constraint string) (*Interpreter, error) {
	all, err := f.FindAll()
	if err != nil {
		return nil, err
	}

	if constraint == "" {
		if len(all) > 0 {
			return all[0], nil
		}
		return nil, fmt.Errorf("no Python interpreter found")
	}

	specs, err := pep440.ParseSpecifiers(constraint)
	if err != nil {
		// Try as bare version prefix like "3.11"
		ver, verErr := pep440.Parse(constraint)
		if verErr != nil {
			return nil, fmt.Errorf("invalid version constraint: %q", constraint)
		}
		for _, interp := range all {
			if interp.Version.Major() == ver.Major() && interp.Version.Minor() == ver.Minor() {
				return interp, nil
			}
		}
		return nil, fmt.Errorf("no Python %s found", constraint)
	}

	for _, interp := range all {
		if specs.Contains(interp.Version) {
			return interp, nil
		}
	}
	return nil, fmt.Errorf("no Python matching %q found", constraint)
}

func (f *Finder) findManaged() ([]*Interpreter, error) {
	entries, err := os.ReadDir(f.managedDir)
	if err != nil {
		return nil, nil
	}

	var interpreters []*Interpreter
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		// Directory names like "cpython-3.11.8"
		name := entry.Name()
		path := filepath.Join(f.managedDir, name, pythonBinaryRelPath())
		if _, err := os.Stat(path); err != nil {
			continue
		}
		interp, err := Probe(path)
		if err != nil {
			continue
		}
		interp.IsManaged = true
		interpreters = append(interpreters, interp)
	}
	return interpreters, nil
}

func (f *Finder) findInPath() ([]*Interpreter, error) {
	pathEnv := os.Getenv("PATH")
	dirs := filepath.SplitList(pathEnv)

	candidates := pythonCandidateNames()
	var interpreters []*Interpreter

	for _, dir := range dirs {
		for _, name := range candidates {
			path := filepath.Join(dir, name)
			if _, err := os.Stat(path); err != nil {
				continue
			}
			interp, err := Probe(path)
			if err != nil {
				continue
			}
			interpreters = append(interpreters, interp)
		}
	}
	return interpreters, nil
}

func pythonCandidateNames() []string {
	if runtime.GOOS == "windows" {
		return []string{"python.exe", "python3.exe"}
	}
	return []string{
		"python3.13", "python3.12", "python3.11", "python3.10", "python3.9",
		"python3", "python",
	}
}

func pythonBinaryRelPath() string {
	if runtime.GOOS == "windows" {
		return filepath.Join("python.exe")
	}
	return filepath.Join("bin", "python3")
}

type probeResult struct {
	Version        string   `json:"version"`
	Prefix         string   `json:"prefix"`
	SitePackages   string   `json:"site_packages"`
	SysPaths       []string `json:"sys_paths"`
	IsVenv         bool     `json:"is_venv"`
	Implementation string   `json:"implementation"`
	PureLib        string   `json:"purelib"`
	PlatLib        string   `json:"platlib"`
	Scripts        string   `json:"scripts"`
}

const probeScript = `
import json, sys, sysconfig
paths = sysconfig.get_paths()
result = {
    "version": "%d.%d.%d" % sys.version_info[:3],
    "prefix": sys.prefix,
    "site_packages": paths.get("purelib", ""),
    "sys_paths": sys.path,
    "is_venv": sys.prefix != sys.base_prefix,
    "implementation": sys.implementation.name,
    "purelib": paths.get("purelib", ""),
    "platlib": paths.get("platlib", ""),
    "scripts": paths.get("scripts", ""),
}
print(json.dumps(result))
`

func Probe(pythonPath string) (*Interpreter, error) {
	if cached := loadCachedProbe(pythonPath); cached != nil {
		return cached, nil
	}

	interp, err := probeRaw(pythonPath)
	if err != nil {
		return nil, err
	}

	saveCachedProbe(pythonPath, interp)
	return interp, nil
}

func probeRaw(pythonPath string) (*Interpreter, error) {
	cmd := exec.Command(pythonPath, "-c", probeScript)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to probe %s: %w", pythonPath, err)
	}

	var result probeResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse probe output: %w", err)
	}

	version, err := pep440.Parse(result.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to parse version %q: %w", result.Version, err)
	}

	impl := CPython
	switch strings.ToLower(result.Implementation) {
	case "pypy":
		impl = PyPy
	case "graalpy":
		impl = GraalPy
	}

	return &Interpreter{
		Path:         pythonPath,
		Version:      version,
		Prefix:       result.Prefix,
		SitePackages: result.SitePackages,
		SysPaths:     result.SysPaths,
		IsVenv:       result.IsVenv,
		Impl:         impl,
		Scheme: InstallScheme{
			PureLib: result.PureLib,
			PlatLib: result.PlatLib,
			Scripts: result.Scripts,
		},
	}, nil
}

func ReadPythonVersionFile(dir string) (string, error) {
	path := filepath.Join(dir, ".python-version")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func WritePythonVersionFile(dir string, version string) error {
	path := filepath.Join(dir, ".python-version")
	return os.WriteFile(path, []byte(version+"\n"), 0644)
}

func interpreterCacheKey(pythonPath string) string {
	resolved, err := filepath.EvalSymlinks(pythonPath)
	if err != nil {
		resolved = pythonPath
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return ""
	}
	raw := fmt.Sprintf("%s:%d", resolved, info.ModTime().UnixNano())
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func interpreterCacheDir() string {
	return filepath.Join(config.CacheDir(), "interpreters")
}

func loadCachedProbe(pythonPath string) *Interpreter {
	key := interpreterCacheKey(pythonPath)
	if key == "" {
		return nil
	}
	path := filepath.Join(interpreterCacheDir(), key+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var interp Interpreter
	if json.Unmarshal(data, &interp) != nil {
		return nil
	}
	return &interp
}

func saveCachedProbe(pythonPath string, interp *Interpreter) {
	key := interpreterCacheKey(pythonPath)
	if key == "" {
		return
	}
	dir := interpreterCacheDir()
	os.MkdirAll(dir, 0755)
	data, err := json.Marshal(interp)
	if err != nil {
		return
	}
	os.WriteFile(filepath.Join(dir, key+".json"), data, 0644)
}
