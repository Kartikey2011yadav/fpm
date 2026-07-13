package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/config"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/python"
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
	activeVenv, _ := venv.Detect(cwd)

	showAll, _ := cmd.Flags().GetBool("all")
	managerFilter, _ := cmd.Flags().GetString("manager")

	seen := make(map[string]bool)
	var dirs []string

	if flagSystem {
		sysDirs := findSystemSitePackages()
		if len(sysDirs) == 0 {
			return fmt.Errorf("no Python environment found")
		}
		for _, d := range sysDirs {
			dirs = append(dirs, d)
			seen[d] = true
		}
	} else if activeVenv != nil && activeVenv.SitePackages != "" {
		dirs = append(dirs, activeVenv.SitePackages)
		seen[activeVenv.SitePackages] = true

		if showAll {
			sysDirs := findSystemSitePackages()
			for _, d := range sysDirs {
				if !seen[d] {
					dirs = append(dirs, d)
					seen[d] = true
				}
			}
			if activeVenv.Interpreter != nil {
				for _, d := range env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths) {
					if !seen[d] {
						dirs = append(dirs, d)
						seen[d] = true
					}
				}
			}
		}
	} else {
		return fmt.Errorf("no virtual environment found. Use --system to list system packages, or run from a project directory")
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

	// Check if --mutable flag is set (show pinned column)
	showMutable, _ := cmd.Flags().GetBool("mutable")

	// Build immutable lookup if needed
	var immutableMap map[string]string
	if showMutable {
		immutableMap = make(map[string]string)
		cfg, _ := config.LoadFromCwd()
		for _, im := range cfg.Immutable.Packages {
			normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(im.Name), "-", "_"), ".", "_")
			immutableMap[normalized] = im.Version
		}
	}

	// Table output with colors
	if showMutable {
		fmt.Printf("\033[1m\033[4m%-28s %-12s %-8s %-10s %s\033[0m\n", "Package", "Version", "Manager", "Pinned", "Location")
	} else {
		fmt.Printf("\033[1m\033[4m%-28s %-12s %-8s %s\033[0m\n", "Package", "Version", "Manager", "Location")
	}

	for _, pkg := range packages {
		location := pkg.Location

		mgrStr := pkg.Manager.String()
		mgrColored := mgrStr
		switch pkg.Manager {
		case env.ManagerFpm:
			mgrColored = "\033[32m" + mgrStr + "\033[0m"
		case env.ManagerPip:
			mgrColored = "\033[33m" + mgrStr + "\033[0m"
		case env.ManagerUv:
			mgrColored = "\033[35m" + mgrStr + "\033[0m"
		case env.ManagerConda:
			mgrColored = "\033[36m" + mgrStr + "\033[0m"
		default:
			mgrColored = "\033[2m" + mgrStr + "\033[0m"
		}

		if showMutable {
			normalized := strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(pkg.Name.Normalized()), "-", "_"), ".", "_")
			pinnedVer, isPinned := immutableMap[normalized]
			pinnedStr := "\033[32mmutable\033[0m"
			if isPinned {
				pinnedStr = fmt.Sprintf("\033[31m🔒 %s\033[0m", pinnedVer)
			}
			fmt.Printf("\033[1m%-28s\033[0m \033[36m%-12s\033[0m %-8s %-10s \033[2m%s\033[0m\n",
				pkg.Name.Raw(),
				pkg.Version.String(),
				mgrColored,
				pinnedStr,
				location,
			)
		} else {
			fmt.Printf("\033[1m%-28s\033[0m \033[36m%-12s\033[0m %-8s \033[2m%s\033[0m\n",
				pkg.Name.Raw(),
				pkg.Version.String(),
				mgrColored,
				location,
			)
		}
	}

	// Summary with counts
	counts := make(map[string]int)
	for _, p := range packages {
		counts[p.Manager.String()]++
	}
	parts := []string{}
	for mgr, count := range counts {
		parts = append(parts, fmt.Sprintf("%d %s", count, mgr))
	}
	fmt.Printf("\n\033[1m%d packages\033[0m (%s)\n", len(packages), strings.Join(parts, ", "))

	return nil
}

func runPipFreeze(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)

	var dirs []string
	if activeVenv != nil && !flagSystem {
		dirs = []string{activeVenv.SitePackages}
	} else {
		dirs = findSystemSitePackages()
		if len(dirs) == 0 {
			return fmt.Errorf("no Python environment found")
		}
	}

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
	activeVenv, _ := venv.Detect(cwd)

	var dirs []string
	if activeVenv != nil && !flagSystem {
		dirs = []string{activeVenv.SitePackages}
		if activeVenv.Interpreter != nil {
			dirs = append(dirs, env.FindSitePackagesDirs(activeVenv.Interpreter.SysPaths)...)
		}
	} else {
		dirs = findSystemSitePackages()
		if len(dirs) == 0 {
			return fmt.Errorf("no Python environment found")
		}
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

func findSystemSitePackages() []string {
	// Find Python and get its site-packages
	finder := python.NewFinder()
	interp, err := finder.FindBest("")
	if err != nil {
		return nil
	}
	return env.FindSitePackagesDirs(interp.SysPaths)
}
