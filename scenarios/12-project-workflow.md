# Scenario 12: Project Initialization & Full Workflow

## What This Proves

fpm provides a complete project lifecycle — from initialization to build and
publish. It manages pyproject.toml, lockfiles, virtual environments, and
dependencies as a unified workflow.

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
├── .venv/              # Virtual environment
├── .python-version     # Python version pin
└── .fpm-depgraph.json  # Dependency graph (empty)
```

```bash
cat pyproject.toml
```

**Expected Output:**

```toml
[project]
name = "myapp"
version = "0.1.0"
description = ""
requires-python = ">=3.10"
dependencies = []

[build-system]
requires = ["hatchling"]
build-backend = "hatchling.build"
```

---

## Step 2: Activate Environment

```bash
source .venv/bin/activate
python --version
```

**Expected:** Python version matches `.python-version`.

---

## Step 3: Add Dependencies

```bash
fpm install requests flask
```

**Expected:** Packages installed AND pyproject.toml updated:

```bash
cat pyproject.toml | grep -A5 "dependencies"
```

**Expected:**

```toml
dependencies = [
    "requests>=2.31.0",
    "flask>=3.0.0",
]
```

---

## Step 4: Add Development Dependencies

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

---

## Step 7: Generate Lockfile

```bash
fpm lock
cat fpm.lock | head -20
```

**Expected:** Deterministic lockfile with all resolved versions and hashes.

---

## Step 8: Snapshot Before Experiment

```bash
fpm snapshot create "stable baseline v0.1"
```

---

## Step 9: Experiment With New Dependencies

```bash
fpm install pandas numpy
fpm run python -c "import pandas; print(pandas.__version__)"
```

---

## Step 10: Decide Experiment Failed — Rollback

```bash
fpm snapshot restore <baseline-id>
fpm list
```

**Expected:** pandas and numpy gone. Only requests and flask remain.

---

## Step 11: Build the Package

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

## Step 12: Full CI Reproduction

```bash
# Simulate CI: fresh clone with only lockfile
cd /tmp
rm -rf myapp-ci && mkdir myapp-ci && cd myapp-ci

# Copy only the project files (not .venv)
cp /tmp/fpm-demo-project/myapp/pyproject.toml .
cp /tmp/fpm-demo-project/myapp/fpm.lock .
cp -r /tmp/fpm-demo-project/myapp/src .

# Reproduce exact environment
fpm venv
source .venv/bin/activate
fpm sync

# Verify
fpm list
python -c "from myapp import get_status; print('Import OK')"
```

**Expected:** Exact same versions as the original environment. Fully
reproducible.

---

## Step 13: Configuration

```bash
fpm config show
```

**Expected Output:**

```
Configuration:
  cross-manager-policy: ask
  link-mode: auto (reflink on APFS)
  concurrency: 50
  cache-dir: ~/.cache/fpm
  data-dir: ~/.local/share/fpm

Project:
  python: 3.12.0
  venv: .venv
  lockfile: fpm.lock
  immutable: (none)
```

---

## Complete Workflow Summary

```bash
# Day 1: Start project
fpm init myapp && cd myapp
source .venv/bin/activate
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

# CI: Reproduce exactly
fpm venv && fpm sync
```

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-project /tmp/myapp-ci
```

---

## Key Takeaway

> fpm provides the complete project lifecycle in one tool: init, install, lock,
> snapshot, audit, build, publish. Every step is reproducible and reversible.
> Combined with snapshots and immutable pins, you get a workflow that's both
> fast to develop with and safe to deploy from.
