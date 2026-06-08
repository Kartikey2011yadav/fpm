# Dependency Graph Tracking

## The Problem

When you `pip install flask`, pip installs flask + 8 dependencies.
When you `pip uninstall flask`, pip removes flask but leaves all 8 deps
orphaned forever. There's no way to clean them up because pip doesn't know
which packages you explicitly wanted vs which were just pulled in.

## fpm's Solution

fpm maintains a **dependency graph** that tracks:
- **Requested** packages — you explicitly typed `fpm install X`
- **Transitive** packages — installed as dependencies of something else

```json
// ~/.local/share/fpm/depgraph.json (system)
// .venv/.fpm-depgraph.json (per-project)
{
  "packages": {
    "flask": {
      "requested": true,
      "version": "3.1.3",
      "dependencies": ["werkzeug", "jinja2", "click", "blinker", "itsdangerous"]
    },
    "werkzeug": {
      "requested": false,
      "version": "3.1.8",
      "dependencies": ["markupsafe"]
    }
  }
}
```

## How It Works

### On Install

```bash
fpm install -s flask requests
```

fpm resolves the full dependency tree, then records:
- `flask` → **requested** (user asked for it)
- `requests` → **requested** (user asked for it)
- `werkzeug`, `jinja2`, `click`, `urllib3`, `certifi`, etc. → **transitive**

### On Remove with --purge

```bash
fpm remove -sp flask
```

1. Remove flask from site-packages
2. Remove flask from the graph
3. Find graph orphans: transitive packages that no remaining package depends on
4. `werkzeug` → orphan (only flask needed it) → remove
5. `urllib3` → NOT orphan (requests still needs it) → keep
6. Show interactive confirmation before bulk removal

### Autoremove

```bash
fpm autoremove --system
```

Scans the graph for all transitive packages with zero dependents.
Interactive confirmation with exclude option.

### Mark

```bash
fpm mark --requested click     # protect from autoremove
fpm mark --dependency click    # make eligible for autoremove
fpm mark --show click          # check current status
```

Change a package's status. Like:
- `pacman --asexplicit` / `pacman --asdeps`
- `apt-mark manual` / `apt-mark auto`

## Pre-existing Packages

When fpm encounters packages installed before fpm was set up:
- **Other managers' packages (pip, conda, system)** → never touched, never tracked
- **fpm packages without graph entry** → treated as requested (safe default)
- Graph is built incrementally from the first `fpm install`

## Fallback Recovery

If the graph file is lost or corrupted:
1. Falls back to METADATA scan (reads `Requires-Dist` from dist-info)
2. Rebuilds relationships from installed package metadata
3. Marks all found packages as "requested" (safe — prevents accidental removal)

## Tree Visualization

```bash
$ fpm tree --system
● flask 3.1.3
├── blinker 1.9.0
├── click 8.4.1
├── jinja2 3.1.6
│   └── markupsafe 3.0.3
├── werkzeug 3.1.8
│   └── markupsafe 3.0.3
└── itsdangerous 2.2.0
● requests 2.34.2
├── urllib3 2.7.0
├── certifi 2026.5.20
├── charset-normalizer 3.4.7
└── idna 3.18
```

`●` marks requested (top-level) packages. Their deps are shown as a tree.

## Comparison with Other Tools

| Tool | Tracks install reason | Auto-cleans orphans | Mark command |
|------|----------------------|---------------------|--------------|
| **fpm** | Yes (depgraph.json) | `autoremove` / `--purge` | `fpm mark` |
| **pacman** | Yes (local DB) | `pacman -Rns` | `--asexplicit` / `--asdeps` |
| **apt** | Yes (auto/manual) | `apt autoremove` | `apt-mark` |
| **pip** | No | No | No |
| **uv** | No | No | No |
| **npm** | Via package.json | `npm prune` | N/A |
| **cargo** | Via Cargo.toml | No (rebuilds) | N/A |

fpm brings system package manager capabilities (pacman/apt) to Python
for the first time.

## Developer Reference

Key code:
- `internal/depgraph/graph.go` — Graph struct, Load/Save, Orphans(), SyncFromInstalled()
- `internal/cli/mark.go` — `fpm mark` command
- `internal/cli/install_impl.go` — graph update on install (AddRequested/AddTransitive)
- `internal/cli/remove_impl.go` — graph update on remove, purge uses Orphans()
- `internal/cli/tree_impl.go` — `printDepGraphTree()` for system tree

Storage:
- System: `~/.local/share/fpm/depgraph.json`
- Per-venv: `.venv/.fpm-depgraph.json`
