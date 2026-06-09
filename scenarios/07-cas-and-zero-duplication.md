# Scenario 7: Content-Addressable Storage & Zero Duplication

## What This Proves

fpm stores each unique package version exactly once on disk using
content-addressable storage (CAS). Multiple projects sharing the same package
use reflinks or hardlinks — zero additional disk space. This is like Docker's
layer sharing but for Python packages.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-cas && mkdir -p /tmp/fpm-demo-cas && cd /tmp/fpm-demo-cas
```

---

## Step 1: Create Two Separate Projects

```bash
# Project A
mkdir project-a && cd project-a
fpm venv && source .venv/bin/activate
fpm install requests
deactivate
cd ..

# Project B
mkdir project-b && cd project-b
fpm venv && source .venv/bin/activate
fpm install requests flask
deactivate
cd ..
```

Both projects now have `requests` installed.

---

## Step 2: Verify Single Copy in CAS

```bash
# Check the CAS directory
ls ~/.cache/fpm/cas/sha256/ | head -20
```

**Expected:** Hash-prefixed directories. Each unique package version exists
exactly once here.

```bash
# Check cache size
fpm cache size
```

**Expected Output:**

```
Cache breakdown:
  Wheels:     12.3 MB  (8 files)
  CAS:        8.7 MB  (8 entries)
  HTTP:       0.2 MB  (metadata cache)
  Total:     21.2 MB

Referenced by: 2 environments
```

---

## Step 3: Verify Linking (Not Copying)

```bash
# On macOS (APFS) — reflinks have same size but different inodes
# On Linux (btrfs/xfs) — same

# Check if files are linked (same inode = hardlink)
stat project-a/.venv/lib/python3.*/site-packages/requests/__init__.py
stat project-b/.venv/lib/python3.*/site-packages/requests/__init__.py
```

**Expected:** Same inode number (hardlinks) or different inodes but no extra
disk usage (reflinks on APFS/btrfs).

```bash
# Compare disk usage — both projects together use barely more than one
du -sh project-a/.venv/lib/python3.*/site-packages/requests/
du -sh project-b/.venv/lib/python3.*/site-packages/requests/

# But actual disk usage is shared
du -sh ~/.cache/fpm/cas/
```

---

## Step 4: Create a Third Project (Instant Install)

```bash
mkdir project-c && cd project-c
fpm venv && source .venv/bin/activate

# This should be nearly instant — no download, just link
time fpm install requests
```

**Expected:** Installation completes in milliseconds (not seconds). The package
is already in CAS; fpm just creates links.

```bash
deactivate && cd ..
```

---

## Step 5: Reference Tracking

```bash
# See which environments reference which CAS entries
ls ~/.cache/fpm/refs/by-env/
ls ~/.cache/fpm/refs/by-cas/
```

**Expected:** JSON files tracking which projects use which cached packages. This
enables safe garbage collection.

---

## Step 6: Remove One Project (CAS Stays)

```bash
rm -rf project-c

# CAS still has requests (project-a and project-b still reference it)
fpm cache size
```

**Expected:** CAS size unchanged. Packages are only removed by explicit
`fpm cache gc`.

---

## Step 7: Garbage Collection (Safe Cleanup)

```bash
# List unreferenced packages
fpm cache list-unused

# Remove only truly unused packages
fpm cache gc
```

**Expected Output:**

```
Scanning references...
  Active environments: 2
  Referenced CAS entries: 8
  Unreferenced CAS entries: 0

Nothing to garbage collect.
```

Since project-a and project-b still exist, nothing is collected.

```bash
# Remove both projects
rm -rf project-a project-b

# Now GC will find unreferenced entries
fpm cache gc
```

**Expected:**

```
Removed 8 unreferenced CAS entries (8.7 MB freed)
```

---

## Disk Savings Example

| Scenario                  | pip/uv            | fpm             |
| ------------------------- | ----------------- | --------------- |
| 10 projects with numpy    | 10 × 30MB = 300MB | 1 × 30MB = 30MB |
| 5 projects with torch     | 5 × 2GB = 10GB    | 1 × 2GB = 2GB   |
| 20 projects with requests | 20 × 1MB = 20MB   | 1 × 1MB = 1MB   |

---

## Linking Strategy Selection

fpm automatically picks the best strategy:

| Filesystem      | Strategy                | Disk Cost | Speed   |
| --------------- | ----------------------- | --------- | ------- |
| macOS APFS      | Reflink (copy-on-write) | Zero      | Instant |
| Linux btrfs/xfs | Reflink (copy-on-write) | Zero      | Instant |
| Linux ext4      | Hardlink                | Zero      | Instant |
| Any (fallback)  | Copy                    | Full      | Slower  |

---

## Cleanup

```bash
rm -rf /tmp/fpm-demo-cas
fpm cache clean  # optional: clear all cache
```

---

## Key Takeaway

> fpm eliminates package duplication across projects. Whether you have 2 or 200
> projects using numpy, you store it exactly once. Combined with reflinks, new
> project installs are nearly instant. The reference-tracked GC ensures you
> never accidentally lose packages that are still in use.
