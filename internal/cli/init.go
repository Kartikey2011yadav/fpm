package cli

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create a new Python project",
	Args:  cobra.MaximumNArgs(1),
	RunE: nil, // implemented in init_impl.go
}

func init() {
	rootCmd.AddCommand(initCmd)
}
