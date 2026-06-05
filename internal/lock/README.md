# lock

TOML-based universal lockfile for reproducible installations.

## Key Types

- `Lockfile` — lockfile structure (version, packages, metadata)
- `LockedPackage` — package entry (name, version, source, hashes, dependencies,
  markers)
- `LockfileDiff` — differences between two lockfiles

## Features

- TOML format (human-readable, git-friendly diffs)
- Universal: includes platform markers for cross-platform support
- Content hashes for integrity verification
- Diff computation for efficient syncing
- Export to requirements.txt format

## Files

- `lockfile.go` — generation, read/write, diff, export
