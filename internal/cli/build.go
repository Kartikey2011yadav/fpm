package cli

import (
	"github.com/spf13/cobra"
)

var buildCmd = &cobra.Command{
	Use:   "build [path]",
	Short: "Build a wheel or source distribution",
	Args:  cobra.MaximumNArgs(1),
	// implemented in build_impl.go
}

func init() {
	buildCmd.Flags().Bool("wheel", false, "Build only wheel")
	buildCmd.Flags().Bool("sdist", false, "Build only source distribution")
	buildCmd.Flags().StringP("out", "o", "dist", "Output directory")
	rootCmd.AddCommand(buildCmd)
}
