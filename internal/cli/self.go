package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Manage the fpm binary",
}

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update fpm to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm self update: not yet implemented")
		return nil
	},
}

func init() {
	selfCmd.AddCommand(selfUpdateCmd)
	rootCmd.AddCommand(selfCmd)
}
