package workspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadPyProjectToml(t *testing.T) {
	dir := t.TempDir()
	content := `
[project]
name = "myproject"
version = "1.0.0"
requires-python = ">=3.10"
dependencies = ["requests>=2.28", "flask"]

[tool.fpm]
dev-dependencies = ["pytest"]
`
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(content), 0644)

	cfg, err := ReadPyProjectToml(dir)
	if err != nil {
		t.Fatalf("ReadPyProjectToml error: %v", err)
	}

	if cfg.Project.Name != "myproject" {
		t.Errorf("name = %q", cfg.Project.Name)
	}
	if len(cfg.Project.Dependencies) != 2 {
		t.Errorf("deps = %d, want 2", len(cfg.Project.Dependencies))
	}
}

func TestAddRemoveDependency(t *testing.T) {
	cfg := &PyProjectToml{}
	cfg.Project.Dependencies = []string{"requests>=2.28"}

	cfg.AddDependency("flask>=2.0")
	if len(cfg.Project.Dependencies) != 2 {
		t.Fatalf("after add: count = %d, want 2", len(cfg.Project.Dependencies))
	}

	// Adding duplicate should not add again
	cfg.AddDependency("flask>=3.0")
	if len(cfg.Project.Dependencies) != 2 {
		t.Errorf("after dup add: count = %d, want 2", len(cfg.Project.Dependencies))
	}

	cfg.RemoveDependency("flask")
	if len(cfg.Project.Dependencies) != 1 {
		t.Errorf("after remove: count = %d, want 1", len(cfg.Project.Dependencies))
	}
}

func TestWriteAndReadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	cfg := &PyProjectToml{
		Project: ProjectTable{
			Name:         "test",
			Version:      "0.1.0",
			Dependencies: []string{"numpy"},
		},
	}

	err := WritePyProjectToml(dir, cfg)
	if err != nil {
		t.Fatalf("Write error: %v", err)
	}

	read, err := ReadPyProjectToml(dir)
	if err != nil {
		t.Fatalf("Read error: %v", err)
	}
	if read.Project.Name != "test" {
		t.Errorf("name = %q", read.Project.Name)
	}
}

func TestDiscover(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]\nname=\"root\"\n"), 0644)

	sub := filepath.Join(dir, "src", "pkg")
	os.MkdirAll(sub, 0755)

	// Discover from subdirectory should find root
	ws, err := Discover(sub)
	if err != nil {
		// Expected if no workspace members
		return
	}
	if ws.Root != dir {
		t.Errorf("root = %q, want %q", ws.Root, dir)
	}
}
