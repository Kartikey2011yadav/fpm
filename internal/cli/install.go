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

Without a virtual environment, installs to system Python (like pip).
Use --global to explicitly target the system even when inside a venv.`,
	Example: `  # Install a package
  fpm install requests

  # Install with version constraint
  fpm install "numpy>=1.24"

  # Install multiple packages
  fpm install flask sqlalchemy redis

  # Install globally (system-wide)
  fpm install --global black`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    nil, // implemented in install_impl.go
}

func init() {
	rootCmd.AddCommand(installCmd)
}
