package cli

import (
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment from lockfile",
	RunE: nil, // implemented in sync_impl.go
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
