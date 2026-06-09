# Scenario 6: Conflict Resolution Policies

## What This Proves

When fpm detects that another manager already has a package installed (possibly
at a different version), it doesn't silently overwrite or fail. It applies a
configurable policy: ask the user, always install, or always skip.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-conflict && mkdir -p /tmp/fpm-demo-conflict && cd /tmp/fpm-demo-conflict
fpm venv && source .venv/bin/activate

# Pre-install a package via pip (simulates existing environment)
pip install "six==1.16.0"
```

---

## Policy 1: "ask" (Default — Interactive)

### Step 1: Configure Ask Policy

```bash
cat > fpm.toml << 'EOF'
[tool]
cross-manager-policy = "ask"
EOF
```

### Step 2: Install Same Package Via fpm

```bash
fpm install six
```

**Expected Output (interactive prompt):**

```
six 1.16.0 is already installed via pip.

  [1] Skip installation (keep pip's 1.16.0)
  [2] Install anyway (fpm's version will take priority)
  [3] Abort

Choice [1/2/3]:
```

This gives you full control over what happens.

---

## Policy 2: "install" (Always Install — For CI/CD)

### Step 1: Configure Install Policy

```bash
cat > fpm.toml << 'EOF'
[tool]
cross-manager-policy = "install"
EOF
```

### Step 2: Install Same Package

```bash
fpm install six
```

**Expected Output:**

```
six 1.16.0 already installed via pip — installing fpm version (policy: install)
  + six 1.16.0 (fpm)
```

No prompt. fpm installs its version, which shadows the pip version in the Python
path.

**When to use:** CI/CD pipelines where interactive prompts aren't possible.

---

## Policy 3: "skip" (Preserve Existing — Conservative)

### Step 1: Configure Skip Policy

```bash
cat > fpm.toml << 'EOF'
[tool]
cross-manager-policy = "skip"
EOF
```

### Step 2: Install Same Package

```bash
fpm install six
```

**Expected Output:**

```
six 1.16.0 already installed via pip — skipping (policy: skip)
```

No installation happens. The existing pip version is preserved.

**When to use:** Teams migrating from pip/poetry who want fpm to defer to
existing packages.

---

## Scenario: Version Mismatch Conflict

```bash
# pip has an older version
pip install "urllib3==1.26.0"

# fpm wants a newer version (as dependency of requests)
cat > fpm.toml << 'EOF'
[tool]
cross-manager-policy = "ask"
EOF

fpm install requests
```

**Expected (with ask policy):**

```
urllib3 1.26.0 is already installed via pip, but requests requires urllib3>=2.0.

  [1] Skip (keep pip's 1.26.0 — may cause compatibility issues)
  [2] Install fpm's urllib3 2.x.x (will shadow pip's version)
  [3] Abort

Choice [1/2/3]:
```

---

## Scenario: Force Remove Another Manager's Package

```bash
# Try to remove pip's package with fpm
fpm remove six
```

**Expected:**

```
error: Cannot remove six: installed by pip (not managed by fpm)
hint: Use --force (-f) to remove regardless of manager
```

```bash
# Force removal
fpm remove -f six
```

**Expected:** Package removed regardless of which manager installed it.

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-conflict
```

---

## Key Takeaway

> Cross-manager conflicts are inevitable in real projects. fpm handles them
> gracefully with three policies: ask (safe default), install (CI-friendly),
> skip (conservative). This prevents the silent corruption that happens when pip
> and poetry fight over the same package.
