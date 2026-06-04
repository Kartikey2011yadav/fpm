package cli

import (
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display the dependency tree",
	RunE: nil, // implemented in tree_impl.go
}

func init() {
	treeCmd.Flags().Int("depth", 0, "Maximum depth to display (0 = unlimited)")
	treeCmd.Flags().Bool("invert", false, "Show reverse dependencies")
	rootCmd.AddCommand(treeCmd)
}
