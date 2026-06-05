# Testing Scenarios

Complete testing guide for verifying fpm functionality in a Docker environment.

## Setup

```bash
# Build the test image
docker build -f Dockerfile.test -t fpm-test .

# Run interactive shell
docker run -it --rm fpm-test bash

# Verify all tools available
fpm version
python3 --version
pip3 --version
uv --version
conda --version
```

---

## Scenario 1: Project Initialization

**Test:** Create a new project from scratch.

```bash
# Create project
fpm init myproject
cd myproject

# Verify files created
ls -la
# Expected: pyproject.toml, .python-version, .venv/

cat pyproject.toml
# Expected: [project] with name, version, requires-python, dependencies

cat .python-version
# Expected: Python version (e.g., "3.11")

ls .venv/bin/
# Expected: python3, activate, activate.fish, etc.

# Verify venv works
source .venv/bin/activate
python --version
deactivate
```

**Expected:** All files created, venv activates successfully.

---

## Scenario 2: Package Installation

**Test:** Install packages, verify they resolve and install correctly.

```bash
cd myproject

# Install a simple package
fpm install requests
# Expected: resolves dependencies (urllib3, certifi, etc.), downloads, installs

# Verify installation
fpm run python -c "import requests; print(requests.__version__)"
# Expected: prints version number

# Verify in list
fpm pip list
# Expected: shows requests + deps with "fpm" as manager

# Verify pyproject.toml updated
cat pyproject.toml
# Expected: dependencies includes "requests"

# Verify lockfile created
cat fpm.lock
# Expected: TOML lockfile with all resolved packages + hashes
```

---

## Scenario 3: Version Constraints

```bash
# Install specific version
fpm install "numpy==1.24.0"
fpm run python -c "import numpy; print(numpy.__version__)"
# Expected: "1.24.0"

# Install with range
fpm install "flask>=2.0,<3.0"
fpm run python -c "import flask; print(flask.__version__)"
# Expected: some 2.x version

# Install with extras
fpm install "requests[security]"
```

---

## Scenario 4: Package Removal

```bash
# Remove a package
fpm remove requests

# Verify removed from environment
fpm run python -c "import requests" 2>&1
# Expected: ModuleNotFoundError

# Verify removed from pyproject.toml
grep requests pyproject.toml
# Expected: no output (not found)

# Verify files cleaned
ls .venv/lib/python*/site-packages/ | grep requests
# Expected: no output (removed)
```

---

## Scenario 5: Lock and Sync

```bash
# Add dependencies to pyproject.toml manually
cat >> pyproject.toml << 'EOF'
dependencies = ["flask>=2.0", "click"]
EOF

# Generate lockfile
fpm lock
# Expected: resolves all deps, writes fpm.lock

# Delete venv and recreate
rm -rf .venv
fpm venv

# Sync from lockfile
fpm sync
# Expected: installs exactly what's in lockfile

# Verify
fpm run python -c "import flask; print(flask.__version__)"
```

---

## Scenario 6: Cross-Manager Coexistence

**Test:** fpm detects packages from pip and handles conflicts.

```bash
# Install via pip first
pip install numpy==1.24.0

# Now try fpm install of same version
fpm install numpy==1.24.0
# Expected: "numpy 1.24.0 is already installed via pip — skipping download"

# Try different version
fpm install numpy==1.26.0
# Expected: prompts about existing 1.24.0 from pip, asks to install/skip/abort

# See all packages from all managers
fpm pip list --all
# Expected: shows packages with correct manager attribution (pip/fpm/system)
```

---

## Scenario 7: Cross-Manager with uv

```bash
# Install via uv
uv pip install black

# Check fpm detects it
fpm pip list --all
# Expected: shows black with manager "uv"

# Install via fpm
fpm install ruff

# Both should be visible
fpm pip list --all
# Expected: black (uv) + ruff (fpm)
```

---

## Scenario 8: Cross-Manager with conda

```bash
# Install via conda
conda install -y scipy

# Check fpm detects it
fpm pip list --all
# Expected: shows scipy with manager "conda"
```

---

## Scenario 9: Environment Snapshots

```bash
cd myproject

# Create initial snapshot
fpm install requests flask
fpm snapshot create "base setup"

# Add more packages
fpm install pandas numpy
fpm snapshot create "added data science"

# View history
fpm snapshot list
# Expected: 2 snapshots with timestamps and messages

# Compare snapshots
fpm snapshot diff <id1> <id2>
# Expected: shows added packages

# Restore first snapshot
fpm snapshot restore <id1>
# Expected: removes pandas/numpy, keeps requests/flask

# Verify state
fpm run python -c "import requests; print('ok')"  # works
fpm run python -c "import pandas" 2>&1            # fails (removed)
```

---

## Scenario 10: Snapshot with External Manager Changes

```bash
# Take snapshot with pip package
pip install httpx
fpm snapshot create "with httpx from pip"

# Pip changes something
pip install httpx==0.25.0  # different version

# Compare
fpm snapshot diff <id>
# Expected: shows httpx version drift

# Restore
fpm snapshot restore <id>
# Expected: warns about httpx version mismatch from pip
```

---

## Scenario 11: Python Version Management

```bash
# List available
fpm python list
# Expected: shows system Python

# Install a version
fpm python install 3.11
# Expected: downloads and installs

# Create venv with specific Python
fpm venv --python 3.11 .venv-311
# Expected: creates venv using 3.11

# Verify
.venv-311/bin/python --version
# Expected: Python 3.11.x

# Pin for project
fpm python pin 3.11
cat .python-version
# Expected: "3.11"
```

---

## Scenario 12: Virtual Environment Isolation

```bash
# System Python version
python3 --version
# Expected: system version (e.g., 3.10)

# Create venv with different Python
fpm venv --python 3.12

# Venv has different Python
.venv/bin/python --version
# Expected: 3.12.x (different from system)

# System Python unchanged
python3 --version
# Expected: still system version
```

---

## Scenario 13: Cache Management

```bash
# Install some packages to populate cache
fpm install requests numpy flask

# Check cache size
fpm cache size
# Expected: shows non-zero sizes for CAS, wheels, etc.

# List unused (after removing from env)
fpm remove numpy
fpm cache list-unused
# Expected: numpy appears as unreferenced

# Garbage collect
fpm cache gc
# Expected: removes unreferenced entries, shows freed space

# Verify
fpm cache size
# Expected: smaller than before
```

---

## Scenario 14: Tool Management

```bash
# Install a CLI tool
fpm tool install black
# Expected: creates isolated env, installs black

# Verify tool works
fpm tool run black --version
# Expected: shows black version

# List tools
fpm tool list
# Expected: shows black with entrypoints

# Direct execution (if bin is on PATH)
black --version

# Uninstall
fpm tool uninstall black
fpm tool list
# Expected: empty
```

---

## Scenario 15: Vulnerability Auditing

```bash
# Install packages (some may have known vulns)
fpm install requests==2.25.0

# Audit
fpm audit
# Expected: may report vulnerabilities for older versions
# Shows: ID, severity, summary, fix version
```

---

## Scenario 16: Dependency Tree

```bash
fpm install flask

fpm tree
# Expected:
# flask 2.3.0
# ├── werkzeug 2.3.0
# ├── jinja2 3.1.2
# │   └── markupsafe 2.1.3
# ├── click 8.1.7
# └── ...

fpm tree --invert
# Expected: shows reverse deps (who requires what)
```

---

## Scenario 17: Build and Publish

```bash
# Create a simple package
mkdir -p src/mylib
cat > src/mylib/__init__.py << 'EOF'
__version__ = "0.1.0"
EOF

cat > pyproject.toml << 'EOF'
[project]
name = "mylib"
version = "0.1.0"
[build-system]
requires = ["setuptools"]
build-backend = "setuptools.build_meta"
EOF

# Build
fpm build
# Expected: creates dist/mylib-0.1.0-py3-none-any.whl and .tar.gz

ls dist/
# Expected: .whl and .tar.gz files

# Publish (to test PyPI)
fpm publish --repository testpypi --token $TEST_PYPI_TOKEN
```

---

## Scenario 18: Global vs Local Scope

```bash
# Local install (default)
fpm install requests
# Only visible in current venv

# Global install
fpm install httpie --global
# Available system-wide

# Verify isolation
deactivate
python3 -c "import httpie"  # works (global)
python3 -c "import requests"  # may fail (local only)
```

---

## Scenario 19: Immutable Packages

```bash
# Configure immutable package
cat > fpm.toml << 'EOF'
[project]
name = "test"
dependencies = []

[immutable]
packages = [
    { name = "numpy", version = "1.24.0" }
]
EOF

# Install the pinned version
fpm install numpy==1.24.0
# Expected: installs successfully

# Try to install different version
fpm install numpy==2.0.0
# Expected: ERROR — numpy is pinned as immutable at 1.24.0

# Install something that depends on a different numpy
fpm install scipy
# Expected: resolver finds scipy version compatible with numpy 1.24.0
# OR: clear error explaining the conflict
```

---

## Scenario 20: Self-Update

```bash
fpm version
# Expected: current version

fpm self update
# Expected: checks GitHub, downloads if newer version available
```

---

## Scenario 21: Error Handling

```bash
# Non-existent package
fpm install this-package-definitely-does-not-exist-12345
# Expected: clear error message with hint

# Invalid version specifier
fpm install "numpy==invalid"
# Expected: parse error with helpful message

# No venv found
cd /tmp
fpm install requests
# Expected: "no virtual environment found. Run 'fpm init' or 'fpm venv' first"

# Network failure (disconnect network)
fpm install requests
# Expected: clear timeout/connection error
```

---

## Scenario 22: PEP 723 Script Support

```bash
cat > script.py << 'EOF'
# /// script
# dependencies = ["requests"]
# requires-python = ">=3.10"
# ///
import requests
print(requests.get("https://httpbin.org/get").status_code)
EOF

fpm run script.py
# Expected: installs requests in ephemeral env, runs script, prints 200
```

---

## Scenario 23: Concurrent Operations (stress test)

```bash
# Multiple installs in parallel
fpm install requests numpy pandas scipy matplotlib &
fpm install flask django celery &
wait
# Expected: no corruption, all packages installed correctly
```

---

## Scenario 24: Large Dependency Tree

```bash
# Install something with many transitive deps
fpm install jupyterlab
# Expected: resolves 50+ packages, installs all correctly

fpm tree --depth 2
# Expected: shows full tree without errors
```

---

## Verification Checklist

After all scenarios, verify:

- [ ] `fpm pip list --all` shows all packages with correct managers
- [ ] `fpm cache size` shows reasonable numbers
- [ ] `fpm snapshot list` shows all created snapshots
- [ ] No orphaned files in site-packages
- [ ] .venv/bin/ has correct symlinks
- [ ] pyproject.toml reflects current state
- [ ] fpm.lock is consistent with installed packages
- [ ] No permission errors
- [ ] Colored output works (and `NO_COLOR=1` disables it)
- [ ] `--json` flag produces valid JSON
- [ ] `--quiet` suppresses output
- [ ] `--verbose` shows debug info
