package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

const Version = "0.1.0-dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show fpm version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fpm %s\n", Version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
