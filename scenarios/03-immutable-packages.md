# Scenario 3: Immutable Package Pinning

## What This Proves

fpm can lock specific packages to exact versions that **cannot** be changed —
not by direct install, not by transitive dependency resolution. This is critical
for production environments where an accidental numpy upgrade could break your
entire ML pipeline.

No other Python package manager has this feature.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-immutable && mkdir -p /tmp/fpm-demo-immutable && cd /tmp/fpm-demo-immutable
fpm venv && source .venv/bin/activate
```

---

## Step 1: Install a Package at Specific Version

```bash
fpm install "requests==2.31.0"
```

**Expected:** requests 2.31.0 installed successfully.

---

## Step 2: Create fpm.toml With Immutable Pin

```bash
cat > fpm.toml << 'EOF'
[immutable]
packages = [
    { name = "requests", version = "2.31.0" },
]
EOF
```

---

## Step 3: Attempt to Install a Different Version (Should Fail)

```bash
fpm install "requests==2.30.0"
```

**Expected Output:**

```
error: cannot install requests==2.30.0: pinned as immutable at 2.31.0 in fpm.toml
```

The install is **blocked**. The immutable pin acts as a hard constraint.

---

## Step 4: Attempt to Upgrade (Should Also Fail)

```bash
fpm install "requests>=2.32.0"
```

**Expected Output:**

```
error: cannot install requests>=2.32.0: pinned as immutable at 2.31.0 in fpm.toml
```

Even "upgrades" are blocked. The version is truly immutable.

---

## Step 5: Pin Multiple Packages

```bash
cat > fpm.toml << 'EOF'
[immutable]
packages = [
    { name = "requests", version = "2.31.0" },
    { name = "urllib3", version = "2.1.0" },
]
EOF
```

Now both requests AND its dependency urllib3 are pinned.

---

## Step 6: Transitive Dependency Conflict (Should Fail)

```bash
# Attempt to install a package that needs a different urllib3 version
fpm install "some-package-that-needs-urllib3>=2.2.0"
```

**Expected Output:**

```
error: cannot install urllib3>=2.2.0 (required by some-package): pinned as immutable at 2.1.0 in fpm.toml
```

Even transitive dependencies are blocked from violating immutable pins.

---

## Step 7: Installing the Exact Pinned Version (Should Succeed)

```bash
fpm install "requests==2.31.0"
```

**Expected:** Succeeds (or reports already installed) — installing the exact
pinned version is always allowed.

---

## Step 8: Remove Immutable Pin to Allow Changes

```bash
# Edit fpm.toml to remove the requests pin
cat > fpm.toml << 'EOF'
[immutable]
packages = [
    { name = "urllib3", version = "2.1.0" },
]
EOF

# Now upgrading requests works
fpm install "requests>=2.32.0"
```

**Expected:** requests upgrades successfully (pin removed).

---

## Real-World Use Case: ML Pipeline Protection

```bash
cat > fpm.toml << 'EOF'
[immutable]
packages = [
    { name = "numpy", version = "1.24.0" },
    { name = "torch", version = "2.1.0" },
    { name = "scipy", version = "1.10.0" },
    { name = "scikit-learn", version = "1.3.0" },
]
EOF
```

With this configuration:

- No one can accidentally upgrade numpy (which would break torch ABI)
- torch stays at the tested version (GPU compatibility)
- scipy/scikit-learn stay at versions compatible with numpy 1.24
- Other packages (requests, flask, etc.) can still upgrade freely

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-immutable
```

---

## Key Takeaway

> Immutable pins are the "do not touch" signs for production dependencies. They
> prevent accidental breakage from direct installs AND transitive resolution.
> This is the difference between "we pinned numpy in requirements.txt" (which
> another install can override) and "numpy CANNOT change period" (which fpm
> enforces at the resolver level).
