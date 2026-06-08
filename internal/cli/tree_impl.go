package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kartikeyyadav/fpm/internal/depgraph"
	"github.com/kartikeyyadav/fpm/internal/lock"
	"github.com/kartikeyyadav/fpm/internal/venv"
	"github.com/spf13/cobra"
)

func init() {
	treeCmd.RunE = runTree
}

func runTree(cmd *cobra.Command, args []string) error {
	depth, _ := cmd.Flags().GetInt("depth")
	invert, _ := cmd.Flags().GetBool("invert")

	// System tree: use depgraph
	if flagSystem {
		return printGraphTree(depth, invert)
	}

	// Project tree: use lockfile
	cwd, _ := os.Getwd()
	activeVenv, _ := venv.Detect(cwd)
	if activeVenv != nil {
		// Try depgraph for venv too
		graph := depgraph.Load(activeVenv.Path)
		if len(graph.Packages) > 0 {
			return printDepGraphTree(graph, depth, invert)
		}
	}

	lockPath := filepath.Join(cwd, lock.LockFileName)
	lf, err := lock.Read(lockPath)
	if err != nil {
		return fmt.Errorf("no lockfile (fpm.lock) found in current directory.\nRun 'fpm init' to create a project, or use 'fpm tree --system' for system packages")
	}

	if invert {
		printReverseTree(lf)
	} else {
		printTree(lf, depth)
	}
	return nil
}

func printGraphTree(depth int, invert bool) error {
	graph := depgraph.Load("global")
	if len(graph.Packages) == 0 {
		return fmt.Errorf("no dependency graph found. Install packages with 'fpm install -s' first")
	}
	return printDepGraphTree(graph, depth, invert)
}

func printDepGraphTree(graph *depgraph.Graph, maxDepth int, invert bool) error {
	if invert {
		// Reverse tree: show who depends on each package
		for name := range graph.Packages {
			parents := graph.DependentsOf(name)
			if len(parents) > 0 {
				node := graph.Packages[name]
				fmt.Printf("\033[1m%s\033[0m \033[36m%s\033[0m\n", name, node.Version)
				for _, p := range parents {
					fmt.Printf("  └── required by \033[1m%s\033[0m\n", p)
				}
			}
		}
		return nil
	}

	// Forward tree: show requested packages and their deps
	for name, node := range graph.Packages {
		if !node.Requested {
			continue
		}
		label := "\033[32m●\033[0m"
		fmt.Printf("%s \033[1m%s\033[0m \033[36m%s\033[0m\n", label, name, node.Version)
		printGraphNode(graph, node.Dependencies, 1, maxDepth, make(map[string]bool))
	}
	return nil
}

func printGraphNode(graph *depgraph.Graph, deps []string, depth, maxDepth int, visited map[string]bool) {
	if maxDepth > 0 && depth >= maxDepth {
		return
	}
	for i, dep := range deps {
		if visited[dep] {
			continue
		}
		visited[dep] = true

		prefix := "├── "
		if i == len(deps)-1 {
			prefix = "└── "
		}
		indent := strings.Repeat("│   ", depth-1) + prefix

		version := ""
		var subDeps []string
		if node, ok := graph.Packages[dep]; ok {
			version = node.Version
			subDeps = node.Dependencies
		}
		fmt.Printf("%s%s \033[2m%s\033[0m\n", indent, dep, version)

		if len(subDeps) > 0 {
			printGraphNode(graph, subDeps, depth+1, maxDepth, visited)
		}
		delete(visited, dep)
	}
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
