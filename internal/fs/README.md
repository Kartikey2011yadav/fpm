# fs

Cross-platform filesystem utilities with intelligent linking.

## Key Types

- `LinkMode` — linking strategy (auto, reflink, hardlink, copy, symlink)

## Link Strategy

fpm uses the fastest available method:

1. **Reflink/CoW** (APFS on macOS, btrfs/xfs on Linux) — instant, zero space
2. **Hardlink** — instant, shared inode
3. **Copy** — fallback for cross-device or unsupported filesystems

## Files

- `link.go` — `LinkFile`, `LinkDir`, `BestLinkMode`, `AtomicRename`, `DirSize`
- `reflink_darwin.go` — macOS clonefile(2) syscall
- `reflink_linux.go` — Linux FICLONE ioctl
- `reflink_other.go` — fallback for unsupported platforms
