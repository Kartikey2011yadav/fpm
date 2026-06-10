# snapshot

Environment versioning system — git-like snapshots for Python environments.

## Key Types

- `Snapshot` — captured environment state (all packages, all managers, Python
  version, paths, fpm.toml config)
- `Store` — per-environment snapshot storage and history
- `SnapshotDiff` — differences between two snapshots
- `DriftReport` — comparison of snapshot vs current live environment
- `RestoreResult` — outcome of a restore operation

## Features

- Captures packages from ALL managers (fpm, pip, uv, conda, poetry, pdm, system)
- Captures `fpm.toml` config (immutable pins, project settings)
- Full restore for fpm packages via CAS re-linking (instant, no network)
- Full restore for external packages via pip install (re-downloads if needed)
- Removes packages that were added after the snapshot
- Restores fpm.toml to exact state at snapshot time (including immutable config)
- If CAS entry is missing, auto-downloads from PyPI (never gets stuck)
- Scoped per-environment (each venv/project has independent history)
- System-level snapshots supported via `--system` flag

## Restore Behavior

| What changed                          | What restore does                     |
| ------------------------------------- | ------------------------------------- |
| fpm package removed                   | Re-links from CAS (or re-downloads)   |
| pip/uv package removed                | Reinstalls via `pip install --target` |
| New package added after snapshot      | Removes it                            |
| Package version changed               | Reverts to snapshot version           |
| fpm.toml modified                     | Overwrites with snapshot version      |
| Immutable config added after snapshot | Removes it (restores original state)  |

## Usage

```go
store := snapshot.NewStore(envPath)
snap, _ := store.Capture(scanResult, pythonVer, pythonPath, paths, "msg", projectDir)
snapshots, _ := store.List()
result, _ := snapshot.Restore(snap, currentScan, snapshot.RestoreOptions{
    Cache:           cache,
    RefTracker:      tracker,
    PyPIClient:      client,
    SitePackages:    sitePackages,
    EnvPath:         envPath,
    AutoDownload:    true,
    RestoreExternal: true,
    ProjectDir:      projectDir,
})
result.PrintLog()
```

## Files

- `snapshot.go` — capture, list, diff, drift detection
- `restore.go` — full restore from CAS + external package restore + config
  restore
