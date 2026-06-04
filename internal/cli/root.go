package cli

import (
	"github.com/spf13/cobra"
)

var (
	flagVerbose    bool
	flagQuiet      bool
	flagColor      string
	flagNoProgress bool
	flagJSON       bool
	flagGlobal     bool
)

var rootCmd = &cobra.Command{
	Use:   "fpm",
	Short: "Fast Package Manager for Python",
	Long:  "fpm is a high-performance Python package manager that coexists with pip/conda, provides immutable package pinning, and uses content-addressable caching for zero-duplication installs.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.PersistentFlags().StringVar(&flagColor, "color", "auto", "Control color output (auto, always, never)")
	rootCmd.PersistentFlags().BoolVar(&flagNoProgress, "no-progress", false, "Disable progress bars")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagGlobal, "global", false, "Apply to global/system environment instead of local")
}

func Execute() error {
	return rootCmd.Execute()
}
