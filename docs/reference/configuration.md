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

[network]
allow-insecure-host = ["internal-pypi.corp.example.com"]
# system-certs = true       # force platform certificate verifier
# client-cert = "/path/to/cert.pem"
# client-key = "/path/to/key.pem"
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

| Variable                   | Description                         | Default                   |
| -------------------------- | ----------------------------------- | ------------------------- |
| `FPM_CACHE_DIR`            | Cache directory                     | `~/.cache/fpm`            |
| `FPM_DATA_DIR`             | Data directory                      | `~/.local/share/fpm`      |
| `FPM_CONFIG_DIR`           | Config directory                    | `~/.config/fpm`           |
| `FPM_INDEX_URL`            | Override default index              | `https://pypi.org/simple` |
| `FPM_INDEX_TOKEN`          | Auth token for indexes              | —                         |
| `FPM_INDEX_USERNAME`       | Auth username                       | —                         |
| `FPM_INDEX_PASSWORD`       | Auth password                       | —                         |
| `FPM_CROSS_MANAGER_POLICY` | Cross-manager behavior              | `ask`                     |
| `FPM_PUBLISH_TOKEN`        | PyPI upload token                   | —                         |
| `FPM_ALLOW_INSECURE_HOST`  | Skip TLS for these hosts (csv)      | —                         |
| `FPM_INSECURE`             | Disable ALL TLS verification (`1`)  | —                         |
| `FPM_SYSTEM_CERTS`         | Force platform certificate store    | —                         |
| `SSL_CERT_FILE`            | Custom CA bundle (overrides system) | —                         |
| `SSL_CERT_DIR`             | Directory of CA certs               | —                         |
| `SSL_CLIENT_CERT`          | mTLS client certificate             | —                         |
| `NO_COLOR`                 | Disable colored output              | —                         |

## Platform Directories

| Platform | Cache                      | Data                                | Config                              |
| -------- | -------------------------- | ----------------------------------- | ----------------------------------- |
| Linux    | `~/.cache/fpm`             | `~/.local/share/fpm`                | `~/.config/fpm`                     |
| macOS    | `~/Library/Caches/fpm`     | `~/Library/Application Support/fpm` | `~/Library/Application Support/fpm` |
| Windows  | `%LOCALAPPDATA%\fpm\cache` | `%LOCALAPPDATA%\fpm\data`           | `%APPDATA%\fpm`                     |

### Subdirectories

| Directory         | Location                                        | Purpose                                |
| ----------------- | ----------------------------------------------- | -------------------------------------- |
| Cache root        | `$FPM_CACHE_DIR`                                | All cached data                        |
| CAS               | `<cache>/cas/sha256/`                           | Extracted packages (content-addressed) |
| Wheels            | `<cache>/wheels/`                               | Downloaded .whl files                  |
| HTTP cache        | `<cache>/http/`                                 | PyPI metadata responses                |
| Refs              | `<cache>/refs/`                                 | Reference tracking (env↔package)       |
| Python installs   | `$FPM_PYTHON_INSTALL_DIR` or `<data>/python/`   | Managed Python versions                |
| Tool environments | `$FPM_TOOL_DIR` or `<data>/tools/`              | Isolated tool venvs                    |
| Tool binaries     | `$FPM_TOOL_BIN_DIR` or `~/.local/bin/`          | Tool executables (in PATH)             |
| Credentials       | `$FPM_CREDENTIALS_DIR` or `<data>/credentials/` | Stored auth credentials                |

### Directory Override Environment Variables

| Variable                 | Default              | Purpose                      |
| ------------------------ | -------------------- | ---------------------------- |
| `FPM_CACHE_DIR`          | `~/.cache/fpm`       | Cache root                   |
| `FPM_DATA_DIR`           | `~/.local/share/fpm` | Data root                    |
| `FPM_CONFIG_DIR`         | `~/.config/fpm`      | Config root                  |
| `FPM_PYTHON_INSTALL_DIR` | `<data>/python`      | Managed Python installations |
| `FPM_TOOL_DIR`           | `<data>/tools`       | Tool venv storage            |
| `FPM_TOOL_BIN_DIR`       | `~/.local/bin`       | Tool executables             |
| `FPM_CREDENTIALS_DIR`    | `<data>/credentials` | Stored credentials           |
