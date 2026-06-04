package cli

import (
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:   "remove [packages...]",
	Short: "Remove packages",
	Args:  cobra.MinimumNArgs(1),
	RunE: nil, // implemented in remove_impl.go
}

func init() {
	rootCmd.AddCommand(removeCmd)
}
