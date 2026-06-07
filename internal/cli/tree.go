package cli

import (
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree",
	Short: "Display the dependency tree",
	Long: `Show the full dependency tree for the current environment.
Visualizes which packages depend on which, with version information.`,
	Example: `  # Show full dependency tree
  fpm tree

  # Limit depth
  fpm tree --depth 2

  # Show what depends on a package (reverse deps)
  fpm tree --invert`,
	GroupID: "package",
	RunE:    nil, // implemented in tree_impl.go
}

func init() {
	treeCmd.Flags().Int("depth", 0, "Maximum depth to display (0 = unlimited)")
	treeCmd.Flags().Bool("invert", false, "Show reverse dependencies")
	rootCmd.AddCommand(treeCmd)
}
