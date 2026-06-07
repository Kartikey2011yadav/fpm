package cli

import (
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List installed packages",
	Long: `List packages managed by fpm in the current environment.
Use --all (-a) to show packages from all managers (pip, uv, conda, etc.).
Use --manager to filter by a specific manager.`,
	Example: `  # List fpm-managed packages
  fpm list

  # List ALL packages (fpm + pip + system + etc.)
  fpm list -a

  # Filter by a specific manager
  fpm list --manager pip

  # JSON output
  fpm list --json`,
	GroupID: "package",
	RunE:    nil, // set in list_impl.go
}

func init() {
	listCmd.Flags().BoolP("all", "a", false, "Show all packages from all managers")
	listCmd.Flags().String("manager", "", "Filter by manager (fpm, pip, uv, conda, poetry, pdm, system)")
	rootCmd.AddCommand(listCmd)
}
