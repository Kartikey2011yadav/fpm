package cli

import (
	"context"
	"fmt"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/spf13/cobra"
)

func init() {
	pythonListCmd.RunE = runPythonList
	pythonInstallCmd.RunE = runPythonInstall
	pythonUseCmd.RunE = runPythonUse
	pythonPinCmd.RunE = runPythonPin
	pythonUninstallCmd.RunE = runPythonUninstall
}

func runPythonPin(cmd *cobra.Command, args []string) error {
	return python.UseVersion(args[0], false)
}

func runPythonList(cmd *cobra.Command, args []string) error {
	// List managed installations
	managed, err := python.ListManaged()
	if err != nil {
		return err
	}

	// List system Python
	finder := python.NewFinder()
	all, _ := finder.FindAll()

	fmt.Println("Installed Python versions:")
	fmt.Println()

	if len(managed) > 0 {
		fmt.Println("  fpm-managed:")
		for _, m := range managed {
			marker := " "
			if m.Current {
				marker = "*"
			}
			fmt.Printf("    %s %s  (%s)\n", marker, m.Version, m.Path)
		}
		fmt.Println()
	}

	fmt.Println("  System:")
	for _, interp := range all {
		if interp.IsManaged {
			continue
		}
		marker := " "
		fmt.Printf("    %s %s  (%s)\n", marker, interp.VersionString(), interp.Path)
	}

	fmt.Printf("\n  Managed install directory: %s\n", config.PythonInstallDir())
	fmt.Printf("  Bin directory: %s\n", config.BinDir())

	return nil
}

func runPythonInstall(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	for _, version := range args {
		_, err := python.InstallVersion(ctx, version)
		if err != nil {
			fmt.Printf("  Failed to install Python %s: %v\n", version, err)
			continue
		}
	}

	return nil
}

func runPythonUse(cmd *cobra.Command, args []string) error {
	version := args[0]
	global := flagSystem

	if err := python.UseVersion(version, global); err != nil {
		return err
	}

	if global {
		fmt.Printf("Switched global Python to %s\n", version)
		fmt.Printf("  Symlinks updated in %s\n", config.BinDir())
	} else {
		fmt.Printf("Pinned Python %s for this project (.python-version)\n", version)
	}

	return nil
}

func runPythonUninstall(cmd *cobra.Command, args []string) error {
	for _, version := range args {
		if err := python.UninstallVersion(version); err != nil {
			fmt.Printf("  %v\n", err)
			continue
		}
		fmt.Printf("  Uninstalled Python %s\n", version)
	}
	return nil
}
