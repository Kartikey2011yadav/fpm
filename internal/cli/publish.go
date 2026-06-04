package cli

import (
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish [dist...]",
	Short: "Upload distributions to a package index",
	// implemented in publish_impl.go
}

func init() {
	publishCmd.Flags().String("repository", "pypi", "Target repository")
	publishCmd.Flags().String("token", "", "Authentication token")
	rootCmd.AddCommand(publishCmd)
}
