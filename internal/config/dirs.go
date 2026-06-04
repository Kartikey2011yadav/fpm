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
		return "/etc/fpm"
	}
}

func PythonInstallDir() string {
	return filepath.Join(DataDir(), "python")
}

func ToolsDir() string {
	return filepath.Join(DataDir(), "tools")
}

func BinDir() string {
	return filepath.Join(DataDir(), "bin")
}

func homeDir() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return "."
	}
	return h
}
