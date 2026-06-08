package depgraph

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/kartikeyyadav/fpm/internal/config"
)

// PackageNode represents a package in the dependency graph.
type PackageNode struct {
	Name         string   `json:"name"`
	Version      string   `json:"version"`
	Requested    bool     `json:"requested"`
	Dependencies []string `json:"dependencies"`
	InstalledAt  string   `json:"installed_at"`
}

// Graph tracks all fpm-managed packages and their relationships.
type Graph struct {
	Packages map[string]*PackageNode `json:"packages"`
	mu       sync.RWMutex
}

func graphPath(envPath string) string {
	if envPath == "" || strings.HasPrefix(envPath, "global") {
		return filepath.Join(config.DataDir(), "depgraph.json")
	}
	return filepath.Join(envPath, ".fpm-depgraph.json")
}

// Load reads the dependency graph from disk. Returns empty graph if not found.
func Load(envPath string) *Graph {
	g := &Graph{Packages: make(map[string]*PackageNode)}

	path := graphPath(envPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return g
	}
	json.Unmarshal(data, g)
	if g.Packages == nil {
		g.Packages = make(map[string]*PackageNode)
	}
	return g
}

// Save writes the dependency graph to disk.
func (g *Graph) Save(envPath string) error {
	g.mu.RLock()
	defer g.mu.RUnlock()

	path := graphPath(envPath)
	os.MkdirAll(filepath.Dir(path), 0755)

	data, err := json.MarshalIndent(g, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// AddRequested records a package explicitly installed by the user.
func (g *Graph) AddRequested(name, version string, deps []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.Packages[name] = &PackageNode{
		Name:         name,
		Version:      version,
		Requested:    true,
		Dependencies: deps,
		InstalledAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

// AddTransitive records a package installed as a dependency.
func (g *Graph) AddTransitive(name, version string, deps []string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Don't downgrade from requested to transitive
	if existing, ok := g.Packages[name]; ok && existing.Requested {
		existing.Version = version
		existing.Dependencies = deps
		return
	}

	g.Packages[name] = &PackageNode{
		Name:         name,
		Version:      version,
		Requested:    false,
		Dependencies: deps,
		InstalledAt:  time.Now().UTC().Format(time.RFC3339),
	}
}

// Remove deletes a package from the graph.
func (g *Graph) Remove(name string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.Packages, name)
}

// IsRequested returns true if the package was explicitly installed by user.
func (g *Graph) IsRequested(name string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	if node, ok := g.Packages[name]; ok {
		return node.Requested
	}
	return false
}

// Orphans returns packages that are:
// 1. Not requested (transitive only)
// 2. Not depended on by any other package in the graph
func (g *Graph) Orphans() []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	// Build set of all packages that are depended on
	needed := make(map[string]bool)
	for _, node := range g.Packages {
		for _, dep := range node.Dependencies {
			needed[dep] = true
		}
	}

	// Find packages that are transitive AND not needed
	var orphans []string
	for name, node := range g.Packages {
		if !node.Requested && !needed[name] {
			orphans = append(orphans, name)
		}
	}
	return orphans
}

// DependentsOf returns packages that depend on the given package.
func (g *Graph) DependentsOf(name string) []string {
	g.mu.RLock()
	defer g.mu.RUnlock()

	var dependents []string
	for _, node := range g.Packages {
		for _, dep := range node.Dependencies {
			if dep == name {
				dependents = append(dependents, node.Name)
				break
			}
		}
	}
	return dependents
}
