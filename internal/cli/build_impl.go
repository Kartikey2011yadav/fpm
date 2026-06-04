package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/build"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

func init() {
	buildCmd.RunE = runBuild
}

func runBuild(cmd *cobra.Command, args []string) error {
	sourceDir := "."
	if len(args) > 0 {
		sourceDir = args[0]
	}

	outDir, _ := cmd.Flags().GetString("out")
	buildWheel, _ := cmd.Flags().GetBool("wheel")
	buildSdist, _ := cmd.Flags().GetBool("sdist")

	// Default: build both
	if !buildWheel && !buildSdist {
		buildWheel = true
		buildSdist = true
	}

	absSource, _ := filepath.Abs(sourceDir)
	absOut, _ := filepath.Abs(outDir)

	// Find Python
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found. Run 'fpm init' first")
	}

	frontend := build.NewFrontend(activeVenv.PythonPath, absSource, absOut)

	if buildWheel {
		fmt.Println("Building wheel...")
		result, err := frontend.BuildWheel()
		if err != nil {
			return fmt.Errorf("wheel build failed: %w", err)
		}
		fmt.Printf("  Built: %s\n", result.WheelPath)
	}

	if buildSdist {
		fmt.Println("Building source distribution...")
		result, err := frontend.BuildSdist()
		if err != nil {
			return fmt.Errorf("sdist build failed: %w", err)
		}
		fmt.Printf("  Built: %s\n", result.SdistPath)
	}

	return nil
}
