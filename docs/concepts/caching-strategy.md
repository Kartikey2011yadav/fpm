# Caching Strategy

## Cache Directory Structure

```
~/.cache/fpm/
├── cas/sha256/          # Content-addressable package storage
├── wheels/              # Raw downloaded .whl files
├── http/                # PyPI metadata responses (10-min TTL)
├── refs/                # Reference tracking (env ↔ package)
│   ├── by-env/          # Which packages each environment uses
│   └── by-cas/          # Which environments use each package
├── interpreters/        # Cached Python probe results
├── osv/                 # Vulnerability scan responses (1-hr TTL)
├── git/                 # Cloned git repositories
├── environments/        # Cached ephemeral tool environments
└── tmp/                 # Atomic staging area
```

## Cache Layers

### 1. HTTP Cache (PyPI Metadata)

When fpm queries `https://pypi.org/simple/requests/`, the JSON response
is cached to `http/requests.json`. On subsequent requests within 10 minutes,
the cached response is used without any network call.

**TTL**: 10 minutes (matches PyPI's Cache-Control header)
**Invalidation**: Automatic after TTL expires

### 2. Wheel Cache

Downloaded `.whl` files are kept in `wheels/`. If you install the same
version again (different project), the wheel is already there.

**Invalidation**: Never (wheels are immutable by hash). Clean with `fpm cache clean`.

### 3. CAS (Extracted Packages)

Extracted wheel contents live in `cas/sha256/{prefix}/{hash}/`. This is
what gets linked into your site-packages.

**Invalidation**: Reference-tracked. `fpm cache gc` removes unreferenced entries.

### 4. Interpreter Cache

Python interpreter metadata (version, paths, sysconfig) is cached by
`SHA256(binary_path + modification_time)`. If you upgrade Python, the
mtime changes and the cache entry is invalidated.

**TTL**: Until Python binary is modified

### 5. OSV Cache (Vulnerability Data)

Responses from `api.osv.dev` are cached per package+version.

**TTL**: 1 hour

### 6. Ephemeral Tool Environments

`fpm tool run <pkg>` caches the tool's venv. Second run reuses it.

**Invalidation**: `fpm cache clean` removes all.

## Reference Tracking

fpm tracks which environments use which CAS entries:

```json
// refs/by-env/{env_hash}.json
{
  "cas_keys": ["sha256:abc123", "sha256:def456"],
  "updated_at": "2026-06-07T..."
}

// refs/by-cas/sha256-abc123.json
{
  "environments": ["global:/usr/bin/python3", "/home/user/project/.venv"],
  "package_name": "requests",
  "version": "2.31.0"
}
```

This enables:
- `fpm cache gc` — safely remove unreferenced entries
- `fpm cache list-unused` — show what can be cleaned
- Instant restore from snapshots (find package by CAS key)

## Garbage Collection

```bash
fpm cache gc              # remove unreferenced packages
fpm cache gc --older-than 30d  # only GC entries unused for 30 days
fpm cache list-unused     # preview what would be removed
fpm cache size            # show breakdown by category
fpm cache clean           # nuclear: remove everything
```

`fpm cache gc` is safe — it only removes CAS entries that no environment
references. Installed packages are never affected.

## Cache Clearing Safety

| Action | Installed packages affected? | Need to re-download? |
|--------|------------------------------|---------------------|
| `fpm cache clean` | No (hardlinks survive) | Yes (on next install) |
| `fpm cache gc` | No | Only for unreferenced packages |
| Delete `~/.cache/fpm/` manually | No | Yes (on next install) |

Installed packages use hardlinks/reflinks from CAS. The link maintains
its own reference to the inode. Deleting the CAS path doesn't affect
the installed copy (on same filesystem).

## Developer Reference

Key code:
- `internal/cache/cache.go` — `Cache` struct, `Store()`, `Retrieve()`, `Init()`
- `internal/cache/reference.go` — `RefTracker`, bidirectional reference management
- `internal/client/client.go` — HTTP response caching (`readCache`, `writeCache`)
- `internal/python/discovery.go` — Interpreter metadata caching
- `internal/audit/audit.go` — OSV response caching
- `internal/tool/tool.go` — Ephemeral environment caching
