# config

Configuration parsing and management with hierarchical override support.

## Key Types

- `Config` — top-level configuration struct
- `ImmutableConfig` — packages that must never change version
- `IndexConfig` — PyPI/private registry URLs

## Configuration Hierarchy

1. Built-in defaults (`DefaultConfig()`)
2. System config (`/etc/fpm/config.toml` or platform equivalent)
3. User config (`~/.config/fpm/config.toml`)
4. Project config (`./fpm.toml`)
5. Environment variables (`FPM_*`)
6. CLI flags (highest priority)

## Files

- `config.go` — `Config` struct, `Load()`, `FindProjectRoot()`
- `dirs.go` — platform-specific directory paths (XDG, macOS Library, Windows
  AppData)

## Platform Directories

| Platform | Cache                      | Data                                | Config                              |
| -------- | -------------------------- | ----------------------------------- | ----------------------------------- |
| Linux    | `~/.cache/fpm`             | `~/.local/share/fpm`                | `~/.config/fpm`                     |
| macOS    | `~/Library/Caches/fpm`     | `~/Library/Application Support/fpm` | `~/Library/Application Support/fpm` |
| Windows  | `%LOCALAPPDATA%\fpm\cache` | `%LOCALAPPDATA%\fpm\data`           | `%APPDATA%\fpm`                     |

All paths are overridable via `FPM_CACHE_DIR`, `FPM_DATA_DIR`, `FPM_CONFIG_DIR`.
