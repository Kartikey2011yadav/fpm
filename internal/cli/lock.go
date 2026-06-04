package cli

import (
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate or update the lockfile",
	RunE: nil, // implemented in lock_impl.go
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
