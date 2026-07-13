package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/depgraph"
	"github.com/kartikeyyadav/fpm/internal/env"
	"github.com/kartikeyyadav/fpm/internal/pep508"
	"github.com/kartikeyyadav/fpm/internal/python"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/kartikeyyadav/fpm/internal/workspace"
	"github.com/kartikeyyadav/fpm/pkg/types"
	"github.com/spf13/cobra"
)

func init() {
	removeCmd.RunE = runRemove
	autoremoveCmd.RunE = runAutoremove
}

func runAutoremove(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()

	var targetSitePackages string
	activeVenv, _ := venv.Detect(cwd)

	if activeVenv != nil && !flagSystem {
		targetSitePackages = activeVenv.SitePackages
	} else if flagSystem {
		finder := python.NewFinder()
		interp, err := finder.FindBest("")
		if err != nil {
			return fmt.Errorf("no Python found")
		}
		targetSitePackages = interp.SitePackages
		if targetSitePackages == "" {
			for _, p := range interp.SysPaths {
				if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
					targetSitePackages = p
					break
				}
			}
		}
	} else {
		return fmt.Errorf("no virtual environment found. Use --system to check system packages")
	}

	// Use graph first, fall back to METADATA scan (recursive until no new orphans)
	envPath := "global"
	if activeVenv != nil && !flagSystem {
		envPath = activeVenv.Path
	}
	graph := depgraph.Load(envPath)

	removed := 0
	for {
		orphans := graph.Orphans()

		// Only fall back to METADATA scan if graph has no data at all
		if len(orphans) == 0 && len(graph.Packages) == 0 {
			orphans = findOrphanDeps(targetSitePackages, nil)
		}

		if len(orphans) == 0 {
			break
		}

		confirmed := confirmRemoval(orphans)
		if confirmed == nil {
			break
		}

		for _, name := range confirmed {
			pkgName := types.NewPackageName(name)
			if err := uninstallPackage(targetSitePackages, pkgName); err != nil {
				fmt.Printf("  \033[31m✗\033[0m %s: %v\n", name, err)
				continue
			}
			graph.Remove(types.NewPackageName(name).Normalized())
			fmt.Printf("  \033[32m✓\033[0m Removed %s\n", name)
			removed++
		}
	}

	if removed == 0 {
		fmt.Println("  No orphaned packages found.")
		return nil
	}

	graph.Save(envPath)
	fmt.Printf("\n  %d package(s) removed.\n", removed)
	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	purge, _ := cmd.Flags().GetBool("purge")
	force, _ := cmd.Flags().GetBool("force")

	// Determine target site-packages
	var targetSitePackages string
	activeVenv, _ := venv.Detect(cwd)

	if activeVenv != nil && !flagSystem {
		targetSitePackages = activeVenv.SitePackages
	} else if flagSystem {
		finder := python.NewFinder()
		interp, err := finder.FindBest("")
		if err != nil {
			return fmt.Errorf("no Python found")
		}
		targetSitePackages = interp.SitePackages
		if targetSitePackages == "" {
			for _, p := range interp.SysPaths {
				if strings.Contains(p, "site-packages") || strings.Contains(p, "dist-packages") {
					targetSitePackages = p
					break
				}
			}
		}
	} else {
		// No venv and no --system: try pyproject.toml only (backward compat)
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err != nil {
			return fmt.Errorf("no virtual environment found. Use --system to remove from system Python")
		}
		for _, name := range args {
			pyproject.RemoveDependency(name)
			fmt.Printf("  \033[32m✓\033[0m Removed %s from dependencies\n", name)
		}
		workspace.WritePyProjectToml(cwd, pyproject)
		fmt.Println("\n  Run 'fpm sync' to apply changes to your environment.")
		return nil
	}

	// Uninstall packages from site-packages
	removed := 0
	for _, name := range args {
		pkgName := types.NewPackageName(name)
		if err := uninstallPackage(targetSitePackages, pkgName, force); err != nil {
			fmt.Printf("  \033[31m✗\033[0m %s: %v\n", name, err)
			continue
		}
		fmt.Printf("  \033[32m✓\033[0m Removed %s\n", name)
		removed++
	}

	// Also update pyproject.toml if in a project
	if activeVenv != nil && !flagSystem {
		pyproject, err := workspace.ReadPyProjectToml(cwd)
		if err == nil {
			for _, name := range args {
				pyproject.RemoveDependency(name)
			}
			workspace.WritePyProjectToml(cwd, pyproject)
		}
	}

	// Update dependency graph
	envPath := "global"
	if activeVenv != nil && !flagSystem {
		envPath = activeVenv.Path
	}
	graph := depgraph.Load(envPath)
	for _, name := range args {
		graph.Remove(types.NewPackageName(name).Normalized())
	}

	// Purge unused dependencies using the graph (recursive until no new orphans)
	if purge && removed > 0 && targetSitePackages != "" {
		for {
			orphans := graph.Orphans()
			// Only fall back to METADATA scan if the graph has no packages at all
			// (pre-existing environment with no graph data)
			if len(orphans) == 0 && len(graph.Packages) == 0 {
				orphans = findOrphanDeps(targetSitePackages, args)
			}
			if len(orphans) == 0 {
				break
			}
			confirmed := confirmRemoval(orphans)
			if confirmed == nil {
				break
			}
			fmt.Printf("\n  Removing %d unused dependencies:\n", len(confirmed))
			for _, orphan := range confirmed {
				pkgName := types.NewPackageName(orphan)
				if err := uninstallPackage(targetSitePackages, pkgName); err != nil {
					fmt.Printf("  \033[31m✗\033[0m %s: %v\n", orphan, err)
					continue
				}
				graph.Remove(types.NewPackageName(orphan).Normalized())
				fmt.Printf("  \033[32m✓\033[0m Removed %s \033[2m(unused dependency)\033[0m\n", orphan)
				removed++
			}
		}
	}

	graph.Save(envPath)

	if removed > 0 {
		fmt.Printf("\n  %d package(s) removed.\n", removed)
	}
	return nil
}

func confirmRemoval(packages []string) []string {
	fmt.Printf("\n  Found %d unused package(s):\n\n", len(packages))
	for _, name := range packages {
		fmt.Printf("    \033[2m×\033[0m %s\n", name)
	}

	// Non-interactive mode: remove all
	if !isInteractive() {
		return packages
	}

	fmt.Printf("\n  \033[1m?\033[0m Action:\n")
	fmt.Printf("    [a] Remove all (%d packages)\n", len(packages))
	fmt.Printf("    [e] Exclude some (select which to keep)\n")
	fmt.Printf("    [n] Abort (remove nothing)\n")
	fmt.Printf("\n  Choice [a/e/n]: ")

	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))

	switch input {
	case "a", "":
		return packages
	case "n":
		fmt.Println("  Aborted.")
		return nil
	case "e":
		return excludePackages(packages)
	default:
		return packages
	}
}

func excludePackages(packages []string) []string {
	fmt.Printf("\n  Enter package names to KEEP (comma-separated):\n")
	fmt.Printf("  > ")

	var input string
	fmt.Scanln(&input)

	keepSet := make(map[string]bool)
	for _, name := range strings.Split(input, ",") {
		name = strings.TrimSpace(strings.ToLower(name))
		if name != "" {
			keepSet[name] = true
		}
	}

	// Also keep dependencies of kept packages
	// (if user keeps jinja2, also keep markupsafe)
	envPath := "global"
	activeVenv, _ := venv.Detect("")
	if activeVenv != nil && !flagSystem {
		envPath = activeVenv.Path
	}
	graph := depgraph.Load(envPath)

	// Transitively find all deps of kept packages
	for name := range keepSet {
		addTransitiveDeps(graph, name, keepSet)
	}

	var result []string
	for _, pkg := range packages {
		norm := strings.ToLower(pkg)
		if !keepSet[norm] {
			result = append(result, pkg)
		}
	}

	kept := len(packages) - len(result)
	if kept > 0 {
		fmt.Printf("  Keeping %d package(s) (+ their deps), removing %d.\n", kept, len(result))
	}
	return result
}

func addTransitiveDeps(graph *depgraph.Graph, name string, keepSet map[string]bool) {
	node, exists := graph.Packages[name]
	if !exists {
		return
	}
	for _, dep := range node.Dependencies {
		if !keepSet[dep] {
			keepSet[dep] = true
			addTransitiveDeps(graph, dep, keepSet)
		}
	}
}

func isInteractive() bool {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func findOrphanDeps(sitePackages string, removedPkgs []string) []string {
	// Scan remaining packages
	scanner := env.NewScanner([]string{sitePackages})
	result, err := scanner.Scan()
	if err != nil || len(result.Packages) == 0 {
		return nil
	}

	// Build set of top-level requested packages from pyproject.toml
	requestedSet := make(map[string]bool)
	cwd, _ := os.Getwd()
	if pyproject, err := workspace.ReadPyProjectToml(cwd); err == nil {
		for _, dep := range pyproject.Project.Dependencies {
			req, err := pep508.ParseRequirement(dep)
			if err == nil {
				requestedSet[req.Name.Normalized()] = true
			}
		}
	}

	// Build forward dependency map: package -> its deps
	depsOf := make(map[string][]string)
	for _, pkg := range result.Packages {
		if pkg.Manager != env.ManagerFpm {
			continue
		}
		deps := readPackageDeps(pkg.DistInfo)
		depsOf[pkg.Name.Normalized()] = deps
	}

	// Compute the full set of protected packages: requested + all their transitive deps
	protectedSet := make(map[string]bool)
	var protect func(name string)
	protect = func(name string) {
		if protectedSet[name] {
			return
		}
		protectedSet[name] = true
		for _, dep := range depsOf[name] {
			protect(dep)
		}
	}
	for name := range requestedSet {
		protect(name)
	}

	// Find packages that are not protected and not required by any remaining protected package
	removedSet := make(map[string]bool)
	for _, name := range removedPkgs {
		removedSet[types.NewPackageName(name).Normalized()] = true
	}

	var orphans []string
	for _, pkg := range result.Packages {
		if pkg.Manager != env.ManagerFpm {
			continue
		}
		norm := pkg.Name.Normalized()
		if removedSet[norm] {
			continue
		}
		if protectedSet[norm] {
			continue
		}
		orphans = append(orphans, pkg.Name.Raw())
	}

	return orphans
}

func readPackageDeps(distInfoPath string) []string {
	metadataPath := filepath.Join(distInfoPath, "METADATA")
	f, err := os.Open(metadataPath)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		line := s.Text()
		if line == "" {
			break // end of headers
		}
		if strings.HasPrefix(line, "Requires-Dist: ") {
			dep := strings.TrimPrefix(line, "Requires-Dist: ")
			// Extract package name (before any version specifier or semicolon)
			dep = strings.Split(dep, " ")[0]
			dep = strings.Split(dep, ";")[0]
			dep = strings.Split(dep, "(")[0]
			dep = strings.Split(dep, "[")[0]
			dep = strings.TrimSpace(dep)
			if dep != "" {
				deps = append(deps, types.NewPackageName(dep).Normalized())
			}
		}
	}
	return deps
}

func uninstallPackage(sitePackages string, name types.PackageName, force ...bool) error {
	forceRemove := len(force) > 0 && force[0]
	// Find the dist-info directory
	entries, err := os.ReadDir(sitePackages)
	if err != nil {
		return fmt.Errorf("cannot read %s: %w", sitePackages, err)
	}

	normalized := strings.ReplaceAll(name.Normalized(), "-", "_")
	var distInfoDir string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasSuffix(entry.Name(), ".dist-info") {
			continue
		}
		entryNorm := strings.ToLower(strings.ReplaceAll(entry.Name(), "-", "_"))
		if strings.HasPrefix(entryNorm, normalized+"_") || strings.HasPrefix(entryNorm, normalized+"-") {
			distInfoDir = filepath.Join(sitePackages, entry.Name())
			break
		}
	}

	if distInfoDir == "" {
		return fmt.Errorf("package not found in %s", sitePackages)
	}

	// Check INSTALLER — only remove fpm-installed packages (unless --force)
	if !forceRemove {
		installerPath := filepath.Join(distInfoDir, "INSTALLER")
		if data, err := os.ReadFile(installerPath); err == nil {
			mgr := strings.TrimSpace(string(data))
			if mgr != "fpm" && mgr != "" {
				scanner := env.NewScanner([]string{sitePackages})
				result, _ := scanner.Scan()
				found := result.FindByName(name)
				if len(found) > 0 {
					return fmt.Errorf("installed by %s, not fpm (use --force to remove anyway)", found[0].Manager)
				}
			}
		}
	}

	// Read RECORD file to get list of installed files
	recordPath := filepath.Join(distInfoDir, "RECORD")
	files, err := parseRecord(recordPath)
	if err != nil {
		// No RECORD: just remove dist-info and top-level package dir
		os.RemoveAll(distInfoDir)
		pkgDir := filepath.Join(sitePackages, normalized)
		os.RemoveAll(pkgDir)
		return nil
	}

	// Remove all files listed in RECORD
	for _, file := range files {
		path := filepath.Join(sitePackages, file)
		os.Remove(path)
	}

	// Remove empty directories
	os.RemoveAll(distInfoDir)
	pkgDir := filepath.Join(sitePackages, normalized)
	os.RemoveAll(pkgDir)

	return nil
}

func parseRecord(recordPath string) ([]string, error) {
	f, err := os.Open(recordPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var files []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ",", 2)
		if len(parts) > 0 && parts[0] != "" {
			files = append(files, parts[0])
		}
	}
	return files, nil
}
