package cli

import (
	"github.com/spf13/cobra"
)

var lockCmd = &cobra.Command{
	Use:   "lock",
	Short: "Generate or update the lockfile",
	Long: `Resolve all dependencies and write a deterministic lockfile (fpm.lock).
The lockfile ensures reproducible installs across machines and CI.`,
	Example: `  # Generate lockfile from pyproject.toml
  fpm lock

  # After adding new dependencies
  fpm install requests && fpm lock`,
	GroupID: "package",
	RunE:    nil, // implemented in lock_impl.go
}

func init() {
	rootCmd.AddCommand(lockCmd)
}
