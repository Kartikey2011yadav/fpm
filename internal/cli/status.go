package cli

import (
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show environment status and drift from lockfile",
	Long:    `Show packages that differ from the lockfile: added, removed, changed, or missing.`,
	GroupID: "package",
	RunE:    runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
