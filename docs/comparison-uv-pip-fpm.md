# Comparison: fpm vs uv vs pip

## Installation Model

| Aspect        | pip                                       | uv                                  | fpm                                      |
| ------------- | ----------------------------------------- | ----------------------------------- | ---------------------------------------- |
| Storage       | Direct copy to site-packages              | Direct copy (with cache)            | Content-addressable cache + link         |
| Deduplication | None — each venv gets its own copy        | Hardlinks from cache                | Reflinks/hardlinks from CAS              |
| Cache         | Wheel files only (`~/.cache/pip/wheels/`) | Wheels + extracted (`~/.cache/uv/`) | Wheels + CAS extracted (`~/.cache/fpm/`) |
| Install speed | Sequential download + extract + copy      | Parallel download + extract + copy  | Parallel download + extract once + link  |
| Re-install    | Re-downloads or uses wheel cache          | Uses cache, re-extracts             | Instant link from CAS (no extraction)    |

## Directory Structure Comparison

### pip

```
~/.cache/pip/
├── wheels/          # Cached built wheels
├── http/            # HTTP response cache
└── selfcheck/       # pip self-update check

site-packages/
├── requests/              # Full copy of package
└── requests-2.31.0.dist-info/
    ├── METADATA
    ├── RECORD
    ├── INSTALLER          # "pip\n"
    └── top_level.txt
```

### uv

```
~/.cache/uv/
├── wheels-v3/       # Cached wheel files
├── archive-v0/      # Extracted packages (by hash)
├── simple-v14/      # PyPI metadata cache
├── interpreter-v4/  # Python interpreter info cache
└── builds-v0/       # Source distribution build cache

site-packages/
├── requests/              # Hardlinked from archive
└── requests-2.31.0.dist-info/
    ├── METADATA
    ├── RECORD
    ├── INSTALLER          # "uv\n"
    └── direct_url.json
```

### fpm

```
~/.cache/fpm/
├── wheels/          # Raw .whl downloads
├── cas/sha256/      # Content-addressable storage (extracted)
│   └── <prefix>/<full_hash>/
│       ├── requests/
│       └── requests-2.31.0.dist-info/
├── http/            # PyPI metadata cache (10-min TTL)
├── refs/            # Bidirectional reference tracking
│   ├── by-env/     # Which CAS entries each env uses
│   └── by-cas/     # Which envs reference each CAS entry
└── tmp/             # Atomic staging area

site-packages/
├── requests/              # Reflinked/hardlinked from CAS
└── requests-2.31.0.dist-info/
    ├── METADATA
    ├── RECORD
    └── INSTALLER          # "fpm\n"
```

## Dependency Resolution

| Aspect              | pip                            | uv              | fpm                       |
| ------------------- | ------------------------------ | --------------- | ------------------------- |
| Algorithm           | Backtracking resolver          | PubGrub (Rust)  | PubGrub (Go)              |
| Lockfile            | None (requirements.txt manual) | `uv.lock`       | `fpm.lock`                |
| Immutable pins      | Not supported                  | Not supported   | `[immutable]` in fpm.toml |
| Cross-platform lock | No                             | Yes (universal) | Yes (universal)           |

## TLS/Certificate Handling

| Aspect          | pip                                        | uv                                       | fpm                                     |
| --------------- | ------------------------------------------ | ---------------------------------------- | --------------------------------------- |
| Default certs   | Bundled `certifi` (Python package)         | Bundled `webpki-root-certs` (Rust crate) | Bundled Mozilla CAs (`breml/rootcerts`) |
| Custom CA       | `--cert`, `PIP_CERT`, `REQUESTS_CA_BUNDLE` | `SSL_CERT_FILE`, `SSL_CERT_DIR`          | `SSL_CERT_FILE`, `SSL_CERT_DIR`         |
| Per-host bypass | `--trusted-host`                           | `--allow-insecure-host`                  | `--allow-insecure-host`                 |
| System certs    | Via certifi (not system)                   | `--system-certs` / `UV_SYSTEM_CERTS`     | System pool is default fallback         |
| Global bypass   | N/A                                        | N/A                                      | `FPM_INSECURE=1`                        |

## Cross-Manager Awareness

| Aspect                 | pip                   | uv                      | fpm                                 |
| ---------------------- | --------------------- | ----------------------- | ----------------------------------- |
| Detects other managers | No                    | No                      | Yes (pip, uv, conda, poetry, pdm)   |
| Conflict detection     | No                    | No                      | Yes (warns before overwriting)      |
| Environment snapshots  | No                    | No                      | Yes (`fpm snapshot create/restore`) |
| Shows all managers     | `pip list` (pip only) | `uv pip list` (uv only) | `fpm list -a` (all managers)        |

## Linking Strategy

| Strategy      | pip    | uv                     | fpm                               |
| ------------- | ------ | ---------------------- | --------------------------------- |
| Reflink (CoW) | No     | No                     | Yes (macOS APFS, Linux btrfs/xfs) |
| Hardlink      | No     | Yes (from cache)       | Yes (fallback)                    |
| Copy          | Always | Fallback               | Last fallback                     |
| Space savings | 0%     | Good (hardlinks share) | Best (reflinks are free)          |

## Environment Isolation

| Scenario                   | pip                                    | uv                           | fpm                                     |
| -------------------------- | -------------------------------------- | ---------------------------- | --------------------------------------- |
| Default install target     | System or venv (depends on activation) | Requires explicit target     | Auto-detects venv, falls back to system |
| `pip install` outside venv | System site-packages (may need sudo)   | Error (requires `--system`)  | System site-packages (with warning)     |
| Project isolation          | Manual `python -m venv`                | `uv venv` + `uv pip install` | `fpm init` (creates venv automatically) |

## Garbage Collection

| Aspect            | pip                                | uv                      | fpm                                 |
| ----------------- | ---------------------------------- | ----------------------- | ----------------------------------- |
| Cache cleanup     | `pip cache purge` (all or nothing) | `uv cache clean`        | `fpm cache gc` (only unreferenced)  |
| Tracks references | No                                 | No                      | Yes (by-env and by-cas refs)        |
| Safe GC           | No (removes everything)            | No (removes everything) | Yes (only removes orphaned entries) |

## Key Architectural Differences

### pip

- Written in Python (bootstrapping issue — needs Python to install Python
  packages)
- Single-threaded downloads
- No lockfile (requirements.txt is manual, not resolved)
- No content-addressable storage
- Full copies to every venv

### uv

- Written in Rust (standalone binary)
- Massively parallel downloads
- Universal lockfile (`uv.lock`)
- Cache with hardlinks (deduplication)
- Focus on speed (10-100x faster than pip)
- Does NOT detect packages from other managers

### fpm

- Written in Go (standalone binary)
- Parallel downloads with configurable concurrency
- Content-addressable storage (unique to fpm)
- Reflink support (zero-copy on modern filesystems)
- Reference tracking enables safe garbage collection
- Cross-manager awareness (detects pip, uv, conda, etc.)
- Environment snapshots (unique to fpm)
- Immutable package pinning (unique to fpm)

## Install Flow Comparison

### `pip install requests`

```
1. Resolve deps (backtracking)
2. Download wheels → ~/.cache/pip/wheels/
3. Extract + copy files → site-packages/
4. Write RECORD, INSTALLER
```

### `uv pip install requests`

```
1. Resolve deps (PubGrub, parallel metadata fetch)
2. Download wheels → ~/.cache/uv/wheels-v3/
3. Extract to archive → ~/.cache/uv/archive-v0/<hash>/
4. Hardlink files → site-packages/
5. Write RECORD, INSTALLER
```

### `fpm install requests`

```
1. Resolve deps (PubGrub, parallel metadata fetch)
2. Download wheels → ~/.cache/fpm/wheels/
3. Hash wheel → CAS key
4. Extract to CAS → ~/.cache/fpm/cas/sha256/<prefix>/<hash>/
5. Reflink/hardlink → site-packages/
6. Write INSTALLER marker
7. Track reference (env → CAS key mapping)
8. Update pyproject.toml + fpm.lock
```

## What fpm Does That Others Don't

1. **Reference-tracked cache** — knows exactly which envs use which packages;
   enables safe GC
2. **Environment snapshots** — capture + restore full env state (like git for
   packages)
3. **Cross-manager detection** — sees packages from pip, uv, conda, etc. in one
   view
4. **Immutable pins** — packages that can never be changed (safety for
   production)
5. **Reflink-first linking** — zero-space installs on modern filesystems
6. **Coexistence by design** — doesn't fight other managers, detects and
   respects them
