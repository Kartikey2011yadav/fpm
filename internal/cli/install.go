package cli

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [packages...]",
	Aliases: []string{"add"},
	Short:   "Install packages",
	Long:    "Install packages into the current environment. Use --global to install system-wide.",
	Args:    cobra.MinimumNArgs(1),
	RunE: nil, // implemented in install_impl.go
}

func init() {
	rootCmd.AddCommand(installCmd)
}
