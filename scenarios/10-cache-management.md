# Scenario 10: Cache Management & Garbage Collection

## What This Proves

fpm's cache uses reference tracking to know exactly which environments use which
cached packages. This enables safe garbage collection — unlike pip/uv which
either cache everything forever or nuke the entire cache.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-cache && mkdir -p /tmp/fpm-demo-cache && cd /tmp/fpm-demo-cache
```

---

## Step 1: Install Packages to Populate Cache

```bash
mkdir project1 && cd project1
fpm venv && source .venv/bin/activate
fpm install requests flask numpy
deactivate && cd ..
```

---

## Step 2: Check Cache Size

```bash
fpm cache size
```

**Expected Output:**

```
Cache breakdown:
  Wheels:     45.2 MB  (15 files)
  CAS:       38.7 MB  (15 entries)
  HTTP:       0.3 MB  (metadata cache)
  Refs:       0.1 MB  (reference tracking)
  Total:     84.3 MB

Active environments: 1
Referenced entries:   15
Unreferenced entries: 0
```

---

## Step 3: Create Another Project (Shares Cache)

```bash
mkdir project2 && cd project2
fpm venv && source .venv/bin/activate
fpm install requests pandas  # requests already in cache — instant
deactivate && cd ..
```

```bash
fpm cache size
```

**Expected:** CAS size increases only for new packages (pandas + deps). requests
is shared, not duplicated.

---

## Step 4: Remove One Project

```bash
rm -rf project1
```

```bash
fpm cache list-unused
```

**Expected Output:**

```
Unreferenced CAS entries:
  flask 3.0.0           4.2 MB   (no active environment references this)
  werkzeug 3.0.0        1.8 MB
  jinja2 3.1.0          0.9 MB
  click 8.1.0           0.3 MB
  itsdangerous 2.1.0    0.1 MB
  markupsafe 2.1.0      0.1 MB
  numpy 1.24.0         28.4 MB

Total unreferenced: 35.8 MB
```

These packages are only referenced by the deleted project1. project2 still
references requests (so it stays referenced).

---

## Step 5: Safe Garbage Collection

```bash
fpm cache gc
```

**Expected Output:**

```
Removing 7 unreferenced CAS entries...
  - flask 3.0.0 ✓
  - werkzeug 3.0.0 ✓
  - numpy 1.24.0 ✓
  ...
Freed 35.8 MB

Kept 8 entries (still referenced by 1 environment)
```

Only truly unused packages are removed. project2 still works perfectly.

---

## Step 6: Verify project2 Still Works

```bash
cd project2
source .venv/bin/activate
python -c "import requests; print('OK:', requests.__version__)"
python -c "import pandas; print('OK:', pandas.__version__)"
deactivate && cd ..
```

**Expected:** Both imports work. GC didn't touch referenced packages.

---

## Step 7: Prune Corrupted Entries

```bash
# Simulate corruption (in real life: disk error, interrupted install)
# fpm cache prune detects and removes these
fpm cache prune
```

**Expected Output:**

```
Scanning CAS entries for integrity...
  Checked 8 entries
  0 corrupted entries found

Cache is healthy.
```

If corruption existed:

```
  Found 1 corrupted entry:
    sha256/ab/ab3f7c... (checksum mismatch)
  Removed 1 corrupted entry (0.5 MB freed)
```

---

## Step 8: Full Cache Clean (Nuclear Option)

```bash
fpm cache clean
```

**Expected Output:**

```
Removed all cache entries:
  Wheels:  45.2 MB
  CAS:    38.7 MB
  HTTP:    0.3 MB
  Total:  84.2 MB freed

Warning: Next install will re-download packages.
```

---

## Comparison: Cache Strategies

| Tool | Cache Strategy | GC Method                          | Safe?          |
| ---- | -------------- | ---------------------------------- | -------------- |
| pip  | Wheel cache    | `pip cache purge` (all or nothing) | No granularity |
| uv   | Wheel cache    | `uv cache clean` (all or nothing)  | No granularity |
| fpm  | CAS + refs     | `fpm cache gc` (only unreferenced) | Yes            |

---

## Reference Tracking Architecture

```
~/.cache/fpm/
├── refs/
│   ├── by-env/
│   │   ├── a1b2c3.json  → ["sha256/ab/ab3f7c...", "sha256/cd/cd5e8f..."]
│   │   └── d4e5f6.json  → ["sha256/ab/ab3f7c...", "sha256/gh/gh9i0j..."]
│   └── by-cas/
│       ├── sha256-ab-ab3f7c.json  → ["a1b2c3", "d4e5f6"]
│       └── sha256-cd-cd5e8f.json  → ["a1b2c3"]
```

- `by-env/`: "What CAS entries does this environment use?"
- `by-cas/`: "What environments reference this CAS entry?"
- GC removes entries where by-cas has zero references.

---

## Cleanup

```bash
rm -rf /tmp/fpm-demo-cache
```

---

## Key Takeaway

> fpm's reference-tracked garbage collection means you never have to choose
> between "waste disk space" and "break my environments." It knows exactly
> what's in use and only removes what's truly orphaned. This is the kind of
> precision that makes fpm safe to use on shared development machines.
