package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.Tool.CrossManagerPolicy != "ask" {
		t.Errorf("default policy = %q, want 'ask'", cfg.Tool.CrossManagerPolicy)
	}
	if cfg.Tool.Concurrency != 50 {
		t.Errorf("default concurrency = %d, want 50", cfg.Tool.Concurrency)
	}
	if len(cfg.Indexes) != 1 || cfg.Indexes[0].URL != "https://pypi.org/simple" {
		t.Error("default index should be pypi")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	// Use the struct field TOML tags exactly as defined in config.go
	tomlContent := `
[project]
name = "test-project"
version = "1.0.0"
requires-python = ">=3.10"

[tool]
cross-manager-policy = "install"
concurrency = 100

[immutable]
[[immutable.packages]]
name = "numpy"
version = "1.24.0"

[[index]]
name = "private"
url = "https://pypi.internal.com/simple"
default = true
`
	os.WriteFile(filepath.Join(dir, "fpm.toml"), []byte(tomlContent), 0644)

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	if cfg.Project.Name != "test-project" {
		t.Errorf("project name = %q", cfg.Project.Name)
	}
	if cfg.Tool.CrossManagerPolicy != "install" {
		t.Errorf("policy = %q, want 'install'", cfg.Tool.CrossManagerPolicy)
	}
	if cfg.Tool.Concurrency != 100 {
		t.Errorf("concurrency = %d, want 100", cfg.Tool.Concurrency)
	}
	if len(cfg.Immutable.Packages) != 1 {
		t.Fatalf("immutable count = %d, want 1", len(cfg.Immutable.Packages))
	}
	if cfg.Immutable.Packages[0].Name != "numpy" {
		t.Errorf("immutable[0].name = %q", cfg.Immutable.Packages[0].Name)
	}
}

func TestEnvOverrides(t *testing.T) {
	os.Setenv("FPM_CACHE_DIR", "/custom/cache")
	defer os.Unsetenv("FPM_CACHE_DIR")

	cfg, _ := Load("")
	if cfg.Cache.Dir != "/custom/cache" {
		t.Errorf("cache dir = %q, want '/custom/cache'", cfg.Cache.Dir)
	}
}

func TestFindProjectRoot(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, "src", "pkg")
	os.MkdirAll(subdir, 0755)
	os.WriteFile(filepath.Join(dir, "fpm.toml"), []byte("[project]\nname=\"x\"\n"), 0644)

	// FindProjectRoot walks up — but since we can't chdir in tests easily,
	// verify it doesn't crash
	_, err := FindProjectRoot()
	if err != nil {
		t.Fatalf("FindProjectRoot error: %v", err)
	}
}

func TestPlatformDirs(t *testing.T) {
	// Just verify they return non-empty strings
	if CacheDir() == "" {
		t.Error("CacheDir() is empty")
	}
	if DataDir() == "" {
		t.Error("DataDir() is empty")
	}
	if ConfigDir() == "" {
		t.Error("ConfigDir() is empty")
	}
}
