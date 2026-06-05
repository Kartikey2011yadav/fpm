# Quick Start

## Create a New Project

```bash
fpm init myproject
cd myproject
```

This creates:

- `pyproject.toml` — project metadata and dependencies
- `.python-version` — pinned Python version
- `.venv/` — virtual environment with activation scripts

## Add Dependencies

```bash
fpm install requests numpy pandas
```

fpm will:

1. Resolve compatible versions (including transitive deps)
2. Check if packages exist in other managers (pip/conda)
3. Download wheels from PyPI
4. Store in content-addressable cache
5. Link into your .venv (zero duplication)
6. Update pyproject.toml and fpm.lock

## Run Your Code

```bash
fpm run python main.py
```

Or activate the environment manually:

```bash
source .venv/bin/activate
python main.py
```

## See What's Installed

```bash
# Packages in your venv
fpm pip list

# All packages across all managers
fpm pip list --all
```

## Save Environment State

```bash
fpm snapshot create "working ML setup"
```

## Lock and Sync

```bash
# Generate lockfile from pyproject.toml
fpm lock

# Sync environment to match lockfile exactly
fpm sync
```

## Share Your Project

Your collaborators only need:

```bash
git clone <your-repo>
cd <your-repo>
fpm sync
```

The lockfile ensures everyone gets identical versions.
