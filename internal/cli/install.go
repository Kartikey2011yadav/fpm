package cli

import (
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [packages...]",
	Aliases: []string{"add"},
	Short:   "Install packages",
	Long: `Install packages into the current environment. Resolves dependencies,
downloads wheels, and links them via content-addressable caching.

Without a virtual environment, requires --system flag (like uv).
Use --system (-s) to explicitly target the system Python.`,
	Example: `  # Install a package (requires active venv)
  fpm install requests

  # Install with version constraint
  fpm install "numpy>=1.24"

  # Install multiple packages
  fpm install flask sqlalchemy redis

  # Install into system Python (no venv needed)
  fpm install --system black`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    nil, // implemented in install_impl.go
}

func init() {
	rootCmd.AddCommand(installCmd)
}
