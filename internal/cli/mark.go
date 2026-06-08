package cli

import (
	"fmt"

	"github.com/kartikeyyadav/fpm/internal/depgraph"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

var markCmd = &cobra.Command{
	Use:   "mark <packages...>",
	Short: "Change package install reason (requested/dependency)",
	Long: `Mark packages as explicitly requested or as transitive dependencies.
Requested packages are protected from autoremove. Dependency packages
can be cleaned up when nothing depends on them.`,
	Example: `  # Protect a package from autoremove
  fpm mark --requested numpy click

  # Mark as dependency (eligible for autoremove)
  fpm mark --dependency six setuptools

  # Show current status
  fpm mark --show numpy`,
	Args:    cobra.MinimumNArgs(1),
	GroupID: "package",
	RunE:    runMark,
}

func init() {
	markCmd.Flags().Bool("requested", false, "Mark as explicitly installed (protected)")
	markCmd.Flags().Bool("dependency", false, "Mark as transitive dependency (removable)")
	markCmd.Flags().Bool("show", false, "Show current status of packages")
	rootCmd.AddCommand(markCmd)
}

func runMark(cmd *cobra.Command, args []string) error {
	showFlag, _ := cmd.Flags().GetBool("show")
	requestedFlag, _ := cmd.Flags().GetBool("requested")
	dependencyFlag, _ := cmd.Flags().GetBool("dependency")

	envPath := "global"
	cwd := ""
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv != nil && !flagSystem {
		envPath = activeVenv.Path
	}

	graph := depgraph.Load(envPath)

	if showFlag {
		for _, name := range args {
			norm := types.NewPackageName(name).Normalized()
			if graph.IsRequested(norm) {
				fmt.Printf("  %s: \033[32mrequested\033[0m (protected from autoremove)\n", name)
			} else if _, exists := graph.Packages[norm]; exists {
				fmt.Printf("  %s: \033[33mdependency\033[0m (eligible for autoremove)\n", name)
			} else {
				fmt.Printf("  %s: \033[2mnot tracked\033[0m (not installed by fpm)\n", name)
			}
		}
		return nil
	}

	if !requestedFlag && !dependencyFlag {
		return fmt.Errorf("specify --requested or --dependency\n\n  Example: fpm mark --requested numpy")
	}

	if requestedFlag && dependencyFlag {
		return fmt.Errorf("cannot use both --requested and --dependency")
	}

	changed := 0
	for _, name := range args {
		norm := types.NewPackageName(name).Normalized()
		node, exists := graph.Packages[norm]
		if !exists {
			fmt.Printf("  \033[33m●\033[0m %s: not in dependency graph (install it first)\n", name)
			continue
		}

		if requestedFlag {
			node.Requested = true
			fmt.Printf("  \033[32m✓\033[0m %s → \033[1mrequested\033[0m (protected)\n", name)
		} else {
			node.Requested = false
			fmt.Printf("  \033[32m✓\033[0m %s → \033[1mdependency\033[0m (removable if unused)\n", name)
		}
		changed++
	}

	if changed > 0 {
		graph.Save(envPath)
		fmt.Printf("\n  Updated %d package(s).\n", changed)
	}

	return nil
}
