# Scenario 5: Cross-Manager Awareness

## What This Proves

fpm can see packages installed by ANY Python package manager — pip, uv, conda,
poetry, pdm, or system distro packages. It provides a single unified view
regardless of which tool installed what.

No other package manager gives you this visibility.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-cross && mkdir -p /tmp/fpm-demo-cross && cd /tmp/fpm-demo-cross
fpm venv && source .venv/bin/activate
```

---

## Step 1: Install Packages With Different Managers

```bash
# Install via pip
pip install six

# Install via fpm
fpm install requests

# If uv is available:
# uv pip install black
```

---

## Step 2: View fpm-Only Packages

```bash
fpm list
```

**Expected Output:**

```
Package    Version
requests   2.31.0
urllib3    2.1.0
...
```

Only fpm-managed packages shown.

---

## Step 3: View ALL Packages (All Managers)

```bash
fpm list -a
```

**Expected Output:**

```
Package    Version    Manager    Location
requests   2.31.0     fpm        .venv/lib/python3.x/site-packages
urllib3    2.1.0      fpm        .venv/lib/python3.x/site-packages
six        1.16.0     pip        .venv/lib/python3.x/site-packages
```

The `Manager` column tells you exactly who installed each package.

---

## Step 4: Filter by Manager

```bash
fpm list --manager pip
```

**Expected:** Shows only packages installed by pip (six in this case).

```bash
fpm list --manager fpm
```

**Expected:** Shows only fpm-managed packages.

---

## Step 5: How Detection Works (INSTALLER File)

```bash
# fpm writes its identity to INSTALLER
cat .venv/lib/python3.*/site-packages/requests-*.dist-info/INSTALLER
```

**Expected:** `fpm`

```bash
# pip writes its identity too
cat .venv/lib/python3.*/site-packages/six-*.dist-info/INSTALLER
```

**Expected:** `pip`

---

## Step 6: System-Wide Cross-Manager View

```bash
fpm list -a --system
```

**Expected Output (example):**

```
Package          Version    Manager    Location
requests         2.31.0     fpm        /usr/local/lib/python3.x/site-packages
numpy            1.24.0     pip        /usr/local/lib/python3.x/site-packages
python3-apt      2.4.0      system     /usr/lib/python3/dist-packages
```

The `system` manager means it was installed by the OS package manager (apt/yum).

---

## Step 7: pip-Compatible Commands

```bash
# Traditional pip list format (shows ALL packages)
fpm pip list

# requirements.txt format
fpm pip freeze

# Package details
fpm pip show requests
```

**Expected:** Familiar pip output format, but powered by fpm's detection engine.

---

## Step 8: Dependency Tree Across Managers

```bash
fpm tree
```

**Expected:** Shows dependency tree for all packages, indicating which manager
owns each one.

---

## Step 9: Audit Across All Managers

```bash
fpm audit
```

**Expected:** Scans packages from ALL managers for vulnerabilities — not just
fpm's packages. This means pip-installed packages get audited too.

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-cross
```

---

## Key Takeaway

> In the real world, teams use multiple tools. One person uses pip, another uses
> uv, CI uses poetry. fpm doesn't fight this — it embraces it by providing full
> visibility into what's installed, by whom, and whether anything conflicts.
> It's the "single pane of glass" for Python packaging.
