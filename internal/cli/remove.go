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
  fpm remove -sp requests

  # Force remove a pip-installed package
  fpm remove -sf numpy

  # Combine all: system + force + purge
  fpm remove -sfp numpy

  # Remove multiple packages
  fpm remove -s flask sqlalchemy`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    nil, // implemented in remove_impl.go
}

var autoremoveCmd = &cobra.Command{
	Use:   "autoremove",
	Short: "Remove packages not required by any other package",
	Long: `Find and remove orphaned dependencies — packages that were installed
as dependencies but are no longer needed by any other package.`,
	Example: `  # See and remove orphans from system
  fpm autoremove --system

  # In a project (venv)
  fpm autoremove`,
	GroupID: "package",
	RunE:    nil, // set in remove_impl.go
}

func init() {
	removeCmd.Flags().BoolP("purge", "p", false, "Also remove unused dependencies")
	removeCmd.Flags().BoolP("force", "f", false, "Remove even if installed by another manager (pip, uv, conda)")
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(autoremoveCmd)
}
