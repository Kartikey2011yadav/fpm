package cli

import (
	"github.com/spf13/cobra"
)

var Version = "0.1.0-dev"

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Show fpm version",
	GroupID: "advanced",
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Printf("fpm %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
