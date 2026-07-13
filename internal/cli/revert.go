package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/journal"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

var revertCmd = &cobra.Command{
	Use:   "revert <operation-id>",
	Short: "Undo a previous operation by its journal ID",
	Long: `Reverse a specific install or remove operation.

Use 'fpm log --oneline' to find operation IDs.
Install operations are reverted by removing the packages.
Remove operations are reverted by showing what to reinstall.`,
	GroupID: "advanced",
	Args:    cobra.ExactArgs(1),
	RunE:    runRevert,
}

func init() {
	rootCmd.AddCommand(revertCmd)
}

func runRevert(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	targetID := args[0]

	activeVenv, _ := venv.Detect(cwd)
	envPath := ""
	if activeVenv != nil {
		envPath = activeVenv.Path
	}

	entries, err := journal.Read(envPath, 0)
	if err != nil || len(entries) == 0 {
		return fmt.Errorf("no operation history found. Nothing to revert")
	}

	// Find the entry
	var target *journal.Entry
	for i := range entries {
		if entries[i].ID == targetID {
			target = &entries[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("operation %q not found in history.\n  Run 'fpm log --oneline' to see available operations", targetID)
	}

	switch target.Operation {
	case journal.OpInstall:
		return revertInstall(activeVenv, envPath, target)
	case journal.OpRemove:
		return revertRemove(target)
	default:
		return fmt.Errorf("cannot revert operation type %q. Only install and remove can be reverted", target.Operation)
	}
}

func revertInstall(activeVenv *venv.Venv, envPath string, entry *journal.Entry) error {
	if activeVenv == nil {
		return fmt.Errorf("no virtual environment found")
	}

	fmt.Printf("Reverting: %s\n", entry.Operation)
	removed := 0
	for _, pkgSpec := range entry.Packages {
		name := pkgSpec
		if idx := strings.Index(name, "=="); idx > 0 {
			name = name[:idx]
		}
		pkgName := types.NewPackageName(name)
		if err := uninstallPackage(activeVenv.SitePackages, pkgName); err != nil {
			continue
		}
		fmt.Printf("  \033[32m✓\033[0m Removed %s\n", name)
		removed++
	}

	if removed > 0 {
		journal.Record(envPath, journal.OpRemove, entry.Packages, "revert "+entry.ID)
		fmt.Printf("\nReverted %d package(s).\n", removed)
	} else {
		fmt.Println("  Nothing to revert (packages may already be removed).")
	}
	return nil
}

func revertRemove(entry *journal.Entry) error {
	fmt.Printf("Reverting: %s %s\n", entry.Operation, strings.Join(entry.Packages, ", "))
	fmt.Println("\n  To restore these packages, run:")
	fmt.Printf("    fpm install %s\n", strings.Join(entry.Packages, " "))
	return nil
}
