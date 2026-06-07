package cli

import (
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment from lockfile",
	Long: `Install or update all packages to match the lockfile exactly.
Removes packages not in the lockfile and installs missing ones.`,
	Example: `  # Sync after cloning a project
  fpm sync

  # Sync after editing pyproject.toml
  fpm lock && fpm sync`,
	GroupID: "package",
	RunE:    nil, // implemented in sync_impl.go
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
