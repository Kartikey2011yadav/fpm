package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [packages...]",
	Aliases: []string{"add"},
	Short:   "Install packages",
	Long:    "Install packages into the current environment. Use --global to install system-wide.",
	Args:    cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm install: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(installCmd)
}
