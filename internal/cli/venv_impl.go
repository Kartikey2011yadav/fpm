package cli

import (
	"context"
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

	// Find Python interpreter matching requested version
	finder := python.NewFinder()
	interp, err := finder.FindBest(pythonVersion)

	if err != nil && pythonVersion != "" {
		// Python version not found locally — offer to download it
		fmt.Printf("Python %s not found locally.\n", pythonVersion)
		fmt.Printf("Downloading Python %s...\n", pythonVersion)

		ctx := context.Background()
		_, installErr := python.InstallVersion(ctx, pythonVersion)
		if installErr != nil {
			return fmt.Errorf("could not find or install Python %s: %w", pythonVersion, installErr)
		}

		// Retry finding after install
		interp, err = finder.FindBest(pythonVersion)
		if err != nil {
			return fmt.Errorf("Python %s installed but not found: %w", pythonVersion, err)
		}
	} else if err != nil {
		return fmt.Errorf("could not find Python: %w\n  hint: specify a version with --python 3.11 or install one with 'fpm python install 3.11'", err)
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
	fmt.Printf("\n")
	fmt.Printf("  This venv uses Python %s independently of your system Python.\n", interp.VersionString())
	fmt.Printf("  Your global Python is unaffected.\n")

	return nil
}
