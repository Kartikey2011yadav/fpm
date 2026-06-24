package config

import (
	"os"
	"path/filepath"
	"runtime"
)

func CacheDir() string {
	if v := os.Getenv("FPM_CACHE_DIR"); v != "" {
		return v
	}
	if IsMultiUserMode() {
		return SharedCacheDir()
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Caches", "fpm")
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "fpm", "cache")
		}
		return filepath.Join(homeDir(), "AppData", "Local", "fpm", "cache")
	default:
		if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
			return filepath.Join(v, "fpm")
		}
		return filepath.Join(homeDir(), ".cache", "fpm")
	}
}

func SharedCacheDir() string {
	if v := os.Getenv("FPM_SHARED_CACHE_DIR"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Caches/fpm"
	case "windows":
		if v := os.Getenv("PROGRAMDATA"); v != "" {
			return filepath.Join(v, "fpm", "cache")
		}
		return `C:\ProgramData\fpm\cache`
	default:
		return "/var/cache/fpm"
	}
}

func IsMultiUserMode() bool {
	if v := os.Getenv("FPM_MODE"); v == "multi-user" {
		return true
	}
	modeFile := filepath.Join(SystemConfigDir(), "mode")
	if data, err := os.ReadFile(modeFile); err == nil {
		return string(data) == "multi-user"
	}
	return false
}

func DataDir() string {
	if v := os.Getenv("FPM_DATA_DIR"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Application Support", "fpm")
	case "windows":
		if v := os.Getenv("LOCALAPPDATA"); v != "" {
			return filepath.Join(v, "fpm", "data")
		}
		return filepath.Join(homeDir(), "AppData", "Local", "fpm", "data")
	default:
		if v := os.Getenv("XDG_DATA_HOME"); v != "" {
			return filepath.Join(v, "fpm")
		}
		return filepath.Join(homeDir(), ".local", "share", "fpm")
	}
}

func ConfigDir() string {
	if v := os.Getenv("FPM_CONFIG_DIR"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(homeDir(), "Library", "Application Support", "fpm")
	case "windows":
		if v := os.Getenv("APPDATA"); v != "" {
			return filepath.Join(v, "fpm")
		}
		return filepath.Join(homeDir(), "AppData", "Roaming", "fpm")
	default:
		if v := os.Getenv("XDG_CONFIG_HOME"); v != "" {
			return filepath.Join(v, "fpm")
		}
		return filepath.Join(homeDir(), ".config", "fpm")
	}
}

func SystemConfigDir() string {
	switch runtime.GOOS {
	case "darwin":
		return "/Library/Application Support/fpm"
	case "windows":
		if v := os.Getenv("PROGRAMDATA"); v != "" {
			return filepath.Join(v, "fpm")
		}
		return `C:\ProgramData\fpm`
	default:
		// Check XDG_CONFIG_DIRS first (like uv), fallback to /etc/fpm
		if v := os.Getenv("XDG_CONFIG_DIRS"); v != "" {
			for _, dir := range filepath.SplitList(v) {
				candidate := filepath.Join(dir, "fpm")
				if _, err := os.Stat(filepath.Join(candidate, "config.toml")); err == nil {
					return candidate
				}
			}
		}
		return "/etc/fpm"
	}
}

func PythonInstallDir() string {
	if v := os.Getenv("FPM_PYTHON_INSTALL_DIR"); v != "" {
		return v
	}
	return filepath.Join(DataDir(), "python")
}

func ToolsDir() string {
	if v := os.Getenv("FPM_TOOL_DIR"); v != "" {
		return v
	}
	return filepath.Join(DataDir(), "tools")
}

func BinDir() string {
	if v := os.Getenv("FPM_TOOL_BIN_DIR"); v != "" {
		return v
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(DataDir(), "bin")
	default:
		return filepath.Join(homeDir(), ".local", "bin")
	}
}

func CredentialsDir() string {
	if v := os.Getenv("FPM_CREDENTIALS_DIR"); v != "" {
		return v
	}
	return filepath.Join(DataDir(), "credentials")
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}
