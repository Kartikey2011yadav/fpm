package cli

import (
	"fmt"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

func init() {
	venvCmd.RunE = runVenv
}

func runVenv(cmd *cobra.Command, args []string) error {
	path := ".venv"
	if len(args) > 0 {
		path = args[0]
	}

	pythonVersion, _ := cmd.Flags().GetString("python")
	systemPackages, _ := cmd.Flags().GetBool("system-packages")
	prompt, _ := cmd.Flags().GetString("prompt")

	// Find Python interpreter
	finder := python.NewFinder()
	interp, err := finder.FindBest(pythonVersion)
	if err != nil {
		return fmt.Errorf("could not find Python: %w", err)
	}

	absPath, _ := filepath.Abs(path)
	if prompt == "" {
		prompt = filepath.Base(filepath.Dir(absPath))
	}

	v, err := venv.Create(venv.CreateOptions{
		Path:           path,
		Interpreter:    interp,
		SystemPackages: systemPackages,
		Prompt:         prompt,
	})
	if err != nil {
		return fmt.Errorf("failed to create virtual environment: %w", err)
	}

	fmt.Printf("Created virtual environment at %s\n", v.Path)
	fmt.Printf("  Python: %s (%s)\n", interp.VersionString(), interp.Path)
	fmt.Printf("  Activate: source %s/activate\n", v.BinDir)

	return nil
}
