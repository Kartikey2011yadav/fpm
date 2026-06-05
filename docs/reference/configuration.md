# Configuration

## fpm.toml

Project-level configuration file:

```toml
[project]
name = "my-project"
version = "0.1.0"
description = "My Python project"
requires-python = ">=3.10"
dependencies = [
    "requests>=2.28",
    "numpy>=1.24",
]

[tool.fpm]
cross-manager-policy = "ask"  # ask | install | skip
link-mode = "auto"            # auto | hardlink | copy | symlink
concurrency = 50              # parallel downloads

[tool.fpm.dependency-groups]
test = ["pytest>=7.0", "coverage"]
docs = ["sphinx", "furo"]
dev = ["ruff", "mypy"]

[tool.fpm.overrides]
numpy = "==1.24.0"  # force version regardless of dep requests

[python]
version = "3.11"
preference = "managed"  # managed | system | only-managed

[cache]
dir = "/custom/cache/path"

[immutable]
packages = [
    { name = "numpy", version = "1.24.0" },
    { name = "torch", version = "2.1.0" },
]

[[index]]
name = "pypi"
url = "https://pypi.org/simple"
default = true

[[index]]
name = "private"
url = "https://pypi.company.com/simple"
```

## Configuration Hierarchy

Priority (highest wins):

1. CLI flags (`--global`, `--verbose`, etc.)
2. Environment variables (`FPM_*`)
3. Project config (`./fpm.toml`)
4. User config (`~/.config/fpm/config.toml`)
5. System config (`/etc/fpm/config.toml`)
6. Built-in defaults

## Environment Variables

| Variable                   | Description            | Default                   |
| -------------------------- | ---------------------- | ------------------------- |
| `FPM_CACHE_DIR`            | Cache directory        | `~/.cache/fpm`            |
| `FPM_DATA_DIR`             | Data directory         | `~/.local/share/fpm`      |
| `FPM_CONFIG_DIR`           | Config directory       | `~/.config/fpm`           |
| `FPM_INDEX_URL`            | Override default index | `https://pypi.org/simple` |
| `FPM_INDEX_TOKEN`          | Auth token for indexes | —                         |
| `FPM_INDEX_USERNAME`       | Auth username          | —                         |
| `FPM_INDEX_PASSWORD`       | Auth password          | —                         |
| `FPM_CROSS_MANAGER_POLICY` | Cross-manager behavior | `ask`                     |
| `FPM_PUBLISH_TOKEN`        | PyPI upload token      | —                         |
| `NO_COLOR`                 | Disable colored output | —                         |

## Platform Directories

| Platform | Cache                      | Data                                | Config                              |
| -------- | -------------------------- | ----------------------------------- | ----------------------------------- |
| Linux    | `~/.cache/fpm`             | `~/.local/share/fpm`                | `~/.config/fpm`                     |
| macOS    | `~/Library/Caches/fpm`     | `~/Library/Application Support/fpm` | `~/Library/Application Support/fpm` |
| Windows  | `%LOCALAPPDATA%\fpm\cache` | `%LOCALAPPDATA%\fpm\data`           | `%APPDATA%\fpm`                     |
