# Environment Snapshots

Snapshots let you version your Python environment like git versions code.

## Creating Snapshots

```bash
# Capture current state with a message
fpm snapshot create "before ML experiment"

# Quick snapshot (no message)
fpm snapshot create
```

A snapshot captures:

- All packages from ALL managers (fpm, pip, uv, conda, poetry, pdm, system)
- Package versions and locations
- Python version and path
- sys.path ordering

## Viewing History

```bash
fpm snapshot list
```

Output:

```
Environment snapshots:

* 20260605-143200  2026-06-05 14:32  [45 packages]  added ML stack
  20260605-140500  2026-06-05 14:05  [12 packages]  before ML experiment
  20260605-093000  2026-06-05 09:30  [12 packages]  initial setup
```

The `*` marks the current snapshot.

## Comparing Snapshots

```bash
# Compare two snapshots
fpm snapshot diff 20260605-093000 20260605-143200

# Compare snapshot against current state
fpm snapshot diff 20260605-093000
```

Output:

```
Diff: 20260605-093000 → 20260605-143200

  + torch 2.1.0 (fpm)
  + numpy 1.24.0 (fpm)
  + transformers 4.35.0 (fpm)
  ~ scipy 1.10.0 → 1.11.0 (pip)
  - old-package 0.1.0 (fpm)
```

## Restoring Snapshots

```bash
fpm snapshot restore 20260605-093000
```

Output:

```
Restoring snapshot 20260605-093000...
  ✓ Restored 10 packages from cache
  ↓ Re-downloaded 2 packages (missing from cache)

  External package status:
    ~ scipy (pip): expected 1.10.0, found 1.11.0
    ✗ old-lib (conda): was 0.5.0, now missing
    + new-thing (uv): 1.0.0 (not in snapshot)
```

Key behaviors:

- **fpm packages**: fully restored from CAS (instant, no network)
- **Missing from CAS**: automatically re-downloaded from PyPI
- **External packages (pip/uv/conda)**: reinstalled to exact snapshot version
- **New packages (added after snapshot)**: removed from environment
- **fpm.toml config**: reverted to snapshot state (including immutable pins)

### System Conflict Resolution

If a project restore conflicts with system packages, fpm prompts:

```
  ⚠ System package conflicts detected:
    numpy: snapshot needs 1.24.0, system has 2.0.0

  [1] Roll back system packages too
  [2] Install at project level (overrides system)
  [3] Abort
```

## Scoping

Snapshots are per-environment:

- Each project's `.venv` has its own snapshot history
- Project A's snapshots are independent of Project B's
- System-level snapshots use `--system`:

```bash
fpm snapshot create --system "before system upgrade"
fpm snapshot list --system
fpm snapshot restore --system <id>
```

## Deleting Snapshots

```bash
fpm snapshot delete 20260605-093000
```

Deleting a snapshot may allow GC to clean packages only referenced by that
snapshot.

## Best Practices

1. **Snapshot before experiments**: `fpm snapshot create "before trying X"`
2. **Snapshot working states**: when everything works, save it
3. **Use messages**: they're your commit messages for the environment
4. **Snapshot before upgrades**: easy rollback if something breaks
