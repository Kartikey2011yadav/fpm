package cli

import (
	"fmt"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm pip install: not yet implemented")
		return nil
	},
}

var pipFreezeCmd = &cobra.Command{
	Use:   "freeze",
	Short: "Output installed packages in requirements format",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm pip freeze: not yet implemented")
		return nil
	},
}

var pipListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed packages",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm pip list: not yet implemented")
		return nil
	},
}

var pipShowCmd = &cobra.Command{
	Use:   "show [packages...]",
	Short: "Show package information",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm pip show: not yet implemented")
		return nil
	},
}

func init() {
	pipCmd.AddCommand(pipInstallCmd)
	pipCmd.AddCommand(pipFreezeCmd)
	pipCmd.AddCommand(pipListCmd)
	pipCmd.AddCommand(pipShowCmd)
	rootCmd.AddCommand(pipCmd)
}
