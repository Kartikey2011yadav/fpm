package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

func init() {
	pipListCmd.RunE = runPipList
	pipFreezeCmd.RunE = runPipFreeze
	pipInstallCmd.RunE = runPipInstall
	pipShowCmd.RunE = runPipShow
	pipListCmd.Flags().Bool("all", false, "Show packages from all site-packages (including system)")
	pipListCmd.Flags().String("manager", "", "Filter by manager (fpm, pip, uv, conda, poetry, pdm, system)")
}

func runPipList(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found")
	}

	showAll, _ := cmd.Flags().GetBool("all")
	managerFilter, _ := cmd.Flags().GetString("manager")

	// Gather all site-packages directories
	var dirs []string
	if activeVenv.SitePackages != "" {
		dirs = append(dirs, activeVenv.SitePackages)
	}
	if showAll && activeVenv.Interpreter != nil {
		dirs = append(dirs, env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)...)
	}

	scanner := env.NewScanner(dirs)
	result, err := scanner.Scan()
	if err != nil {
		return err
	}

	if len(result.Packages) == 0 {
		fmt.Println("No packages installed.")
		return nil
	}

	// Apply filter
	packages := result.Packages
	if managerFilter != "" {
		var filtered []env.InstalledPackage
		for _, p := range packages {
			if strings.EqualFold(p.Manager.String(), managerFilter) {
				filtered = append(filtered, p)
			}
		}
		packages = filtered
	}

	if flagJSON {
		printPackagesJSON(packages)
		return nil
	}

	// Table output
	fmt.Printf("%-30s %-12s %-8s %s\n", "Package", "Version", "Manager", "Location")
	fmt.Printf("%-30s %-12s %-8s %s\n", strings.Repeat("─", 30), strings.Repeat("─", 12), strings.Repeat("─", 8), strings.Repeat("─", 20))

	for _, pkg := range packages {
		location := pkg.Location
		if len(location) > 40 {
			location = "..." + location[len(location)-37:]
		}
		fmt.Printf("%-30s %-12s %-8s %s\n",
			pkg.Name.Raw(),
			pkg.Version.String(),
			pkg.Manager.String(),
			location,
		)
	}

	fmt.Printf("\n%d packages total", len(packages))
	if managerFilter == "" {
		// Count by manager
		counts := make(map[string]int)
		for _, p := range packages {
			counts[p.Manager.String()]++
		}
		parts := []string{}
		for mgr, count := range counts {
			parts = append(parts, fmt.Sprintf("%d %s", count, mgr))
		}
		fmt.Printf(" (%s)", strings.Join(parts, ", "))
	}
	fmt.Println()

	return nil
}

func runPipFreeze(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found")
	}

	dirs := []string{activeVenv.SitePackages}
	scanner := env.NewScanner(dirs)
	result, _ := scanner.Scan()

	for _, pkg := range result.Packages {
		fmt.Printf("%s==%s\n", pkg.Name.Raw(), pkg.Version.String())
	}
	return nil
}

func runPipInstall(cmd *cobra.Command, args []string) error {
	// Delegate to fpm install
	return runInstall(cmd, args)
}

func runPipShow(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, err := venv.Detect(cwd)
	if err != nil {
		return fmt.Errorf("no virtual environment found")
	}

	dirs := []string{activeVenv.SitePackages}
	if activeVenv.Interpreter != nil {
		dirs = append(dirs, env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)...)
	}
	scanner := env.NewScanner(dirs)
	result, _ := scanner.Scan()

	for _, name := range args {
		found := result.FindByName(types.NewPackageName(name))
		if len(found) == 0 {
			fmt.Printf("Package %q not found.\n", name)
			continue
		}
		pkg := found[0]
		fmt.Printf("Name: %s\n", pkg.Name.Raw())
		fmt.Printf("Version: %s\n", pkg.Version.String())
		fmt.Printf("Manager: %s\n", pkg.Manager.String())
		fmt.Printf("Location: %s\n", pkg.Location)
		fmt.Println("---")
	}
	return nil
}

func printPackagesJSON(packages []env.InstalledPackage) {
	fmt.Println("[")
	for i, pkg := range packages {
		comma := ","
		if i == len(packages)-1 {
			comma = ""
		}
		fmt.Printf("  {\"name\": %q, \"version\": %q, \"manager\": %q, \"location\": %q}%s\n",
			pkg.Name.Raw(), pkg.Version.String(), pkg.Manager.String(), pkg.Location, comma)
	}
	fmt.Println("]")
}

