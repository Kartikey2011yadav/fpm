package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolCmd = &cobra.Command{
	Use:   "tool",
	Short: "Manage Python CLI tools",
}

var toolRunCmd = &cobra.Command{
	Use:   "run <tool> [args...]",
	Short: "Run a tool in an ephemeral environment",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm tool run: not yet implemented")
		return nil
	},
}

var toolInstallCmd = &cobra.Command{
	Use:   "install <tool>",
	Short: "Install a tool permanently",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm tool install: not yet implemented")
		return nil
	},
}

var toolListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm tool list: not yet implemented")
		return nil
	},
}

var toolUninstallCmd = &cobra.Command{
	Use:   "uninstall <tool>",
	Short: "Uninstall a tool",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm tool uninstall: not yet implemented")
		return nil
	},
}

func init() {
	toolCmd.AddCommand(toolRunCmd)
	toolCmd.AddCommand(toolInstallCmd)
	toolCmd.AddCommand(toolListCmd)
	toolCmd.AddCommand(toolUninstallCmd)
	rootCmd.AddCommand(toolCmd)
}
