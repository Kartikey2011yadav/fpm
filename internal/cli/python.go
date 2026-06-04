package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pythonCmd = &cobra.Command{
	Use:   "python",
	Short: "Manage Python versions",
}

var pythonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed Python versions",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm python list: not yet implemented")
		return nil
	},
}

var pythonInstallCmd = &cobra.Command{
	Use:   "install [versions...]",
	Short: "Install Python versions",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm python install: not yet implemented")
		return nil
	},
}

var pythonUseCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "Switch active Python version",
	Long:  "Switch the active Python version for the current project (local) or system-wide (--global).",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm python use: not yet implemented")
		return nil
	},
}

var pythonPinCmd = &cobra.Command{
	Use:   "pin <version>",
	Short: "Pin Python version for this project",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm python pin: not yet implemented")
		return nil
	},
}

var pythonUninstallCmd = &cobra.Command{
	Use:   "uninstall [versions...]",
	Short: "Uninstall Python versions",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm python uninstall: not yet implemented")
		return nil
	},
}

func init() {
	pythonCmd.AddCommand(pythonListCmd)
	pythonCmd.AddCommand(pythonInstallCmd)
	pythonCmd.AddCommand(pythonUseCmd)
	pythonCmd.AddCommand(pythonPinCmd)
	pythonCmd.AddCommand(pythonUninstallCmd)
	rootCmd.AddCommand(pythonCmd)
}
