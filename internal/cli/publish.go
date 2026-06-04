package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish [dist...]",
	Short: "Upload distributions to a package index",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm publish: not yet implemented")
		return nil
	},
}

func init() {
	publishCmd.Flags().String("repository", "pypi", "Target repository")
	publishCmd.Flags().String("token", "", "Authentication token")
	rootCmd.AddCommand(publishCmd)
}
