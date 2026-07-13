# Git-Inspired Feature Roadmap

fpm treats Python environments like git treats source code — versioned, diffable,
and reversible. This document outlines features inspired by git that would make
fpm the most powerful environment management tool available.

---

## Feature Priority Matrix

| Feature | Git Equivalent | User Impact | Complexity | Priority |
|---------|---------------|-------------|------------|----------|
| `fpm status` | `git status` | High | Low | **P0** |
| `fpm log` | `git log` | High | Medium | **P0** |
| `fpm tag <name>` | `git tag` | Medium | Low | **P0** |
| `fpm stash` / `pop` | `git stash` | High | Medium | **P1** |
| `fpm revert <id>` | `git revert` | High | Medium | **P1** |
| `fpm blame <pkg>` | `git blame` | Medium | Low | **P1** |
| `fpm cherry-pick` | `git cherry-pick` | High | Medium | **P1** |
| `fpm diff <env1> <env2>` | `git diff` | Medium | Medium | **P1** |
| `fpm reset --hard` | `git reset --hard` | Low | Low | **P1** |
| `fpm bisect` | `git bisect` | High | High | **P2** |
| `fpm branch` | `git branch` | High | High | **P2** |
| `fpm reflog` | `git reflog` | Medium | Medium | **P2** |
| `fpm hooks` | `.git/hooks` | Medium | Medium | **P2** |
| `fpm merge <env>` | `git merge` | Medium | High | **P3** |

---

## P0 Features (Ship First)

### `fpm status`

Show environment drift from lockfile — the most requested "what's going on?" command.

```bash
$ fpm status
On environment: .venv (Python 3.12.1)
Lockfile: fpm.lock (last updated 2 days ago)

Changes not in lockfile:
  (use "fpm lock" to update lockfile)

    added:   httpx 0.27.0
    added:   httpcore 1.0.2
    upgraded: requests 2.28.0 → 2.31.0

Lockfile packages not installed:
  (use "fpm sync" to install)

    missing: flask 3.0.0
```

**Implementation:**
- Compare `fpm.lock` packages vs `env.Scan()` results
- Show added/removed/changed/missing packages
- Colored output (green=added, red=removed, yellow=changed)
- Exit code 0 if clean, 1 if dirty (for CI scripting)

---

### `fpm log`

Operation history journal — every install, remove, upgrade recorded.

```bash
$ fpm log
2024-03-15 14:32  install   requests==2.31.0, urllib3==2.1.0, certifi==2024.2.2
2024-03-15 14:30  install   flask==3.0.0, werkzeug==3.0.0, jinja2==3.1.0
2024-03-15 14:28  init      Created project (Python 3.12.1)

$ fpm log --oneline
abc123  install requests==2.31.0 (+2 deps)
def456  install flask==3.0.0 (+5 deps)
789abc  init    Created project
```

**Implementation:**
- Append-only JSONL file: `{envPath}/.fpm-journal.jsonl`
- Each entry: `{id, timestamp, operation, packages[], user, message}`
- Enables `fpm revert`, `fpm blame`
- `--since`, `--until`, `--grep` filters

---

### `fpm tag <name> [snapshot-id]`

Named aliases for snapshots (human-friendly references).

```bash
$ fpm snapshot create "deploy candidate"
Snapshot created: 20240315-143200-001

$ fpm tag production 20240315-143200-001
Tagged "production" → 20240315-143200-001

$ fpm snapshot restore production
Restoring tagged snapshot "production"...
```

**Implementation:**
- Tags stored in `{snapshot-dir}/tags.json` (name → snapshot-id map)
- All snapshot commands accept tag names as aliases
- `fpm tag list`, `fpm tag delete <name>`

---

## P1 Features (High Value)

### `fpm stash` / `fpm stash pop`

Temporarily save environment changes, work in clean state, restore later.

```bash
$ fpm install httpx pytest  # experimenting
$ fpm stash                 # save current delta
Stashed 2 packages (httpx, pytest) — environment matches lockfile

$ fpm stash list
stash@{0}: 2 packages (httpx==0.27.0, pytest==8.0.0)

$ fpm stash pop             # restore experiment
Restored: httpx==0.27.0, pytest==8.0.0
```

**Use cases:**
- "Let me try something but keep my current state safe"
- "I need a clean env to test a bug, then get back to development"
- Quick context switching between feature experiments

**Implementation:**
- Stash = packages installed but not in lockfile (diff vs lockfile)
- Store: `{envPath}/.fpm-stash/` (stack of diffs)
- `stash`: remove non-lockfile packages, save list
- `pop`: re-install saved packages

---

### `fpm revert <operation-id>`

Undo a specific past operation by its journal ID.

```bash
$ fpm log --oneline
abc123  install numpy==2.0.0
def456  install requests==2.31.0

$ fpm revert abc123
Reverting: install numpy==2.0.0
  Removed numpy 2.0.0
  Removed 3 transitive dependencies
```

**Implementation:**
- Read journal entry by ID
- Generate inverse operation (install→remove, remove→install, upgrade→downgrade)
- Execute inverse
- Record revert in journal

---

### `fpm blame <package>`

Show who/when/why a package was installed.

```bash
$ fpm blame certifi
certifi 2024.2.2
  Installed: 2024-03-15 14:32
  Reason: transitive dependency of requests==2.31.0
  Required by: requests, httpx
  Journal: abc123 (install requests==2.31.0)
```

**Implementation:**
- Query journal for first appearance of package
- Query depgraph for `DependentsOf()`
- Show full provenance chain

---

### `fpm cherry-pick <snapshot> <package>`

Restore a single package from a snapshot without full restore.

```bash
$ fpm cherry-pick 20240315-143200-001 numpy
Cherry-picked numpy==1.24.0 from snapshot 20240315-143200-001
  (was: numpy==2.0.0 → now: numpy==1.24.0)
```

**Implementation:**
- Read snapshot, find package entry
- If CAS key available: link from CAS (instant)
- If not: download specific version from PyPI
- Update graph (version change)

---

### `fpm diff <env1> <env2>`

Compare two environments (snapshots, tags, or live envs).

```bash
$ fpm diff production staging
- requests==2.28.0       (production only)
+ requests==2.31.0       (staging only)
+ httpx==0.27.0          (staging only)
  flask==3.0.0           (both, same version)
~ numpy 1.24.0 → 2.0.0  (version differs)
```

---

### `fpm reset --hard <snapshot>`

Force restore, discarding ALL current state without safety snapshot.

```bash
$ fpm reset --hard production
WARNING: This will discard all current packages without backup.
Proceed? [y/N]: y
Reset to "production" (20240315-143200-001)
```

---

## P2 Features (Strategic)

### `fpm bisect`

Binary search through snapshots to find which change introduced a bug.

```bash
$ fpm bisect start
$ fpm bisect good 20240301-100000-001   # this snapshot works
$ fpm bisect bad  20240315-143200-001   # this one is broken

Bisecting: 5 snapshots left to test
[20240308-120000-001] Restored. Run your test and mark good/bad:
  fpm bisect good   OR   fpm bisect bad

$ fpm bisect bad
Bisecting: 2 snapshots left...
[20240305-090000-001] Restored.

$ fpm bisect good
Found! First bad snapshot: 20240306-150000-001
  Changes in that snapshot:
    + numpy 2.0.0 (was 1.24.0)
    + scipy 1.12.0 (new)
```

---

### `fpm branch`

Named environment branches for parallel experimentation.

```bash
$ fpm branch create feature-ml
Created branch "feature-ml" from current state

$ fpm branch switch feature-ml
Switched to environment branch "feature-ml"

$ fpm install torch  # only in this branch
$ fpm branch switch main
Switched to "main" (torch not present here)

$ fpm branch list
* main
  feature-ml (3 extra packages)
```

---

### `fpm reflog`

Show ALL state changes including ones that were undone or lost.

```bash
$ fpm reflog
HEAD@{0}: snapshot restore 20240315-143200-001
HEAD@{1}: install httpx==0.27.0
HEAD@{2}: remove numpy (purge: -3 deps)
HEAD@{3}: snapshot create "before experiment"
```

---

### `fpm hooks`

Pre/post operation hooks (like `.git/hooks`).

```toml
# fpm.toml
[hooks]
post-install = "pytest tests/ -x"
pre-remove = "echo 'Removing: {packages}'"
post-sync = "./scripts/validate-env.sh"
```

---

## Implementation Roadmap

### Phase 1 (v0.3): Foundation
- [ ] Operation journal (`fpm log`)
- [ ] `fpm status`
- [ ] `fpm tag`

### Phase 2 (v0.4): Power Features
- [ ] `fpm stash` / `fpm stash pop`
- [ ] `fpm revert`
- [ ] `fpm blame`
- [ ] `fpm cherry-pick`
- [ ] `fpm diff`

### Phase 3 (v0.5): Advanced Git Parity
- [ ] `fpm bisect`
- [ ] `fpm branch`
- [ ] `fpm reflog`
- [ ] `fpm hooks`

### Phase 4 (v1.0): Production
- [ ] `fpm merge`
- [ ] Remote environment sync (team environments)
- [ ] Signed snapshots (tamper-proof)

---

## Design Principles

1. **Non-destructive by default** — every operation creates a recovery point
2. **Fast paths for common cases** — CAS makes restore instant
3. **Progressive disclosure** — simple commands for beginners, power features for experts
4. **Composable** — commands work together (`log` enables `revert`, `blame`, `bisect`)
5. **Offline-first** — all local operations work without network
