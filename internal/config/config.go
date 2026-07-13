package config

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Project   ProjectConfig   `toml:"project"`
	Tool      ToolConfig      `toml:"tool"`
	Immutable ImmutableConfig `toml:"immutable"`
	Python    PythonConfig    `toml:"python"`
	Cache     CacheConfig     `toml:"cache"`
	Indexes   []IndexConfig   `toml:"index"`
	Network   NetworkConfig   `toml:"network"`
	Log       LogConfig       `toml:"log"`
}

type ProjectConfig struct {
	Name            string   `toml:"name"`
	Version         string   `toml:"version"`
	Description     string   `toml:"description"`
	RequiresPython  string   `toml:"requires-python"`
	Dependencies    []string `toml:"dependencies"`
	DevDependencies []string `toml:"dev-dependencies"`
}

type ToolConfig struct {
	CrossManagerPolicy string `toml:"cross-manager-policy"` // "ask", "install", "skip"
	LinkMode           string `toml:"link-mode"`            // "auto", "hardlink", "copy", "symlink"
	Concurrency        int    `toml:"concurrency"`
	Mode               string `toml:"mode"` // "single-user" (default), "multi-user"
}

type ImmutableConfig struct {
	Packages []ImmutablePackage `toml:"packages"`
}

type ImmutablePackage struct {
	Name    string `toml:"name"`
	Version string `toml:"version"`
}

type PythonConfig struct {
	Version    string `toml:"version"`
	Preference string `toml:"preference"` // "managed", "system", "only-managed"
}

type CacheConfig struct {
	Dir string `toml:"dir"`
}

type IndexConfig struct {
	Name    string `toml:"name"`
	URL     string `toml:"url"`
	Default bool   `toml:"default"`
}

type NetworkConfig struct {
	AllowInsecureHost []string `toml:"allow-insecure-host"`
	SystemCerts       bool     `toml:"system-certs"`
	ClientCert        string   `toml:"client-cert"`
	ClientKey         string   `toml:"client-key"`
}

type LogConfig struct {
	Level string `toml:"level"` // "debug", "info", "warn", "error", "off"
	File  string `toml:"file"`  // path to log file (empty = no file logging)
}

func LogDir() string {
	return filepath.Join(DataDir(), "logs")
}

func DefaultLogFile() string {
	return filepath.Join(LogDir(), "fpm.log")
}

func DefaultConfig() *Config {
	return &Config{
		Tool: ToolConfig{
			CrossManagerPolicy: "ask",
			LinkMode:           "auto",
			Concurrency:        50,
		},
		Cache: CacheConfig{
			Dir: CacheDir(),
		},
		Indexes: []IndexConfig{
			{Name: "pypi", URL: "https://pypi.org/simple", Default: true},
		},
	}
}

func Load(projectDir string) (*Config, error) {
	cfg := DefaultConfig()

	// Layer 1: System config
	systemPath := filepath.Join(SystemConfigDir(), "config.toml")
	_ = loadFile(systemPath, cfg)

	// Layer 2: User config
	userPath := filepath.Join(ConfigDir(), "config.toml")
	_ = loadFile(userPath, cfg)

	// Layer 3: Project config (fpm.toml)
	if projectDir != "" {
		projectPath := filepath.Join(projectDir, "fpm.toml")
		_ = loadFile(projectPath, cfg)
	}

	// Layer 4: Environment variables
	applyEnvOverrides(cfg)

	return cfg, nil
}

func LoadFromCwd() (*Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return DefaultConfig(), nil
	}
	return Load(cwd)
}

func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return toml.Unmarshal(data, cfg)
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("FPM_CACHE_DIR"); v != "" {
		cfg.Cache.Dir = v
	}
	if v := os.Getenv("FPM_INDEX_URL"); v != "" {
		cfg.Indexes = []IndexConfig{{Name: "custom", URL: v, Default: true}}
	}
	if v := os.Getenv("FPM_CROSS_MANAGER_POLICY"); v != "" {
		cfg.Tool.CrossManagerPolicy = v
	}
	if v := os.Getenv("FPM_ALLOW_INSECURE_HOST"); v != "" {
		var hosts []string
		for _, h := range strings.Split(v, ",") {
			h = strings.TrimSpace(h)
			if h != "" {
				hosts = append(hosts, h)
			}
		}
		cfg.Network.AllowInsecureHost = append(cfg.Network.AllowInsecureHost, hosts...)
	}
	if v := os.Getenv("FPM_SYSTEM_CERTS"); v == "1" || v == "true" {
		cfg.Network.SystemCerts = true
	}
	if v := os.Getenv("SSL_CLIENT_CERT"); v != "" {
		cfg.Network.ClientCert = v
	}
}

func FindProjectRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "fpm.toml")); err == nil {
			return dir, nil
		}
		if _, err := os.Stat(filepath.Join(dir, "pyproject.toml")); err == nil {
			return dir, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	cwd, _ := os.Getwd()
	return cwd, nil
}
