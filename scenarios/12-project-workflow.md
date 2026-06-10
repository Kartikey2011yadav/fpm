# Scenario 12: Project Initialization & Full Workflow

## What This Proves

fpm provides a complete project lifecycle — from initialization to build and
publish. It manages pyproject.toml, lockfiles, virtual environments, and
dependencies as a unified workflow. No activation needed — just `cd` into your
project.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-project && mkdir -p /tmp/fpm-demo-project && cd /tmp/fpm-demo-project
```

---

## Step 1: Initialize a New Project

```bash
fpm init myapp
cd myapp
```

**Expected:** Creates the following structure:

```
myapp/
├── pyproject.toml      # PEP 621 project metadata
├── fpm.toml            # fpm configuration (optional)
├── .venv/              # Virtual environment (auto-created)
├── .python-version     # Python version pin
└── .fpm-depgraph.json  # Dependency graph (empty)
```

> No activation needed! Just `cd myapp` and all fpm commands target the venv.

---

## Step 2: Add Dependencies

```bash
fpm install requests flask
```

**Expected:** Packages installed AND pyproject.toml updated:

```bash
cat pyproject.toml | grep -A5 "dependencies"
```

```toml
dependencies = [
    "requests>=2.34.2",
    "flask>=3.1.3",
]
```

---

## Step 3: Add Development Dependencies

```bash
fpm install --dev pytest black mypy
```

**Expected:** Dev dependencies added to a separate group:

```toml
[project.optional-dependencies]
dev = [
    "pytest>=7.0",
    "black>=24.0",
    "mypy>=1.0",
]
```

---

## Step 4: Configure Immutable Pins

```bash
cat > fpm.toml << 'EOF'
[project]
name = "myapp"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
EOF
```

Now `requests` can never be changed from 2.34.2, regardless of what anyone runs.

---

## Step 5: Write Some Code

```bash
mkdir -p src/myapp
cat > src/myapp/__init__.py << 'EOF'
import requests

def get_status(url: str) -> int:
    """Return HTTP status code for a URL."""
    response = requests.get(url, timeout=10)
    return response.status_code
EOF
```

---

## Step 6: Run Code in Managed Environment

```bash
fpm run python -c "from myapp import get_status; print(get_status('https://httpbin.org/status/200'))"
```

**Expected:** `200`

> `fpm run` handles PATH and environment — no manual activation needed.

---

## Step 7: Check Package Status

```bash
# See all packages with mutable/immutable status
fpm list --mutable

# See dependency tree
fpm tree
```

**Expected:**

```
Package         Version    Manager  Pinned      Location
requests        2.34.2     fpm      🔒 2.34.2   .venv/...
flask           3.1.3      fpm      mutable     .venv/...
certifi         2026.5.20  fpm      mutable     .venv/...
...
```

---

## Step 8: Generate Lockfile

```bash
fpm lock
cat fpm.lock | head -20
```

**Expected:** Deterministic lockfile with all resolved versions and hashes.

---

## Step 9: Snapshot Before Experiment

```bash
fpm snapshot create "stable baseline v0.1"
```

This captures: all packages, all versions, AND your fpm.toml config.

---

## Step 10: Experiment With New Dependencies

```bash
fpm install pandas numpy
fpm run python -c "import pandas; print(pandas.__version__)"
```

---

## Step 11: Experiment Failed — Rollback

```bash
SNAP_ID=$(fpm snapshot list | grep baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore "$SNAP_ID"
fpm list
```

**Expected:** pandas and numpy gone. Only requests and flask remain. fpm.toml is
also restored to its original state.

---

## Step 12: Audit for Vulnerabilities

```bash
fpm audit
```

**Expected:** Scans all packages against OSV database, reports any CVEs.

---

## Step 13: Build the Package

```bash
fpm build
```

**Expected:**

```
Building myapp 0.1.0...
  Created: dist/myapp-0.1.0-py3-none-any.whl
  Created: dist/myapp-0.1.0.tar.gz
```

---

## Step 14: Full CI Reproduction

```bash
# Simulate CI: fresh clone with only lockfile
cd /tmp
rm -rf myapp-ci && mkdir myapp-ci && cd myapp-ci

# Copy only the project files (not .venv)
cp /tmp/fpm-demo-project/myapp/pyproject.toml .
cp /tmp/fpm-demo-project/myapp/fpm.lock .
cp -r /tmp/fpm-demo-project/myapp/src .

# Reproduce exact environment (no activation needed)
fpm venv
fpm sync

# Verify
fpm list
fpm run python -c "from myapp import get_status; print('Import OK')"
```

**Expected:** Exact same versions as the original environment. Fully
reproducible.

---

## Step 15: Repair and Diagnostics

```bash
fpm config show     # View all configuration
fpm repair          # Check installation health
fpm cache size      # See cache disk usage
```

---

## Complete Workflow Summary

```bash
# Day 1: Start project
fpm init myapp && cd myapp
fpm install requests flask
fpm install --dev pytest
fpm lock
fpm snapshot create "initial setup"

# Day 2: Add feature
fpm install pandas
fpm lock
fpm snapshot create "added data processing"

# Day 3: Something broke — rollback
fpm snapshot restore <day1-id>
fpm lock

# Day N: Ship it
fpm audit          # check for vulns
fpm build          # create wheel
fpm publish        # upload to PyPI

# CI: Reproduce exactly (no activation needed)
fpm venv && fpm sync
fpm run pytest
```

---

## Cleanup

```bash
rm -rf /tmp/fpm-demo-project /tmp/myapp-ci
```

---

## Key Takeaway

> fpm provides the complete project lifecycle in one tool: init, install, lock,
> snapshot, audit, build, publish. No activation needed — just `cd` into your
> project. Every step is reproducible and reversible. Combined with snapshots,
> immutable pins, and the `--mutable` flag for visibility, you get a workflow
> that's both fast to develop with and safe to deploy from.
