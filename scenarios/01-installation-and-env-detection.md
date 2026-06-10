# Scenario 1: Installation & Environment Detection

## What This Proves

fpm enforces clear boundaries between virtual environments and system Python.
Unlike pip (which silently pollutes system Python), fpm requires explicit intent
via `--system` or being in a project directory. No activation needed — fpm
detects your project by directory alone.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-env /tmp/fpm-demo-outside
mkdir -p /tmp/fpm-demo-outside && cd /tmp/fpm-demo-outside
```

---

## Step 1: Install Without Venv or --system (Should Fail)

```bash
cd /tmp/fpm-demo-outside
fpm install requests
```

**Expected Output:**

```
error: No virtual environment found

hint: Run `fpm venv` to create an environment, or pass `--system` (`-s`) to
      install into the system Python.
```

**Why this matters:** pip would silently install into system Python, potentially
breaking OS tools. fpm forces you to be explicit.

---

## Step 2: Install With --system Flag

```bash
fpm install -s requests
```

**Expected Output:**

```
Resolving 1 package(s)... done
✓ requests 2.34.2
✓ urllib3 2.7.0
✓ charset-normalizer 3.4.7
✓ idna 3.18
✓ certifi 2026.5.20
✓ Installed 5 package(s)
```

---

## Step 3: Verify System Installation

```bash
fpm list --system
fpm tree --system
```

**Expected:** Shows requests and its dependencies in system site-packages.

---

## Step 4: Create a Project (Auto-Detects Venv)

```bash
rm -rf /tmp/fpm-demo-env && mkdir /tmp/fpm-demo-env && cd /tmp/fpm-demo-env
fpm init .
```

**Expected:** Creates `pyproject.toml` + `.venv`. No activation required.

```bash
fpm install flask
```

**Expected:** Flask installs into `.venv/` — no `--system` flag needed because
fpm detected the project by directory.

```bash
fpm list
```

**Expected:** Shows flask + dependencies (werkzeug, jinja2, click, etc.)

---

## Step 5: Verify Isolation

```bash
# Inside project: sees venv packages
fpm list
# Shows: flask, werkzeug, jinja2, click, ...

# System packages are separate
fpm list --system
# Shows: requests, urllib3, certifi, ...

# Leave the project directory
cd /tmp/fpm-demo-outside

# fpm no longer sees the venv
fpm list
# error: no virtual environment found. Use --system to list system packages

# System still accessible with --system
fpm list --system
# Shows: requests, urllib3, certifi, ...
```

---

## Step 6: Directory-Based Detection (No Activation)

```bash
# Go back into project — venv is detected automatically
cd /tmp/fpm-demo-env
fpm list
# Shows flask packages — no source .venv/bin/activate needed!

# VIRTUAL_ENV env var is ignored by fpm
export VIRTUAL_ENV="/tmp/fpm-demo-env/.venv"
cd /tmp
fpm list
# error: no virtual environment found (VIRTUAL_ENV ignored)
unset VIRTUAL_ENV
```

**Why:** fpm uses directory-based detection only (like uv's project commands).
This prevents accidentally installing to the wrong project's venv from another
directory.

---

## Cleanup

```bash
rm -rf /tmp/fpm-demo-env /tmp/fpm-demo-outside
fpm remove -s requests
echo "a" | fpm autoremove --system
```

---

## Key Takeaway

> fpm never installs into the wrong place by accident. You either `cd` into a
> project directory (auto-detects `.venv`) or explicitly pass `--system`. No
> activation needed, no VIRTUAL_ENV env var, no forgotten deactivations. Just
> directory = environment.
