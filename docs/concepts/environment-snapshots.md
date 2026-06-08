# Environment Snapshots

## What They Are

Snapshots capture the complete state of your Python environment at a point in
time — every package, from every manager, with exact versions. Think of it as
git commits for your environment.

## Why They Exist

- **Experiment safely**: install experimental packages, restore if things break
- **Reproduce**: share exact environment state with teammates
- **Audit**: see what changed between two points in time
- **Recover**: roll back after a bad `pip install` or accidental upgrade

## How They Work

### Capture

```bash
fpm snapshot create "before ML experiment"
```

This scans all site-packages directories and records:
- Package name and version
- Which manager installed it (fpm, pip, uv, conda, etc.)
- CAS key (for fpm packages — enables instant restore)
- Python version and path
- All site-packages paths in order

Stored as JSON in `~/.cache/fpm/snapshots/<env-hash>/snap-<id>.json`.

### Scope

Snapshots are **scoped per environment**. Each venv has its own independent
snapshot history. The scope is determined by hashing the venv path:

```
project-a/.venv → hash: a1b2c3... → snapshots stored separately
project-b/.venv → hash: d4e5f6... → snapshots stored separately
```

This means project A's snapshots never interfere with project B's.

### Restore

```bash
fpm snapshot restore 20260607-143000
```

Restoration strategy:
1. **fpm packages**: restored from CAS (hardlink/reflink — instant, no network)
2. **Missing from CAS**: re-downloaded from PyPI
3. **Other managers' packages**: drift reported but NOT modified

```
  ✓ Restored 12 fpm packages from cache
  ↓ Re-downloaded 1 package (cache was cleaned)
  ⚠ pip's scipy: expected 1.10.0, found 1.11.0
```

fpm respects other managers' ownership — it will tell you about drift
but won't uninstall or modify packages it didn't install.

### Diff

```bash
fpm snapshot diff 20260607-100000 20260607-143000
```

Shows:
```
  + pandas 2.0.0 (fpm)         ← added
  + numpy 1.24.0 (fpm)         ← added
  ~ requests 2.30.0 → 2.31.0 (fpm)  ← version changed
  - flask 2.3.0 (fpm)          ← removed
```

Compare a snapshot against current state:
```bash
fpm snapshot diff 20260607-100000
# Compares snapshot vs what's installed right now
```

## Snapshot Lifecycle

```
create → list → diff → restore → delete
                  ↑
                  └── compare against current
```

- **create**: capture state (with optional message)
- **list**: show all snapshots (newest first, `*` marks current)
- **diff**: compare two snapshots or snapshot vs now
- **restore**: roll back to a previous state
- **delete**: remove a snapshot (CAS data stays until `fpm cache gc`)

## Storage Format

```json
{
  "id": "20260607-143000-001",
  "created_at": "2026-06-07T14:30:00Z",
  "message": "before ML experiment",
  "python_version": "3.12.1",
  "python_path": "/path/to/.venv/bin/python3",
  "packages": [
    {
      "name": "requests",
      "version": "2.31.0",
      "manager": "fpm",
      "cas_key": "sha256:ab3f7c8d...",
      "location": "/path/to/.venv/lib/python3.12/site-packages"
    }
  ]
}
```

The `cas_key` field enables instant restore — fpm can re-link from CAS
without any network access.

## Developer Reference

Key code:
- `internal/snapshot/snapshot.go` — `Store`, `Capture()`, `List()`, `Get()`, `Diff()`
- `internal/snapshot/restore.go` — `Restore()`, CAS re-linking, drift detection
- `internal/cli/snapshot.go` — All CLI commands (create, list, restore, diff, delete)
- `internal/env/scanner.go` — `Scan()` captures the environment state
