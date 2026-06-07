package cli

import (
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build [path]",
	Short: "Build a wheel or source distribution",
	Long: `Build the project into distributable formats (wheel and/or sdist).
Supports PEP 517 build backends (setuptools, flit, hatch, maturin).`,
	Example: `  # Build both wheel and sdist
  fpm build

  # Build only wheel
  fpm build --wheel

  # Output to custom directory
  fpm build -o packages/`,
	Args:    cobra.MaximumNArgs(1),
	GroupID: "project",
}

func init() {
	buildCmd.Flags().Bool("wheel", false, "Build only wheel")
	buildCmd.Flags().Bool("sdist", false, "Build only source distribution")
	buildCmd.Flags().StringP("out", "o", "dist", "Output directory")
	rootCmd.AddCommand(buildCmd)
}
