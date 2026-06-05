package cli

import (
	"github.com/spf13/cobra"
)

var pipCmd = &cobra.Command{
	Use:   "pip",
	Short: "pip-compatible interface",
}

var pipInstallCmd = &cobra.Command{
	Use:   "install [packages...]",
	Short: "Install packages (pip-compatible)",
	Args:  cobra.MinimumNArgs(1),
}

var pipFreezeCmd = &cobra.Command{
	Use:   "freeze",
	Short: "Output installed packages in requirements format",
}

var pipListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all packages visible in the environment (all managers)",
}

var pipShowCmd = &cobra.Command{
	Use:   "show [packages...]",
	Short: "Show package information",
	Args:  cobra.MinimumNArgs(1),
}

func init() {
	pipCmd.AddCommand(pipInstallCmd)
	pipCmd.AddCommand(pipFreezeCmd)
	pipCmd.AddCommand(pipListCmd)
	pipCmd.AddCommand(pipShowCmd)
	rootCmd.AddCommand(pipCmd)
}
