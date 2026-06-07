package cli

import (
	"github.com/spf13/cobra"
)

var repairCmd = &cobra.Command{
	Use:   "repair",
	Short: "Diagnose and fix fpm installation issues",
	Long: `Check fpm's configuration, cache integrity, and directory structure.
Fixes common issues like missing directories, broken symlinks, stale cache,
and configuration conflicts.`,
	Example: `  # Run full diagnostic and repair
  fpm repair

  # Move cache to a new location
  fpm config set cache.dir /new/path`,
	GroupID: "advanced",
	RunE:    nil, // set in repair_impl.go
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View and modify fpm configuration",
	Long: `View current configuration values or modify settings.
Changes are saved to the user config file (~/.config/fpm/config.toml).`,
	GroupID: "advanced",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration and paths",
	RunE:  nil, // set in repair_impl.go
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	Example: `  # Change cache directory
  fpm config set cache.dir /tmp/fpm-cache

  # Change concurrency
  fpm config set tool.concurrency 100`,
	RunE: nil, // set in repair_impl.go
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetCmd)
	rootCmd.AddCommand(repairCmd)
	rootCmd.AddCommand(configCmd)
}
