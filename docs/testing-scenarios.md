# Testing Scenarios

Complete testing guide for verifying fpm functionality.

## Quick Test (Automated)

```bash
# Copy test script to container and run
docker cp scripts/test-features.sh fpm-test:/tmp/
docker exec fpm-test bash /tmp/test-features.sh
```

## Setup

```bash
# Fresh container with env var baked in (important: -e flag, not .bashrc)
docker run -d --name fpm-test \
  -e FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev \
  python:3.12-slim sleep infinity

# Install uv for cross-manager testing
docker exec fpm-test pip install uv \
  --trusted-host pypi.org --trusted-host files.pythonhosted.org

# Build and copy fpm
GOOS=linux GOARCH=amd64 go build -o bin/fpm-linux ./cmd/fpm
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm
```

> **Note:** Use `docker run -e` for env vars, not `.bashrc`. Docker exec
> runs a non-login shell that doesn't source `.bashrc`.

---

## Scenario 1: CLI Basics

```bash
fpm -v                    # prints version
fpm --version             # same
fpm -h                    # shows grouped help
fpm install -h            # shows install help with examples
fpm --help | grep system  # --system flag visible
```

---

## Scenario 2: System Install (no venv)

```bash
# Without --system: errors (like uv)
fpm install requests
# → error: No virtual environment found
#   hint: Run `fpm venv` to create an environment, or pass `--system` (`-s`)

# With --system: installs to system Python
fpm install -s requests
fpm install --system flask

# Verify pip sees them
pip list | grep requests
```

---

## Scenario 3: Project Workflow

```bash
mkdir myproject && cd myproject

# Initialize project
fpm init .
# Creates: pyproject.toml, .venv/, .python-version

# Install packages (auto-detects venv, no --system needed)
fpm install requests numpy

# Verify
fpm run python -c "import requests; print(requests.__version__)"
fpm list                 # shows fpm packages only
cat pyproject.toml       # dependencies updated
cat fpm.lock             # lockfile created

# Lock & sync
fpm lock
rm -rf .venv && fpm venv
fpm sync                 # installs from lockfile
```

---

## Scenario 4: List Variants

```bash
fpm list                   # fpm-managed only
fpm list -a                # all managers (fpm + pip + system)
fpm list --manager pip     # filter by manager
fpm list --json            # JSON output
fpm pip list               # all (pip-compatible)
fpm pip freeze --system    # requirements format
fpm pip show --system requests  # package details
```

---

## Scenario 5: Error UX

```bash
# Package not found (with suggestion)
fpm install -s request
# → error: Package "request" not found on PyPI
#   hint: Did you mean "requests"?

# TLS failure (without insecure bypass)
unset FPM_ALLOW_INSECURE_HOST
fpm install -s requests
# → error: TLS certificate verification failed
#   hint: Try --allow-insecure-host pypi.org ...

# Per-host bypass
fpm install --allow-insecure-host pypi.org --allow-insecure-host files.pythonhosted.org -s httpx
```

---

## Scenario 6: Environment Snapshots

```bash
cd myproject  # must be in a project with .venv

# Create snapshot
fpm install requests flask
fpm snapshot create "base setup"

# Add more
fpm install pandas
fpm snapshot create "added pandas"

# List
fpm snapshot list
# Shows: ID, timestamp, package count, message

# Diff
fpm snapshot diff <id1> <id2>
# Shows: + pandas, + python-dateutil, + six, etc.

# Restore
fpm snapshot restore <id1>
# Removes pandas, keeps requests/flask

# Verify
fpm run python -c "import requests"   # works
fpm run python -c "import pandas"     # fails (removed)
```

---

## Scenario 7: Audit

```bash
fpm audit --system           # scan system packages
fpm audit                    # scan venv packages (in project)
# Shows vulnerabilities with ID, severity, fix version
# Second run is instant (OSV cache, 1hr TTL)
```

---

## Scenario 8: Dependency Tree

```bash
cd myproject
fpm install flask
fpm tree
# flask 3.1.3
# ├── werkzeug 3.1.8
# ├── jinja2 3.1.6
# │   └── markupsafe 3.0.3
# ├── click 8.4.1
# └── ...

fpm tree --depth 1     # limit depth
fpm tree --invert      # reverse deps
```

---

## Scenario 9: Python Version Management

```bash
fpm python list              # shows system + managed Pythons
fpm python install 3.11      # downloads from python-build-standalone
fpm python list              # now shows 3.11 as managed
fpm python use 3.11 --system # updates ~/.local/bin symlinks
fpm python pin 3.11          # writes .python-version
```

---

## Scenario 10: Tool Management

```bash
fpm tool install black       # isolated venv + symlink
fpm tool list                # shows installed tools
fpm tool run ruff --version  # ephemeral (cached after first run)
fpm tool run ruff --version  # instant (reuses cached env)
fpm tool uninstall black     # removes
```

---

## Scenario 11: Config & Repair

```bash
fpm config show              # all directories + settings
fpm config set cache.dir /tmp/fpm-cache  # change setting
fpm repair                   # check + fix issues
```

---

## Scenario 12: Cache Management

```bash
fpm cache size               # show breakdown
fpm cache gc                 # remove unreferenced
fpm cache clean              # remove everything
```

---

## Scenario 13: Remove & Aliases

```bash
fpm remove flask             # removes from pyproject.toml
fpm uninstall numpy          # alias for remove
fpm rm click                 # short alias
```

---

## Scenario 14: Cross-Manager Detection

```bash
# Install via pip
pip install httpx

# fpm detects it
fpm list -a | grep httpx     # shows with manager "pip"

# Install via fpm
fpm install -s requests

# Both visible
fpm list -a                  # shows pip + fpm packages
```

---

## Scenario 15: Global Flags

```bash
fpm list --json              # JSON output
NO_COLOR=1 fpm list          # no ANSI colors
fpm install -s --verbose requests  # verbose output
fpm install -s --quiet requests    # silent except errors
```

---

## Verification Checklist

After all scenarios:

- [ ] `fpm list -a` shows correct managers for each package
- [ ] `fpm -v` prints version
- [ ] `fpm -h` shows grouped commands
- [ ] `fpm install` without venv/--system errors clearly
- [ ] `fpm repair` finds and fixes issues
- [ ] `fpm config show` displays all paths
- [ ] No `fpm.lock` generated outside projects
- [ ] Snapshots scoped to venv (not system)
- [ ] Audit uses cache on second run
- [ ] Tool run reuses cached environment
- [ ] Error messages show hints with actionable commands
