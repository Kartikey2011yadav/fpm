package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/spf13/cobra"
)

func init() {
	initCmd.RunE = runInit
}

func runInit(cmd *cobra.Command, args []string) error {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absDir, 0755); err != nil {
		return err
	}

	projectName := filepath.Base(absDir)

	// Check if pyproject.toml already exists
	pyprojectPath := filepath.Join(absDir, "pyproject.toml")
	if _, err := os.Stat(pyprojectPath); err == nil {
		return fmt.Errorf("pyproject.toml already exists in %s", absDir)
	}

	// Find Python
	finder := python.NewFinder()
	interp, err := finder.FindBest("")
	if err != nil {
		return fmt.Errorf("no Python interpreter found: %w", err)
	}

	pythonVersion := fmt.Sprintf("%d.%d", interp.Version.Major(), interp.Version.Minor())

	// Write pyproject.toml
	cfg := &workspace.PyProjectToml{
		Project: workspace.ProjectTable{
			Name:           projectName,
			Version:        "0.1.0",
			Description:    "",
			RequiresPython: fmt.Sprintf(">=%s", pythonVersion),
			Dependencies:   []string{},
		},
	}

	if err := workspace.WritePyProjectToml(absDir, cfg); err != nil {
		return fmt.Errorf("failed to write pyproject.toml: %w", err)
	}

	// Write .python-version
	if err := python.WritePythonVersionFile(absDir, pythonVersion); err != nil {
		return fmt.Errorf("failed to write .python-version: %w", err)
	}

	// Create virtual environment
	venvPath := filepath.Join(absDir, ".venv")
	_, err = venv.Create(venv.CreateOptions{
		Path:        venvPath,
		Interpreter: interp,
		Prompt:      projectName,
	})
	if err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}

	fmt.Printf("Initialized project %s in %s\n", projectName, absDir)
	fmt.Printf("  Python: %s (%s)\n", interp.VersionString(), interp.Path)
	fmt.Printf("  Venv: %s\n", venvPath)

	return nil
}
