<div align="center">

```
   ███████╗██████╗ ███╗   ███╗
   ██╔════╝██╔══██╗████╗ ████║
   █████╗  ██████╔╝██╔████╔██║
   ██╔══╝  ██╔═══╝ ██║╚██╔╝██║
   ██║     ██║     ██║ ╚═╝ ██║
   ╚═╝     ╚═╝     ╚═╝     ╚═╝
```

# Fast Package Manager for Python

**The package manager that sees everything, breaks nothing, and forgets nothing.**

[![CI](https://github.com/Kartikey2011yadav/fpm/actions/workflows/ci.yml/badge.svg)](https://github.com/Kartikey2011yadav/fpm/actions/workflows/ci.yml)
[![Go](https://img.shields.io/badge/Go-1.25-blue)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-macOS%20%7C%20Linux%20%7C%20Windows-lightgrey)]()

[Install](#install) &bull; [Quick Start](#quick-start) &bull; [Documentation](docs/concepts/README.md) &bull; [Why fpm?](#why-fpm)

</div>

---

A next-generation Python package manager that **coexists** with your existing tools.
Written in Go for speed. Designed for teams who need reproducibility, safety, and
visibility across their entire Python ecosystem.

```bash
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash
```

## Why fpm?

| Problem | pip/uv | fpm |
|---------|--------|-----|
| "What installed this package?" | No idea | Shows exact manager (pip/uv/conda/fpm) |
| "Roll back to yesterday's environment" | Impossible | `fpm snapshot restore <id>` |
| "Don't ever change numpy version" | Can't enforce | `[immutable]` pins in config |
| "10 projects use requests — 10 copies" | Yes | One copy, hardlinked everywhere |
| "pip broke my venv, conda disagrees" | Debug for hours | See all managers, detect conflicts |

## Key Features

### Environment Snapshots (git for your packages)

```bash
fpm snapshot create "before experiment"   # capture state
fpm install torch transformers            # experiment
fpm snapshot diff 20260607-143000         # see what changed
fpm snapshot restore 20260607-143000      # instantly roll back
```

### Cross-Manager Awareness

```bash
$ fpm list -a
Package    Version    Manager    Location
requests   2.31.0     fpm        .venv/lib/.../site-packages
numpy      1.24.0     pip        .venv/lib/.../site-packages
black      23.1.0     uv         .venv/lib/.../site-packages
scipy      1.10.0     conda      /opt/conda/lib/.../site-packages

$ fpm install numpy
  numpy 1.24.0 is already installed via pip — skipping download
```

### Immutable Package Pinning

```toml
# fpm.toml — numpy can NEVER be changed from this version
[immutable]
packages = [{ name = "numpy", version = "1.24.0" }]
```

```bash
$ fpm install numpy==2.0.0
  error: cannot install numpy ==2.0.0: pinned as immutable at 1.24.0 in fpm.toml
```

### Smart Dependency Management (like apt/pacman)

fpm tracks which packages you asked for vs which came as dependencies.
Only you decide what stays:

```bash
$ fpm install -s flask            # flask = REQUESTED (protected)
                                  # jinja2, click, etc. = DEPENDENCY (removable)

$ fpm remove -sp flask            # removes flask + only its unused deps
                                  # keeps urllib3 if requests still needs it

$ fpm autoremove --system         # clean ALL orphaned deps
$ fpm mark --requested click      # protect click from autoremove
$ fpm tree --system               # visualize the full dependency graph
```

No other Python package manager tracks this. pip/uv leave orphans forever.

### Zero-Duplication Storage

fpm stores every package once (content-addressable by SHA256) and links it into
projects via reflink/hardlink. 10 projects using `requests` = one copy on disk.

### Intelligent Error Messages

```bash
$ fpm install request
  error: Package "request" not found on PyPI

    hint: Did you mean "requests"?
          Run: fpm install requests
```

---

## Install

```bash
# Interactive installer (recommended)
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash

# Other methods
pip install fpm-pkg                                      # PyPI
brew install kartikeyyadav/tap/fpm                       # Homebrew
go install github.com/kartikeyyadav/fpm/cmd/fpm@latest   # Go
docker run ghcr.io/kartikey2011yadav/fpm --version       # Docker
```

## Quick Start

```bash
fpm init myproject && cd myproject   # create project + venv
fpm install requests pandas          # install packages
fpm run python main.py               # run in managed env
fpm snapshot create "v1"             # save environment state
fpm list                             # see fpm packages
fpm list -a                          # see ALL packages (pip, conda, etc.)
```

## CLI Reference

### Package Management

| Command | Description |
|---------|-------------|
| `fpm install <pkg>` | Install packages (alias: `add`) |
| `fpm remove <pkg>` | Remove (aliases: `uninstall`, `rm`, flags: `-f` `-p`) |
| `fpm autoremove` | Remove orphaned unused dependencies |
| `fpm mark --show <pkg>` | Show if package is requested/dependency |
| `fpm mark --requested <pkg>` | Protect package from autoremove |
| `fpm list` | List fpm packages (alias: `ls`) |
| `fpm list -a` | List ALL packages (all managers) |
| `fpm sync` | Sync environment from lockfile |
| `fpm lock` | Generate/update lockfile |
| `fpm tree` | Show dependency tree |
| `fpm audit` | Scan for vulnerabilities (OSV) |

### Project

| Command | Description |
|---------|-------------|
| `fpm init` | Create project (pyproject.toml + venv) |
| `fpm run <cmd>` | Run in managed environment |
| `fpm build` | Build wheel/sdist |
| `fpm publish` | Upload to PyPI |

### Environment

| Command | Description |
|---------|-------------|
| `fpm venv` | Create virtual environment |
| `fpm python list` | List Python versions |
| `fpm python install <ver>` | Install Python version |
| `fpm python use <ver>` | Switch Python (`--system` for global) |
| `fpm snapshot create [msg]` | Capture environment state |
| `fpm snapshot restore <id>` | Roll back to snapshot |
| `fpm snapshot diff <id>` | Compare snapshots |

### Tools & System

| Command | Description |
|---------|-------------|
| `fpm tool run <pkg>` | Run tool (cached ephemeral env) |
| `fpm tool install <pkg>` | Install CLI tool permanently |
| `fpm cache gc` | Remove unused packages |
| `fpm repair` | Diagnose + fix issues |
| `fpm config show` | Display all settings |

### Flags

| Flag | Description |
|------|-------------|
| `-v` | Print version |
| `-s`, `--system` | Install to system Python (required without venv) |
| `-q`, `--quiet` | Suppress output |
| `--json` | JSON output |
| `--allow-insecure-host <host>` | Skip TLS for specific hosts |
| `--log-level <level>` | debug/info/warn/error/off |

## Configuration

```toml
# fpm.toml
[project]
name = "my-project"
requires-python = ">=3.10"
dependencies = ["requests>=2.28", "numpy"]

[tool]
cross-manager-policy = "ask"    # ask | install | skip
concurrency = 50

[immutable]
packages = [{ name = "numpy", version = "1.24.0" }]

[network]
allow-insecure-host = ["internal-pypi.corp.example.com"]

[log]
level = "off"                   # debug | info | warn | error | off
```

**Priority** (highest wins): CLI flags > env vars > project fpm.toml > user config > defaults

## How It's Different

| Feature | pip | uv | fpm |
|---------|-----|----|----|
| Content-addressable storage | No | No | Yes (zero duplication) |
| Environment snapshots | No | No | Yes (instant restore) |
| Cross-manager detection | No | No | Yes (pip, uv, conda, poetry, pdm) |
| Immutable version pins | No | No | Yes (enforced by resolver) |
| Dependency graph tracking | No | No | Yes (requested vs transitive) |
| Autoremove orphans | No | No | Yes (like apt/pacman) |
| Reflink support | No | No | Yes (CoW on APFS/btrfs) |
| Reference-tracked GC | No | No | Yes (only removes orphans) |
| Bundled CA certificates | certifi | webpki | Mozilla (built-in) |
| Per-host TLS bypass | `--trusted-host` | `--allow-insecure-host` | `--allow-insecure-host` |
| Requires venv for install | No | Yes (`--system`) | Yes (`--system`) |

## Platforms

| Platform | Status |
|----------|--------|
| macOS (Apple Silicon + Intel) | Supported |
| Linux (x86_64 + ARM64) | Supported |
| Windows (x86_64) | Supported |
| Docker (multi-arch) | Supported |
| JupyterHub / multi-user | Supported |

## Documentation

- [Installation Guide](docs/guides/installation.md)
- [Quick Start](docs/guides/quickstart.md)
- [CLI Reference](docs/reference/cli.md)
- [Configuration](docs/reference/configuration.md)
- [Concepts](docs/concepts/README.md) (CAS, resolution, snapshots, TLS)
- [Architecture](docs/architecture.md)
- [Testing](docs/guides/testing.md)
- [Comparison: fpm vs uv vs pip](docs/comparison-uv-pip-fpm.md)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

## License

MIT
