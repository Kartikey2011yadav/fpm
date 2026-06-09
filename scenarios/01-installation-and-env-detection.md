# Scenario 1: Installation & Environment Detection

## What This Proves

fpm enforces clear boundaries between virtual environments and system Python.
Unlike pip (which silently pollutes system Python), fpm requires explicit intent
via `--system` or an active venv.

---

## Setup

```bash
# Ensure no venv is active
deactivate 2>/dev/null

# Remove any existing test venv
rm -rf /tmp/fpm-demo-env
mkdir -p /tmp/fpm-demo-env && cd /tmp/fpm-demo-env
```

---

## Step 1: Install Without Venv (Should Fail)

```bash
fpm install requests
```

**Expected Output:**

```
error: No virtual environment found
hint: Run `fpm venv` to create an environment, or pass `--system` (`-s`)
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
Resolving dependencies...
Downloading requests-2.x.x-py3-none-any.whl
Installing requests 2.x.x (system)
  + requests 2.x.x
  + urllib3 2.x.x
  + charset-normalizer 3.x.x
  + idna 3.x
  + certifi 2024.x.x
```

---

## Step 3: Verify Installation

```bash
fpm list
```

**Expected:** Shows requests and its dependencies with versions.

```bash
fpm tree --system
```

**Expected:** Tree showing requests as root with urllib3, charset-normalizer,
idna, certifi as children.

---

## Step 4: Create a Venv and Install There

```bash
fpm venv
source .venv/bin/activate

fpm install flask
```

**Expected:** Flask installs into `.venv/` without the `--system` flag.

```bash
fpm list
```

**Expected:** Shows flask + dependencies (werkzeug, jinja2, click, etc.)

---

## Step 5: Verify Isolation

```bash
# Venv packages don't appear in system
fpm list --system  # should NOT show flask

# System packages don't appear in venv
fpm list           # should NOT show requests (unless it's also in venv)
```

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-env
fpm remove -s requests
```

---

## Key Takeaway

> fpm never installs into the wrong place by accident. You either activate a
> venv or explicitly pass `--system`. This prevents the #1 source of Python
> environment corruption.
