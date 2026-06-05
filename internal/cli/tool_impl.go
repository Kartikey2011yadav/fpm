package cli

import (
	"fmt"

	"github.com/kartikeyyadav/fpm/internal/tool"
	"github.com/spf13/cobra"
)

func init() {
	toolRunCmd.RunE = runToolRun
	toolInstallCmd.RunE = runToolInstall
	toolListCmd.RunE = runToolList
	toolUninstallCmd.RunE = runToolUninstall
}

func runToolRun(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("specify a tool to run")
	}

	registry := tool.NewRegistry()
	toolName := args[0]
	toolArgs := args[1:]

	return registry.Run(toolName, toolArgs)
}

func runToolInstall(cmd *cobra.Command, args []string) error {
	registry := tool.NewRegistry()
	name := args[0]

	fmt.Printf("Installing tool %s...\n", name)
	t, err := registry.Install(name, "")
	if err != nil {
		return err
	}

	fmt.Printf("Installed %s\n", t.Name)
	if len(t.Entrypoints) > 0 {
		fmt.Printf("  Entrypoints: %v\n", t.Entrypoints)
	}
	return nil
}

func runToolList(cmd *cobra.Command, args []string) error {
	registry := tool.NewRegistry()
	tools, err := registry.List()
	if err != nil || len(tools) == 0 {
		fmt.Println("No tools installed.")
		return nil
	}

	fmt.Println("Installed tools:")
	for _, t := range tools {
		fmt.Printf("  %s %s (Python %s)\n", t.Name, t.Version, t.PythonVersion)
		for _, ep := range t.Entrypoints {
			fmt.Printf("    - %s\n", ep)
		}
	}
	return nil
}

func runToolUninstall(cmd *cobra.Command, args []string) error {
	registry := tool.NewRegistry()
	name := args[0]

	if err := registry.Uninstall(name); err != nil {
		return err
	}

	fmt.Printf("Uninstalled %s\n", name)
	return nil
}
