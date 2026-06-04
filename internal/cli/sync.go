package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment from lockfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm sync: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
