package cli

import (
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [packages...]",
	Aliases: []string{"uninstall", "rm"},
	Short:   "Remove packages",
	Long: `Remove packages from the current environment and update pyproject.toml.
Use 'fpm sync' afterward to reconcile the lockfile with the environment.`,
	Example: `  # Remove a package
  fpm remove requests

  # Remove multiple packages
  fpm remove flask sqlalchemy

  # Using pip-compatible alias
  fpm uninstall numpy`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    nil, // implemented in remove_impl.go
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
