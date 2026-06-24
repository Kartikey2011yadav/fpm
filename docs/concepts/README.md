# Concepts

Core concepts that explain how fpm works under the hood. Read these to
understand the architecture and design decisions.

## For Users

| Document                                                  | What you'll learn                                                  |
| --------------------------------------------------------- | ------------------------------------------------------------------ |
| [Virtual Environments](virtual-environments.md)           | How fpm manages project isolation, venv detection, `--system` flag |
| [Dependency Resolution](dependency-resolution.md)         | How packages are resolved, lockfiles, immutable pins               |
| [Cross-Manager Coexistence](cross-manager-coexistence.md) | How fpm works alongside pip/uv/conda, conflict handling            |
| [Environment Snapshots](environment-snapshots.md)         | Git-like versioning for your Python environment                    |
| [TLS & Certificates](tls-and-certificates.md)             | Certificate handling, VPN/proxy support, insecure hosts            |
| [Multi-User Systems](multi-user-systems.md)               | Single-user vs multi-user mode, shared CAS, file locking, scaling  |

## For Developers

| Document                                                      | What you'll learn                               |
| ------------------------------------------------------------- | ----------------------------------------------- |
| [Content-Addressable Storage](content-addressable-storage.md) | CAS model, linking strategies, zero-duplication |
| [Caching Strategy](caching-strategy.md)                       | All cache layers, TTLs, reference tracking, GC  |

## Quick Architecture Overview

```
User runs: fpm install requests
                    │
                    ▼
        ┌───────────────────┐
        │  CLI (cobra)       │  Parse args, detect venv
        └─────────┬─────────┘
                  │
                  ▼
        ┌───────────────────┐
        │  Resolver (PubGrub)│  Fetch metadata, resolve deps
        └─────────┬─────────┘
                  │
                  ▼
        ┌───────────────────┐
        │  Cross-Manager     │  Check for existing installs
        └─────────┬─────────┘
                  │
                  ▼
        ┌───────────────────┐
        │  Download + CAS    │  Fetch wheels, store in CAS
        └─────────┬─────────┘
                  │
                  ▼
        ┌───────────────────┐
        │  Link + Track      │  Hardlink to site-packages, update refs
        └─────────┬─────────┘
                  │
                  ▼
        ┌───────────────────┐
        │  Lock + Config     │  Update fpm.lock, pyproject.toml
        └───────────────────┘
```

## Design Principles

1. **Never break what's working** — fpm respects other managers' packages
2. **Zero duplication** — CAS ensures each package version stored once
3. **Instant operations** — linking from CAS is O(1), no extraction
4. **Explicit is better** — `--system` required for global installs
5. **Helpful errors** — every error has an actionable hint
6. **Works offline** — once cached, no network needed for installs
