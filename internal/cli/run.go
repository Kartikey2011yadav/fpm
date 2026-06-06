package cli

import (
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:                "run [command] [args...]",
	Short:              "Run a command in the managed environment",
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: true,
	RunE:               nil, // implemented in run_impl.go
}

func init() {
	rootCmd.AddCommand(runCmd)
}
