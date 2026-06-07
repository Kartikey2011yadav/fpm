package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/spf13/cobra"
)

func init() {
	treeCmd.RunE = runTree
}

func runTree(cmd *cobra.Command, args []string) error {
	cwd, _ := os.Getwd()
	lockPath := filepath.Join(cwd, lock.LockFileName)

	lf, err := lock.Read(lockPath)
	if err != nil {
		return fmt.Errorf("no lockfile (fpm.lock) found in current directory.\nRun 'fpm init' to create a project, or 'fpm lock' to generate a lockfile")
	}

	depth, _ := cmd.Flags().GetInt("depth")
	invert, _ := cmd.Flags().GetBool("invert")

	if invert {
		printReverseTree(lf)
	} else {
		printTree(lf, depth)
	}

	return nil
}

func printTree(lf *lock.Lockfile, maxDepth int) {
	// Build dependency map
	depMap := make(map[string][]string)
	for _, pkg := range lf.Packages {
		var deps []string
		for _, d := range pkg.Dependencies {
			deps = append(deps, d.Name)
		}
		depMap[pkg.Name] = deps
	}

	// Find root packages (not depended on by others)
	isDep := make(map[string]bool)
	for _, pkg := range lf.Packages {
		for _, d := range pkg.Dependencies {
			isDep[d.Name] = true
		}
	}

	// Print roots
	for _, pkg := range lf.Packages {
		if !isDep[pkg.Name] {
			printTreeNode(pkg.Name, pkg.Version, depMap, lf, 0, maxDepth, make(map[string]bool))
		}
	}
}

func printTreeNode(name, version string, depMap map[string][]string, lf *lock.Lockfile, depth, maxDepth int, visited map[string]bool) {
	if maxDepth > 0 && depth >= maxDepth {
		return
	}

	indent := ""
	for i := 0; i < depth; i++ {
		if i == depth-1 {
			indent += "├── "
		} else {
			indent += "│   "
		}
	}

	fmt.Printf("%s%s %s\n", indent, name, version)

	if visited[name] {
		return
	}
	visited[name] = true

	deps := depMap[name]
	for _, dep := range deps {
		depPkg := lf.FindPackage(dep)
		depVer := ""
		if depPkg != nil {
			depVer = depPkg.Version
		}
		printTreeNode(dep, depVer, depMap, lf, depth+1, maxDepth, visited)
	}

	delete(visited, name)
}

func printReverseTree(lf *lock.Lockfile) {
	// Build reverse dependency map
	reverseDeps := make(map[string][]string)
	for _, pkg := range lf.Packages {
		for _, d := range pkg.Dependencies {
			reverseDeps[d.Name] = append(reverseDeps[d.Name], pkg.Name)
		}
	}

	for _, pkg := range lf.Packages {
		parents := reverseDeps[pkg.Name]
		if len(parents) > 0 {
			fmt.Printf("%s %s\n", pkg.Name, pkg.Version)
			for _, parent := range parents {
				fmt.Printf("    └── required by %s\n", parent)
			}
		}
	}
}
