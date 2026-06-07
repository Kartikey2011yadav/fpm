package cli

import (
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create a new Python project",
	Long: `Initialize a new Python project with pyproject.toml and a virtual environment.
Creates the project structure with sensible defaults.`,
	Example: `  # Initialize in current directory
  fpm init

  # Initialize in a new directory
  fpm init my-project`,
	Args:    cobra.MaximumNArgs(1),
	GroupID: "project",
	RunE:    nil, // implemented in init_impl.go
}

func init() {
	rootCmd.AddCommand(initCmd)
}
