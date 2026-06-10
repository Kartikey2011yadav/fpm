package venv

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/python"
)

type Venv struct {
	Path         string
	BinDir       string
	SitePackages string
	PythonPath   string
	Interpreter  *python.Interpreter
}

type CreateOptions struct {
	Path           string
	Interpreter    *python.Interpreter
	SystemPackages bool
	Prompt         string
	Seed           bool
}

func Create(opts CreateOptions) (*Venv, error) {
	if opts.Interpreter == nil {
		return nil, fmt.Errorf("interpreter is required")
	}

	absPath, err := filepath.Abs(opts.Path)
	if err != nil {
		return nil, fmt.Errorf("invalid path: %w", err)
	}

	// Create directory structure
	binDir := filepath.Join(absPath, binDirName())
	libDir := sitePackagesDir(absPath, opts.Interpreter)

	dirs := []string{
		absPath,
		binDir,
		libDir,
	}

	for _, d := range dirs {
		if err := os.MkdirAll(d, 0755); err != nil {
			return nil, fmt.Errorf("failed to create directory %s: %w", d, err)
		}
	}

	// Create pyvenv.cfg
	if err := writePyvenvCfg(absPath, opts); err != nil {
		return nil, fmt.Errorf("failed to write pyvenv.cfg: %w", err)
	}

	// Symlink or copy Python binary
	pythonPath, err := linkPython(binDir, opts.Interpreter)
	if err != nil {
		return nil, fmt.Errorf("failed to link Python: %w", err)
	}

	// Write activation scripts
	prompt := opts.Prompt
	if prompt == "" {
		prompt = filepath.Base(absPath)
	}
	if err := writeActivationScripts(binDir, absPath, prompt); err != nil {
		return nil, fmt.Errorf("failed to write activation scripts: %w", err)
	}

	return &Venv{
		Path:         absPath,
		BinDir:       binDir,
		SitePackages: libDir,
		PythonPath:   pythonPath,
		Interpreter:  opts.Interpreter,
	}, nil
}

func binDirName() string {
	if runtime.GOOS == "windows" {
		return "Scripts"
	}
	return "bin"
}

func sitePackagesDir(venvPath string, interp *python.Interpreter) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Lib", "site-packages")
	}
	return filepath.Join(venvPath, "lib",
		fmt.Sprintf("python%d.%d", interp.Version.Major(), interp.Version.Minor()),
		"site-packages")
}

func writePyvenvCfg(venvPath string, opts CreateOptions) error {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("home = %s\n", opts.Interpreter.BinDir()))
	b.WriteString(fmt.Sprintf("implementation = %s\n", opts.Interpreter.Impl))
	b.WriteString(fmt.Sprintf("version_info = %s\n", opts.Interpreter.VersionString()))

	if opts.SystemPackages {
		b.WriteString("include-system-site-packages = true\n")
	} else {
		b.WriteString("include-system-site-packages = false\n")
	}

	if opts.Prompt != "" {
		b.WriteString(fmt.Sprintf("prompt = %s\n", opts.Prompt))
	}

	cfgPath := filepath.Join(venvPath, "pyvenv.cfg")
	return os.WriteFile(cfgPath, []byte(b.String()), 0644)
}

func linkPython(binDir string, interp *python.Interpreter) (string, error) {
	var targetName string
	if runtime.GOOS == "windows" {
		targetName = "python.exe"
	} else {
		targetName = "python3"
	}

	dest := filepath.Join(binDir, targetName)

	// Try symlink first
	err := os.Symlink(interp.Path, dest)
	if err != nil {
		// Fall back to copy on Windows or cross-device
		data, readErr := os.ReadFile(interp.Path)
		if readErr != nil {
			return "", readErr
		}
		if writeErr := os.WriteFile(dest, data, 0755); writeErr != nil {
			return "", writeErr
		}
	}

	// Also create "python" link
	if runtime.GOOS != "windows" {
		pythonLink := filepath.Join(binDir, "python")
		os.Symlink(targetName, pythonLink) // ignore error, non-critical
	}

	return dest, nil
}

func Detect(dir string) (*Venv, error) {
	// Check VIRTUAL_ENV env var first (set by activate script)
	if envVenv := os.Getenv("VIRTUAL_ENV"); envVenv != "" {
		cfgPath := filepath.Join(envVenv, "pyvenv.cfg")
		if _, err := os.Stat(cfgPath); err == nil {
			binDir := filepath.Join(envVenv, binDirName())
			pythonPath := filepath.Join(binDir, pythonBinName())
			if _, err := os.Stat(pythonPath); err == nil {
				interp, _ := python.Probe(pythonPath)
				return &Venv{
					Path:         envVenv,
					BinDir:       binDir,
					PythonPath:   pythonPath,
					Interpreter:  interp,
					SitePackages: findSitePackages(envVenv),
				}, nil
			}
		}
	}

	// Walk up to find pyvenv.cfg
	current := dir
	for {
		cfgPath := filepath.Join(current, "pyvenv.cfg")
		if _, err := os.Stat(cfgPath); err == nil {
			binDir := filepath.Join(current, binDirName())
			pythonPath := filepath.Join(binDir, pythonBinName())
			if _, err := os.Stat(pythonPath); err == nil {
				interp, _ := python.Probe(pythonPath)
				return &Venv{
					Path:         current,
					BinDir:       binDir,
					PythonPath:   pythonPath,
					Interpreter:  interp,
					SitePackages: findSitePackages(current),
				}, nil
			}
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	// Check .venv in the given directory
	venvPath := filepath.Join(dir, ".venv")
	cfgPath := filepath.Join(venvPath, "pyvenv.cfg")
	if _, err := os.Stat(cfgPath); err == nil {
		binDir := filepath.Join(venvPath, binDirName())
		pythonPath := filepath.Join(binDir, pythonBinName())
		interp, _ := python.Probe(pythonPath)
		return &Venv{
			Path:         venvPath,
			BinDir:       binDir,
			PythonPath:   pythonPath,
			Interpreter:  interp,
			SitePackages: findSitePackages(venvPath),
		}, nil
	}

	return nil, fmt.Errorf("no virtual environment found")
}

func pythonBinName() string {
	if runtime.GOOS == "windows" {
		return "python.exe"
	}
	return "python3"
}

func findSitePackages(venvPath string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvPath, "Lib", "site-packages")
	}
	// Look for lib/python3.X/site-packages
	libDir := filepath.Join(venvPath, "lib")
	entries, err := os.ReadDir(libDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() && strings.HasPrefix(e.Name(), "python") {
			sp := filepath.Join(libDir, e.Name(), "site-packages")
			if _, err := os.Stat(sp); err == nil {
				return sp
			}
		}
	}
	return ""
}
