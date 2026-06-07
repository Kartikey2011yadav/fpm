package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/script"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

func init() {
	runCmd.RunE = runRun
}

func runRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	cwd, _ := os.Getwd()

	// Check if first arg is a Python script with inline deps
	if len(args) >= 1 && isPythonScript(args[0]) {
		return runScript(args[0], args[1:])
	}

	// Find virtual environment
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found.\nRun 'fpm venv' to create one, or 'fpm init' to start a new project")
	}

	// Execute command with venv's bin on PATH
	command := args[0]
	cmdArgs := args[1:]

	// Look for the command in the venv bin directory first
	binPath := filepath.Join(activeVenv.BinDir, command)
	if _, err := os.Stat(binPath); err == nil {
		command = binPath
	}

	execCmd := exec.Command(command, cmdArgs...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin

	// Set up environment with venv
	execCmd.Env = buildVenvEnv(activeVenv)

	return execCmd.Run()
}

func runScript(scriptPath string, args []string) error {
	meta, err := script.ParseInlineMetadata(scriptPath)
	if err != nil {
		// No inline deps, just run with current env
		return runWithCurrentEnv(scriptPath, args)
	}

	if len(meta.Dependencies) == 0 {
		return runWithCurrentEnv(scriptPath, args)
	}

	// For scripts with inline deps, we'd create an ephemeral env
	// For now, run with current env and print a note
	fmt.Fprintf(os.Stderr, "Note: script declares dependencies %v (ephemeral env not yet implemented)\n", meta.Dependencies)
	return runWithCurrentEnv(scriptPath, args)
}

func runWithCurrentEnv(scriptPath string, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found")
	}

	pythonPath := activeVenv.PythonPath
	cmdArgs := append([]string{scriptPath}, args...)

	execCmd := exec.Command(pythonPath, cmdArgs...)
	execCmd.Stdout = os.Stdout
	execCmd.Stderr = os.Stderr
	execCmd.Stdin = os.Stdin
	execCmd.Env = buildVenvEnv(activeVenv)

	return execCmd.Run()
}

func buildVenvEnv(v *venv.Venv) []string {
	env := os.Environ()

	// Prepend venv bin to PATH
	for i, e := range env {
		if len(e) > 5 && e[:5] == "PATH=" {
			env[i] = fmt.Sprintf("PATH=%s%c%s", v.BinDir, os.PathListSeparator, e[5:])
			break
		}
	}

	env = append(env, fmt.Sprintf("VIRTUAL_ENV=%s", v.Path))
	return env
}

func isPythonScript(path string) bool {
	ext := filepath.Ext(path)
	return ext == ".py"
}
