package cli

import (
	"fmt"
	"os"

	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/spf13/cobra"
)

func init() {
	removeCmd.RunE = runRemove
}

func runRemove(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	pyproject, err := workspace.ReadPyProjectToml(cwd)
	if err != nil {
		return fmt.Errorf("no pyproject.toml found")
	}

	for _, name := range args {
		pyproject.RemoveDependency(name)
		fmt.Printf("  Removed %s from dependencies\n", name)
	}

	if err := workspace.WritePyProjectToml(cwd, pyproject); err != nil {
		return fmt.Errorf("failed to update pyproject.toml: %w", err)
	}

	fmt.Println("\nRun 'fpm lock' and 'fpm sync' to update your environment.")
	return nil
}
