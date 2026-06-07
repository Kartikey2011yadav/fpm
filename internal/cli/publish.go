package cli

import (
	"github.com/spf13/cobra"
)

var publishCmd = &cobra.Command{
	Use:   "publish [dist...]",
	Short: "Upload distributions to a package index",
	Long: `Upload wheel and sdist files to PyPI or a private package index.
Defaults to PyPI; use --repository to target a different index.`,
	Example: `  # Publish to PyPI
  fpm publish

  # Publish specific files
  fpm publish dist/*.whl

  # Publish to a private index
  fpm publish --repository private --token $TOKEN`,
	GroupID: "project",
}

func init() {
	publishCmd.Flags().String("repository", "pypi", "Target repository")
	publishCmd.Flags().String("token", "", "Authentication token")
	rootCmd.AddCommand(publishCmd)
}
