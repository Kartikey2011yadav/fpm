# snapshot

Environment versioning system — git-like snapshots for Python environments.

## Key Types

- `Snapshot` — captured environment state (all packages, all managers, Python
  version, paths)
- `Store` — per-environment snapshot storage and history
- `SnapshotDiff` — differences between two snapshots
- `DriftReport` — comparison of snapshot vs current live environment
- `RestoreResult` — outcome of a restore operation

## Features

- Captures packages from ALL managers (fpm, pip, uv, conda, poetry, pdm, system)
- fpm packages are fully restorable via CAS re-linking (instant, no network)
- If CAS entry is missing, auto-downloads from PyPI (never gets stuck)
- External packages produce drift warnings with detailed logs
- Scoped per-environment (each venv/project has independent history)

## Usage

```go
store := snapshot.NewStore(envPath)
snap, _ := store.Capture(scanResult, pythonVer, pythonPath, paths, "message")
snapshots, _ := store.List()
result, _ := snapshot.Restore(snap, currentScan, opts)
result.PrintLog()
```

## Files

- `snapshot.go` — capture, list, diff, drift detection
- `restore.go` — restore from CAS with auto-download fallback, drift logging
