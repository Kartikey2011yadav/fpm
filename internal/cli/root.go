package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	flagVerbose           bool
	flagQuiet             bool
	flagColor             string
	flagNoProgress        bool
	flagJSON              bool
	flagGlobal            bool
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
	rootCmd.PersistentFlags().BoolVarP(&flagVerbose, "verbose", "v", false, "Enable verbose output")
	rootCmd.PersistentFlags().BoolVarP(&flagQuiet, "quiet", "q", false, "Suppress all output except errors")
	rootCmd.PersistentFlags().StringVar(&flagColor, "color", "auto", "Control color output (auto, always, never)")
	rootCmd.PersistentFlags().BoolVar(&flagNoProgress, "no-progress", false, "Disable progress bars")
	rootCmd.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVar(&flagGlobal, "global", false, "Apply to global/system environment instead of local")
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
	rootCmd.Flags().BoolP("version", "V", false, "Print version information")

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\033[31merror:\033[0m %v\n", err)
	}
	return err
}
