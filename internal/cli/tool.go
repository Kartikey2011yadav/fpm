package cli

import (
	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage Python CLI tools",
	Long: `Install and run Python CLI tools in isolated environments.
Similar to pipx — each tool gets its own venv, avoiding conflicts.`,
	Example: `  # Run a tool without installing
  fpm tool run black .

  # Install a tool permanently
  fpm tool install ruff

  # List installed tools
  fpm tool list`,
	GroupID: "advanced",
}

var toolRunCmd = &cobra.Command{
	Use:   "run <tool> [args...]",
	Short: "Run a tool in an ephemeral environment",
	Args:  cobra.MinimumNArgs(1),
}

var toolInstallCmd = &cobra.Command{
	Use:   "install <tool>",
	Short: "Install a tool permanently",
	Args:  cobra.ExactArgs(1),
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed tools",
}

var toolUninstallCmd = &cobra.Command{
	Use:   "uninstall <tool>",
	Short: "Uninstall a tool",
	Args:  cobra.ExactArgs(1),
}

func init() {
	toolCmd.AddCommand(toolRunCmd)
	toolCmd.AddCommand(toolInstallCmd)
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolUninstallCmd)
	rootCmd.AddCommand(toolCmd)
}
