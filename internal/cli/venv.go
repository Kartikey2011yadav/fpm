package cli

import (
	"github.com/spf13/cobra"
)

var venvCmd = &cobra.Command{
	Use:   "venv [path]",
	Short: "Create a virtual environment",
	Long: `Create an isolated virtual environment for the project.
Generates activation scripts for all major shells.`,
	Example: `  # Create .venv in current directory
  fpm venv

  # Create at a custom path
  fpm venv .env

  # Use a specific Python version
  fpm venv --python 3.12`,
	Args:    cobra.MaximumNArgs(1),
	GroupID: "environment",
}

func init() {
	venvCmd.Flags().String("python", "", "Python version or path to use")
	venvCmd.Flags().Bool("system-packages", false, "Allow access to system site-packages")
	venvCmd.Flags().String("prompt", "", "Custom prompt prefix")
	rootCmd.AddCommand(venvCmd)
}
