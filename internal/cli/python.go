package cli

import (
	"github.com/spf13/cobra"
)

var pythonCmd = &cobra.Command{
	Use:   "python",
	Short: "Manage Python versions",
	Long: `Install, switch, and manage Python versions.
Downloads from python-build-standalone for fast, reliable installs.`,
	GroupID: "environment",
}

var pythonListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available and installed Python versions",
}

var pythonInstallCmd = &cobra.Command{
	Use:     "install [versions...]",
	Short:   "Install Python versions",
	Example: `  fpm python install 3.12`,
	Args:    cobra.MinimumNArgs(1),
}

var pythonUseCmd = &cobra.Command{
	Use:   "use <version>",
	Short: "Switch active Python version",
	Long:  "Switch the active Python version for the current project (local) or system-wide (--system).",
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
