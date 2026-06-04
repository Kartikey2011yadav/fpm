package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display the dependency tree",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("fpm tree: not yet implemented")
		return nil
	},
}

func init() {
	treeCmd.Flags().Int("depth", 0, "Maximum depth to display (0 = unlimited)")
	treeCmd.Flags().Bool("invert", false, "Show reverse dependencies")
	rootCmd.AddCommand(treeCmd)
}
