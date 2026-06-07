package tool

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/venv"
)

type Tool struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	PythonVersion string  `json:"python_version"`
	Entrypoints []string  `json:"entrypoints"`
	InstalledAt time.Time `json:"installed_at"`
	Path        string    `json:"path"`
}

type Registry struct {
	dir string
}

func NewRegistry() *Registry {
	return &Registry{dir: config.ToolsDir()}
}

func (r *Registry) Install(packageName, version string) (*Tool, error) {
	toolDir := filepath.Join(r.dir, packageName)

	// Create isolated venv for this tool
	finder := python.NewFinder()
	interp, err := finder.FindBest("")
	if err != nil {
		return nil, fmt.Errorf("no Python found: %w", err)
	}

	venvPath := filepath.Join(toolDir, ".venv")
	v, err := venv.Create(venv.CreateOptions{
		Path:        venvPath,
		Interpreter: interp,
		Prompt:      packageName,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create tool environment: %w", err)
	}

	// Install the package into the isolated venv using pip
	pipArgs := []string{"-m", "pip", "install", "--quiet"}
	if version != "" {
		pipArgs = append(pipArgs, packageName+"=="+version)
	} else {
		pipArgs = append(pipArgs, packageName)
	}

	cmd := exec.Command(v.PythonPath, pipArgs...)
	cmd.Env = append(os.Environ(), "VIRTUAL_ENV="+v.Path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("failed to install %s: %s", packageName, string(output))
	}

	// Find installed entrypoints
	entrypoints := findEntrypoints(v.BinDir)

	// Symlink entrypoints to global bin
	binDir := config.BinDir()
	os.MkdirAll(binDir, 0755)
	for _, ep := range entrypoints {
		src := filepath.Join(v.BinDir, ep)
		dst := filepath.Join(binDir, ep)
		os.Remove(dst) // remove existing symlink
		os.Symlink(src, dst)
	}

	tool := &Tool{
		Name:          packageName,
		Version:       version,
		PythonVersion: interp.VersionString(),
		Entrypoints:   entrypoints,
		InstalledAt:   time.Now(),
		Path:          toolDir,
	}

	// Save receipt
	r.saveReceipt(tool)

	return tool, nil
}

func (r *Registry) Run(packageName string, args []string) error {
	// Check if tool is installed
	tool, err := r.Get(packageName)
	if err != nil {
		// Not installed — create ephemeral environment
		return r.runEphemeral(packageName, args)
	}

	// Find the entrypoint binary
	binPath := filepath.Join(tool.Path, ".venv", "bin", packageName)
	if _, err := os.Stat(binPath); err != nil {
		// Try first entrypoint
		if len(tool.Entrypoints) > 0 {
			binPath = filepath.Join(tool.Path, ".venv", "bin", tool.Entrypoints[0])
		}
	}

	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func (r *Registry) runEphemeral(packageName string, args []string) error {
	// Check cached ephemeral environment first
	envDir := filepath.Join(config.CacheDir(), "environments", packageName)
	venvPath := filepath.Join(envDir, ".venv")
	binPath := filepath.Join(venvPath, "bin", packageName)

	if _, err := os.Stat(binPath); err == nil {
		// Cached environment exists — reuse it
		execCmd := exec.Command(binPath, args...)
		execCmd.Stdout = os.Stdout
		execCmd.Stderr = os.Stderr
		execCmd.Stdin = os.Stdin
		return execCmd.Run()
	}

	// Create cached environment
	os.MkdirAll(envDir, 0755)

	finder := python.NewFinder()
	interp, err := finder.FindBest("")
	if err != nil {
		return err
	}

	v, err := venv.Create(venv.CreateOptions{
		Path:        venvPath,
		Interpreter: interp,
	})
	if err != nil {
		return err
	}

	// Install
	cmd := exec.Command(v.PythonPath, "-m", "pip", "install", "--quiet", packageName)
	cmd.Env = append(os.Environ(), "VIRTUAL_ENV="+v.Path)
	if output, err := cmd.CombinedOutput(); err != nil {
		os.RemoveAll(envDir)
		return fmt.Errorf("failed to install %s: %s", packageName, string(output))
	}

	// Run
	binPath = filepath.Join(v.BinDir, packageName)
	execCmd := exec.Command(binPath, args...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	return execCmd.Run()
}

func (r *Registry) List() ([]*Tool, error) {
	entries, err := os.ReadDir(r.dir)
	if err != nil {
		return nil, nil
	}

	var tools []*Tool
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tool, err := r.Get(entry.Name())
		if err != nil {
			continue
		}
		tools = append(tools, tool)
	}
	return tools, nil
}

func (r *Registry) Get(name string) (*Tool, error) {
	receiptPath := filepath.Join(r.dir, name, "receipt.json")
	data, err := os.ReadFile(receiptPath)
	if err != nil {
		return nil, err
	}
	var tool Tool
	return &tool, json.Unmarshal(data, &tool)
}

func (r *Registry) Uninstall(name string) error {
	tool, err := r.Get(name)
	if err != nil {
		return fmt.Errorf("tool %s not installed", name)
	}

	// Remove symlinks from bin
	binDir := config.BinDir()
	for _, ep := range tool.Entrypoints {
		os.Remove(filepath.Join(binDir, ep))
	}

	// Remove tool directory
	return os.RemoveAll(filepath.Join(r.dir, name))
}

func (r *Registry) saveReceipt(tool *Tool) {
	data, _ := json.MarshalIndent(tool, "", "  ")
	os.MkdirAll(filepath.Dir(filepath.Join(r.dir, tool.Name, "receipt.json")), 0755)
	os.WriteFile(filepath.Join(r.dir, tool.Name, "receipt.json"), data, 0644)
}

func findEntrypoints(binDir string) []string {
	entries, err := os.ReadDir(binDir)
	if err != nil {
		return nil
	}

	skip := map[string]bool{
		"python": true, "python3": true, "pip": true, "pip3": true,
		"activate": true, "activate.fish": true, "activate.csh": true,
		"activate.ps1": true, "activate.bat": true, "deactivate.bat": true,
	}

	var entrypoints []string
	for _, entry := range entries {
		if entry.IsDir() || skip[entry.Name()] {
			continue
		}
		entrypoints = append(entrypoints, entry.Name())
	}
	return entrypoints
}
