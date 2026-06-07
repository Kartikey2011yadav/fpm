package cli

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:                "run [command] [args...]",
	Short:              "Run a command in the managed environment",
	Long:               `Execute a command within the project's virtual environment, ensuring all managed packages are available on the path.`,
	Example: `  # Run a script
  fpm run python app.py

  # Run a tool installed in the venv
  fpm run pytest -v

  # Run with arguments
  fpm run gunicorn app:main --bind 0.0.0.0:8000`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	GroupID:            "project",
	RunE:              nil, // implemented in run_impl.go
}

func init() {
	rootCmd.AddCommand(runCmd)
}
