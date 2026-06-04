package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate or update the lockfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm lock: not yet implemented")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
