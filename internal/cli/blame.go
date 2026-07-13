package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/depgraph"
	"github.com/kartikeyyadav/fpm/internal/journal"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

var blameCmd = &cobra.Command{
	Use:   "blame <package>",
	Short: "Show why and when a package was installed",
	Long: `Show the provenance of a package: when it was installed, whether it was
explicitly requested or pulled in as a dependency, and which packages depend on it.`,
	GroupID: "package",
	Args:    cobra.ExactArgs(1),
	RunE:    runBlame,
}

func init() {
	rootCmd.AddCommand(blameCmd)
}

func runBlame(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	pkgName := types.NewPackageName(args[0])

	activeVenv, _ := venv.Detect(cwd)
	envPath := ""
	if activeVenv != nil {
		envPath = activeVenv.Path
	}

	graph := depgraph.Load(envPath)
	node, exists := graph.Packages[pkgName.Normalized()]
	if !exists {
		return fmt.Errorf("package %q not found in dependency graph.\n  It may not be installed, or was installed outside fpm", args[0])
	}

	// Header
	fmt.Printf("\033[1m%s\033[0m %s\n", node.Name, node.Version)
	fmt.Println()

	// Status
	if node.Requested {
		fmt.Println("  Status:    \033[32mrequested\033[0m (explicitly installed)")
	} else {
		fmt.Println("  Status:    \033[36mtransitive\033[0m (installed as dependency)")
	}

	// Installed timestamp
	if node.InstalledAt != "" && node.InstalledAt != "pre-existing" {
		fmt.Printf("  Installed: %s\n", node.InstalledAt)
	}

	// Dependencies (what this package needs)
	if len(node.Dependencies) > 0 {
		fmt.Printf("  Requires:  %s\n", strings.Join(node.Dependencies, ", "))
	}

	// Dependents (who needs this package)
	dependents := graph.DependentsOf(pkgName.Normalized())
	if len(dependents) > 0 {
		fmt.Printf("  Required by: %s\n", strings.Join(dependents, ", "))
	} else if !node.Requested {
		fmt.Println("  Required by: \033[33m(nothing — orphan candidate)\033[0m")
	}

	// Journal lookup: find when it was first installed
	entries, _ := journal.Read(envPath, 0)
	for i := len(entries) - 1; i >= 0; i-- {
		e := entries[i]
		if e.Operation != journal.OpInstall {
			continue
		}
		for _, pkg := range e.Packages {
			name := pkg
			if idx := strings.Index(name, "=="); idx > 0 {
				name = name[:idx]
			}
			if types.NewPackageName(name).Normalized() == pkgName.Normalized() {
				ts := e.Timestamp.Local().Format("2006-01-02 15:04")
				fmt.Printf("  Journal:   %s (%s) %s\n", e.ID, ts, e.Operation)
				break
			}
		}
	}

	return nil
}
