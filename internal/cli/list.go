package cli

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed packages",
	Long: `List all packages installed in the current environment.
Shows packages from all managers (fpm, pip, uv, conda, etc.).
Without a virtual environment, shows system-wide packages.

Use --global to explicitly target system packages even when inside a venv.`,
	Example: `  # List packages in current environment
  fpm list

  # List system-wide packages
  fpm list --global

  # Filter by manager
  fpm list --manager fpm

  # JSON output
  fpm list --json`,
	GroupID: "package",
	RunE:    nil, // set in list_impl.go
}

func init() {
	listCmd.Flags().Bool("all", false, "Show packages from all site-packages (including system)")
	listCmd.Flags().String("manager", "", "Filter by manager (fpm, pip, uv, conda, poetry, pdm, system)")
	rootCmd.AddCommand(listCmd)
}
