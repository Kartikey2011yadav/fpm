package cli

import (
	"github.com/spf13/cobra"
)

var pythonCmd = &cobra.Command{
	Use:   "python",
	Short: "Manage Python versions",
}

var pythonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed Python versions",
}

var pythonInstallCmd = &cobra.Command{
	Use:   "install [versions...]",
	Short: "Install Python versions",
	Args:  cobra.MinimumNArgs(1),
}

var pythonUseCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "Switch active Python version",
	Long:  "Switch the active Python version for the current project (local) or system-wide (--global).",
	Args:  cobra.ExactArgs(1),
}

var pythonPinCmd = &cobra.Command{
	Use:   "pin <version>",
	Short: "Pin Python version for this project",
	Args:  cobra.ExactArgs(1),
}

var pythonUninstallCmd = &cobra.Command{
	Use:   "uninstall [versions...]",
	Short: "Uninstall Python versions",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	pythonCmd.AddCommand(pythonListCmd)
	pythonCmd.AddCommand(pythonInstallCmd)
	pythonCmd.AddCommand(pythonUseCmd)
	pythonCmd.AddCommand(pythonPinCmd)
	pythonCmd.AddCommand(pythonUninstallCmd)
	rootCmd.AddCommand(pythonCmd)
}
