package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [packages...]",
	Short: "Remove packages",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm remove: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
