package cli

import (
	"github.com/spf13/cobra"
)

var selfCmd = &cobra.Command{
	Use:     "self",
	Short:   "Manage the fpm binary",
	GroupID: "advanced",
}

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update fpm to the latest version",
}

func init() {
	selfCmd.AddCommand(selfUpdateCmd)
	rootCmd.AddCommand(selfCmd)
}
