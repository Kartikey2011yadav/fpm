package cli

import (
	"github.com/spf13/cobra"
)

var venvCmd = &cobra.Command{
	Use:   "venv [path]",
	Short: "Create a virtual environment",
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	venvCmd.Flags().String("python", "", "Python version or path to use")
	venvCmd.Flags().Bool("system-packages", false, "Allow access to system site-packages")
	venvCmd.Flags().String("prompt", "", "Custom prompt prefix")
	rootCmd.AddCommand(venvCmd)
}
