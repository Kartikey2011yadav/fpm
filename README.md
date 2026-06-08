# fpm — Fast Package Manager

A high-performance, space-conservative Python package manager written in Go.
Designed to coexist with pip, uv, conda, poetry, and other managers while
providing unique capabilities like environment snapshots, immutable package
pinning, and content-addressable caching.

## Features

- **Blazing fast** — parallel downloads, content-addressable cache with
  hardlinking
- **Coexists with everything** — works alongside pip, uv, conda, poetry, pdm
  without conflicts
- **Environment snapshots** — version your environment like git; restore any
  previous state instantly
- **Cross-manager awareness** — detects packages from all managers, warns about
  conflicts
- **Immutable pinning** — lock critical packages that must never change version
- **Space efficient** — content-addressable cache with reflink/hardlink, zero
  duplication
- **Python version management** — install and switch between Python versions
  per-project
- **Universal lockfile** — cross-platform lockfile for reproducible installs
- **Global/local scope** — `--system` for system-wide, local by default (safe)
- **Works everywhere** — Linux, macOS, Windows, servers, Jupyter, CI/CD

## Installation

### Direct download (recommended)

```bash
# macOS / Linux
curl -fsSL https://github.com/Kartikey2011yadav/fpm/releases/latest/download/fpm-$(uname -s)-$(uname -m) -o fpm
chmod +x fpm
sudo mv fpm /usr/local/bin/

# Or with install script
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | sh
```

### Homebrew (macOS/Linux)

```bash
brew install kartikeyyadav/tap/fpm
```

### pip

```bash
pip install fpm-pkg
```

### go install

```bash
go install github.com/kartikeyyadav/fpm/cmd/fpm@latest
```

### From source

```bash
git clone https://github.com/Kartikey2011yadav/fpm.git
cd fpm
make build
./bin/fpm version
```

### Custom installation paths

During installation, you can override default directories:

```bash
# Set custom paths via environment variables
export FPM_CACHE_DIR=/custom/cache/path
export FPM_DATA_DIR=/custom/data/path
export FPM_CONFIG_DIR=/custom/config/path

# Or configure after install via fpm.toml
[cache]
dir = "/my/custom/cache"
```

## Quick Start

```bash
# Create a new project
fpm init myproject
cd myproject

# Add dependencies
fpm install requests numpy pandas

# Run your code
fpm run python main.py

# Take a snapshot of your environment
fpm snapshot create "initial setup"

# See what's installed (from ALL managers)
fpm list
```

## CLI Reference

| Command                     | Description                                    |
| --------------------------- | ---------------------------------------------- |
| `fpm init [path]`           | Create a new Python project                    |
| `fpm install <pkg>`         | Install packages (aliases: `add`)              |
| `fpm remove <pkg>`          | Remove packages (aliases: `uninstall`, `rm`)   |
| `fpm list`                  | List installed packages (alias: `ls`)          |
| `fpm sync`                  | Sync environment from lockfile                 |
| `fpm lock`                  | Generate/update lockfile                       |
| `fpm run <cmd>`             | Run command in managed environment             |
| `fpm tree`                  | Display dependency tree                        |
| `fpm audit`                 | Scan for known vulnerabilities                 |
| `fpm venv [path]`           | Create virtual environment                     |
| `fpm python list`           | List Python versions                           |
| `fpm python install <ver>`  | Install Python version                         |
| `fpm python use <ver>`      | Switch Python (local or `--system`)            |
| `fpm snapshot create [msg]` | Capture environment state                      |
| `fpm snapshot list`         | Show snapshot history                          |
| `fpm snapshot restore <id>` | Restore previous state                         |
| `fpm snapshot diff <id>`    | Compare snapshots                              |
| `fpm cache size`            | Show cache usage                               |
| `fpm cache gc`              | Remove unreferenced packages                   |
| `fpm cache list-unused`     | Show orphaned cache entries                    |
| `fpm build`                 | Build wheel/sdist                              |
| `fpm publish`               | Upload to PyPI                                 |
| `fpm pip list`              | List all packages (all managers)               |
| `fpm pip freeze`            | Output requirements format                     |
| `fpm tool run <pkg>`        | Run tool in ephemeral env                      |
| `fpm tool install <pkg>`    | Install tool permanently                       |
| `fpm version`               | Show version (also: `fpm --version`, `fpm -v`) |
| `fpm repair`                | Diagnose and fix installation issues           |
| `fpm config show`           | Show current configuration and paths           |
| `fpm config set <k> <v>`    | Modify user configuration                      |
| `fpm config init`           | Create default user config file                |

### Global Flags

| Flag                           | Description                                |
| ------------------------------ | ------------------------------------------ |
| `-v`, `--version`              | Print version and exit                     |
| `-s`, `--system`               | Install into system Python instead of venv |
| `--verbose`                    | Enable verbose output                      |
| `-q`, `--quiet`                | Suppress output except errors              |
| `--json`                       | Output in JSON format                      |
| `--no-progress`                | Disable progress bars                      |
| `--color <mode>`               | Control colors (auto/always/never)         |
| `--allow-insecure-host <host>` | Skip TLS verification for specific hosts   |
| `--log-level <level>`          | Logging: debug, info, warn, error, off     |

## Environment Snapshots

Think of it as **git for your Python environment**:

```bash
# Capture current state (tracks ALL managers: fpm, pip, uv, conda, etc.)
fpm snapshot create "before ML experiment"

# Make changes...
fpm install torch transformers

# See what changed
fpm snapshot diff 20260605-094738

# Restore previous state instantly (fpm packages from cache, no re-download)
fpm snapshot restore 20260605-094738
#   ✓ Restored 12 fpm packages from cache
#   ↓ Re-downloaded 1 package (cache was cleaned)
#   ⚠ pip's scipy: expected 1.10.0, found 1.11.0
```

Snapshots are **scoped per-environment** — each venv/project has its own
history.

## Cross-Manager Coexistence

fpm detects packages from: **pip, uv, conda, poetry, pdm, system**

```bash
# See everything in your environment
fpm list --all
# Package                  Version    Manager  Location
# numpy                    1.24.0     pip      /usr/lib/python3.11/...
# requests                 2.31.0     fpm      .venv/lib/python3.11/...
# pandas                   2.0.0      conda    /opt/conda/lib/python3.11/...
# black                    23.1.0     uv       .venv/lib/python3.11/...

# When installing, fpm checks other managers first:
fpm install numpy
#   numpy 1.24.0 is already installed via pip — skipping download

# If versions conflict, you get a choice:
fpm install numpy==2.0.0
#   numpy 1.24.0 is installed via pip, but you're requesting 2.0.0.
#   [1] Skip  [2] Install anyway  [3] Abort
```

## Python Version Management

```bash
# Install multiple Python versions
fpm python install 3.11 3.12 3.13

# Use a specific version for this project (doesn't affect global)
fpm python use 3.11

# Create a venv with a different Python than system
fpm venv --python 3.11
# → Auto-downloads Python 3.11 if not installed

# Switch global default
fpm python use 3.12 --system
```

## Configuration

fpm uses `fpm.toml` for project configuration:

```toml
[project]
name = "my-project"
requires-python = ">=3.10"
dependencies = ["requests>=2.28", "numpy"]

[tool.fpm]
cross-manager-policy = "ask"  # ask, install, skip
link-mode = "auto"            # auto, hardlink, copy, symlink
concurrency = 50

[immutable]
packages = [
    { name = "numpy", version = "1.24.0" },
]

[cache]
dir = "/custom/cache/path"  # override default cache location

[log]
level = "off"  # debug, info, warn, error, off
# file = ""    # custom log file path
```

### Configuration hierarchy (later overrides earlier):

1. Built-in defaults
2. System config (`/etc/fpm/config.toml`)
3. User config (`~/.config/fpm/config.toml`)
4. Project config (`./fpm.toml`)
5. Environment variables (`FPM_*`)
6. CLI flags

### Environment variables

| Variable                   | Description                                      |
| -------------------------- | ------------------------------------------------ |
| `FPM_CACHE_DIR`            | Override cache directory                         |
| `FPM_DATA_DIR`             | Override data directory (Python installs, tools) |
| `FPM_CONFIG_DIR`           | Override config directory                        |
| `FPM_INDEX_URL`            | Override PyPI index URL                          |
| `FPM_CROSS_MANAGER_POLICY` | Set cross-manager policy                         |
| `FPM_PUBLISH_TOKEN`        | PyPI upload token                                |
| `FPM_ALLOW_INSECURE_HOST`  | Comma-separated hosts to skip TLS for            |
| `FPM_INSECURE`             | Disable ALL TLS verification (`1` to enable)     |
| `FPM_SYSTEM_CERTS`         | Force platform certificate verifier (`1`)        |
| `SSL_CERT_FILE`            | Custom CA bundle (overrides system certs)        |
| `SSL_CERT_DIR`             | Directory of PEM certs (overrides system certs)  |
| `SSL_CLIENT_CERT`          | mTLS client certificate file                     |
| `NO_COLOR`                 | Disable colored output                           |

## Network & TLS

fpm bundles Mozilla's root CA certificates for reliable HTTPS in Docker
containers and environments with missing system certificates.

```toml
# fpm.toml — network configuration
[network]
allow-insecure-host = ["internal-pypi.corp.example.com"]
# system-certs = true    # force platform certificate store
# client-cert = "/path/to/cert.pem"
# client-key = "/path/to/key.pem"
```

**Certificate resolution precedence:**

1. `SSL_CERT_FILE` / `SSL_CERT_DIR` — if set, these are the ONLY trusted roots
2. System certificate pool — platform's native store
3. Bundled Mozilla CAs — automatic fallback (Docker, air-gapped, etc.)

**For VPN/corporate proxy environments:**

```bash
# Per-host TLS bypass (recommended)
fpm install --allow-insecure-host pypi.org --allow-insecure-host files.pythonhosted.org requests

# Or set once via environment variable
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org

# Nuclear option: skip ALL TLS verification
FPM_INSECURE=1 fpm install requests
```

## Architecture

```
fpm/
├── cmd/fpm/          # CLI entry point
├── internal/
│   ├── cli/          # Command definitions and implementations
│   ├── config/       # Configuration parsing and hierarchy
│   ├── pep440/       # PEP 440 version parsing
│   ├── pep508/       # PEP 508 dependency specifiers
│   ├── platform/     # PEP 425 platform tags
│   ├── resolver/     # PubGrub dependency resolver
│   ├── client/       # PyPI HTTP client (PEP 691)
│   ├── tls/          # TLS certificates + per-host bypass
│   ├── cache/        # Content-addressable cache + GC
│   ├── installer/    # Package installation + linking
│   ├── env/          # Environment scanning + cross-manager
│   ├── venv/         # Virtual environment creation
│   ├── python/       # Python discovery + version management
│   ├── lock/         # Lockfile format
│   ├── wheel/        # Wheel parsing (PEP 427)
│   ├── snapshot/     # Environment snapshots
│   ├── workspace/    # Workspace/monorepo support
│   ├── build/        # PEP 517 build frontend
│   ├── publish/      # PyPI upload
│   ├── script/       # PEP 723 inline script support
│   ├── ui/           # Terminal output + progress
│   └── fs/           # Filesystem utilities + linking
├── pkg/
│   ├── types/        # Shared types
│   └── errors/       # Error types with hints
└── fpm.toml.example  # Reference configuration
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT — see [LICENSE](LICENSE)
