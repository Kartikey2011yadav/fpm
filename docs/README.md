# fpm Documentation

## Getting Started

- [Installation](guides/installation.md) — Install fpm on any platform
- [Quick Start](guides/quickstart.md) — Create your first project in 2 minutes
- [Migration from pip](guides/migration.md) — Move existing projects to fpm

## Guides

- [Managing Dependencies](guides/dependencies.md) — Install, update, remove
  packages
- [Python Versions](guides/python-versions.md) — Install and switch Python
  versions
- [Environment Snapshots](guides/snapshots.md) — Version your environment like
  git
- [Working with Other Managers](guides/coexistence.md) — pip, uv, conda
  coexistence
- [Tools](guides/tools.md) — Install and run CLI tools (pipx-like)
- [Building & Publishing](guides/building.md) — Build wheels and publish to PyPI
- [Workspaces](guides/workspaces.md) — Monorepo and multi-package projects
- [Docker & CI](guides/docker.md) — Using fpm in containers and CI/CD

## Reference

- [CLI Commands](reference/cli.md) — Complete command reference
- [Configuration](reference/configuration.md) — fpm.toml and environment
  variables
- [Lockfile Format](reference/lockfile.md) — fpm.lock specification
- [Cache & Storage](reference/cache.md) — CAS architecture and GC

## Concepts

- [How Resolution Works](concepts/resolution.md) — PubGrub algorithm and
  constraints
- [Path Precedence](concepts/path-precedence.md) — How fpm manages sys.path
- [Content-Addressable Cache](concepts/cas.md) — Zero-duplication storage
- [Cross-Manager Awareness](concepts/cross-manager.md) — Detecting and
  coexisting
