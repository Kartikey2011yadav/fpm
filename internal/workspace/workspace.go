package workspace

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Workspace struct {
	Root    string
	Members []Member
	Config  PyProjectToml
}

type Member struct {
	Name string
	Path string
}

type PyProjectToml struct {
	Project ProjectTable `toml:"project"`
	Tool    ToolTable    `toml:"tool"`
}

type ProjectTable struct {
	Name           string   `toml:"name"`
	Version        string   `toml:"version"`
	Description    string   `toml:"description"`
	RequiresPython string   `toml:"requires-python"`
	Dependencies   []string `toml:"dependencies"`
	License        string   `toml:"license"`
	Authors        []Author `toml:"authors"`
}

type Author struct {
	Name  string `toml:"name"`
	Email string `toml:"email"`
}

type ToolTable struct {
	Fpm FpmToolTable `toml:"fpm"`
}

type FpmToolTable struct {
	DevDependencies []string       `toml:"dev-dependencies"`
	Workspace       WorkspaceTable `toml:"workspace"`
}

type WorkspaceTable struct {
	Members []string `toml:"members"`
	Exclude []string `toml:"exclude"`
}

func Discover(startDir string) (*Workspace, error) {
	dir := startDir
	for {
		pyproject := filepath.Join(dir, "pyproject.toml")
		if _, err := os.Stat(pyproject); err == nil {
			ws, err := loadWorkspace(dir, pyproject)
			if err == nil {
				return ws, nil
			}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return nil, os.ErrNotExist
}

func loadWorkspace(root, pyprojectPath string) (*Workspace, error) {
	var cfg PyProjectToml
	_, err := toml.DecodeFile(pyprojectPath, &cfg)
	if err != nil {
		return nil, err
	}

	ws := &Workspace{
		Root:   root,
		Config: cfg,
	}

	// Discover workspace members
	for _, pattern := range cfg.Tool.Fpm.Workspace.Members {
		matches, err := filepath.Glob(filepath.Join(root, pattern))
		if err != nil {
			continue
		}
		for _, m := range matches {
			memberPyproject := filepath.Join(m, "pyproject.toml")
			if _, err := os.Stat(memberPyproject); err == nil {
				name := filepath.Base(m)
				ws.Members = append(ws.Members, Member{
					Name: name,
					Path: m,
				})
			}
		}
	}

	return ws, nil
}

func ReadPyProjectToml(dir string) (*PyProjectToml, error) {
	path := filepath.Join(dir, "pyproject.toml")
	var cfg PyProjectToml
	_, err := toml.DecodeFile(path, &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}

func WritePyProjectToml(dir string, cfg *PyProjectToml) error {
	path := filepath.Join(dir, "pyproject.toml")
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	encoder := toml.NewEncoder(f)
	encoder.Indent = ""
	return encoder.Encode(cfg)
}

func (p *PyProjectToml) AddDependency(dep string) {
	for _, existing := range p.Project.Dependencies {
		if extractPkgName(existing) == extractPkgName(dep) {
			return
		}
	}
	p.Project.Dependencies = append(p.Project.Dependencies, dep)
}

func (p *PyProjectToml) RemoveDependency(name string) {
	normalized := strings.ToLower(strings.ReplaceAll(name, "-", "_"))
	filtered := p.Project.Dependencies[:0]
	for _, dep := range p.Project.Dependencies {
		if extractPkgName(dep) != normalized {
			filtered = append(filtered, dep)
		}
	}
	p.Project.Dependencies = filtered
}

func extractPkgName(dep string) string {
	dep = strings.TrimSpace(dep)
	for i, ch := range dep {
		if ch == '>' || ch == '<' || ch == '=' || ch == '!' || ch == '~' || ch == '[' || ch == ';' || ch == ' ' {
			dep = dep[:i]
			break
		}
	}
	return strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(dep, "-", "_"), ".", "_"))
}
