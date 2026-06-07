package cli

import (
	"github.com/spf13/cobra"
)

var (
	flagVerbose           bool
	flagQuiet             bool
	flagColor             string
	flagNoProgress        bool
	flagJSON              bool
	flagSystem            bool
	flagAllowInsecureHost []string
)

var rootCmd = &cobra.Command{
	Use:   "fpm",
	Short: "Fast Package Manager for Python",
	Long: `fpm is a high-performance Python package manager that coexists with
pip/conda, provides immutable package pinning, and uses content-addressable
caching for zero-duplication installs.

Get started:
  fpm init            Create a new project
  fpm install <pkg>   Install a package
  fpm list            Show installed packages`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.PersistentFlags().StringVar(&flagColor, "color", "auto", "Control color output (auto, always, never)")
	rootCmd.PersistentFlags().BoolVar(&flagNoProgress, "no-progress", false, "Disable progress bars")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&flagSystem, "system", "s", false, "Install into system Python instead of virtual environment")
	rootCmd.PersistentFlags().StringSliceVar(&flagAllowInsecureHost, "allow-insecure-host", nil, "Skip TLS verification for specific hosts")

	rootCmd.AddGroup(
		&cobra.Group{ID: "package", Title: "Package Management:"},
		&cobra.Group{ID: "project", Title: "Project Management:"},
		&cobra.Group{ID: "environment", Title: "Environment:"},
		&cobra.Group{ID: "advanced", Title: "Advanced:"},
	)
}

func Execute() error {
	rootCmd.Version = Version
	rootCmd.SetVersionTemplate("fpm {{.Version}}\n")
	rootCmd.Flags().BoolP("version", "v", false, "Print version information")

	err := rootCmd.Execute()
	if err != nil {
		formatError(err)
	}
	return err
}
