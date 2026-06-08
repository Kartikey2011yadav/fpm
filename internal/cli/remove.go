package cli

import (
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [packages...]",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove packages",
	Long: `Remove packages from the current environment.
Use --purge to also remove unused dependencies that were only needed
by the removed package.`,
	Example: `  # Remove a package
  fpm remove -s requests

  # Remove and clean unused dependencies
  fpm remove -s requests --purge

  # Remove multiple packages
  fpm remove -s flask sqlalchemy

  # Using pip-compatible alias
  fpm uninstall -s numpy`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    nil, // implemented in remove_impl.go
}

func init() {
	removeCmd.Flags().Bool("purge", false, "Also remove unused dependencies")
	rootCmd.AddCommand(removeCmd)
}
