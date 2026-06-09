# Scenario 11: Error UX & Helpful Hints

## What This Proves

fpm never shows a cryptic error. Every failure includes a clear explanation and
an actionable suggestion. This dramatically reduces time-to-fix for common
mistakes compared to pip's wall-of-text tracebacks.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-errors && mkdir -p /tmp/fpm-demo-errors && cd /tmp/fpm-demo-errors
```

---

## Error 1: Package Typo (Did-You-Mean)

```bash
fpm install -s request
```

**Expected Output:**

```
error: Package "request" not found on PyPI
hint: Did you mean "requests"?
      Run: fpm install -s requests
```

Compare to pip: `ERROR: No matching distribution found for request` (no
suggestion).

---

## Error 2: No Venv Found

```bash
# Ensure no venv is active
deactivate 2>/dev/null

fpm install requests
```

**Expected Output:**

```
error: No virtual environment found
hint: Run `fpm venv` to create an environment, or pass `--system` (`-s`)
      to install system-wide.
```

Compare to pip: installs into system Python without warning.

---

## Error 3: Removing Another Manager's Package

```bash
# Install via pip first
fpm venv && source .venv/bin/activate
pip install six

# Try to remove with fpm
fpm remove six
```

**Expected Output:**

```
error: Cannot remove six: installed by pip (not managed by fpm)
hint: Use --force (-f) to remove regardless of which manager installed it.
      Run: fpm remove -f six
```

---

## Error 4: Immutable Pin Violation

```bash
cat > fpm.toml << 'EOF'
[immutable]
packages = [{ name = "requests", version = "2.31.0" }]
EOF

fpm install "requests==2.28.0"
```

**Expected Output:**

```
error: cannot install requests==2.28.0: pinned as immutable at 2.31.0 in fpm.toml
hint: To change this version, edit [immutable] in fpm.toml first.
```

---

## Error 5: TLS/Network Issues

```bash
# Simulate TLS failure (e.g., corporate proxy)
fpm install --index-url https://broken-proxy.internal/simple requests
```

**Expected Output:**

```
error: TLS certificate verification failed for broken-proxy.internal
hint: Your network may be intercepting HTTPS connections (VPN/corporate proxy).
      Try: fpm install --allow-insecure-host broken-proxy.internal requests
      Or set: export FPM_ALLOW_INSECURE_HOST=broken-proxy.internal
```

Compare to pip: `ssl.SSLCertVerificationError: [SSL: CERTIFICATE_VERIFY_FAILED]`
with no suggestion.

---

## Error 6: Version Not Found

```bash
fpm install "requests==99.99.99"
```

**Expected Output:**

```
error: No version of requests matches ==99.99.99
hint: Available versions: 2.31.0, 2.30.0, 2.29.0, ...
      Run: fpm install "requests>=2.31.0" for the latest compatible version.
```

---

## Error 7: Dependency Conflict

```bash
fpm install "urllib3==1.26.0" "requests>=2.31.0"
```

**Expected Output:**

```
error: Dependency conflict
  requests 2.31.0 requires urllib3>=2.0
  but you requested urllib3==1.26.0

hint: These requirements are incompatible. Either:
      - Remove the urllib3 version constraint
      - Use an older requests version: fpm install "requests<2.29"
```

Compare to pip:
`ERROR: Cannot install requests==2.31.0 and urllib3==1.26.0 because these package versions have conflicting dependencies.`
(no suggestion).

---

## Error 8: Python Version Incompatibility

```bash
fpm install "some-package-requiring-python>=3.13"
```

**Expected Output:**

```
error: some-package 2.0.0 requires Python >=3.13, but you have Python 3.12.0
hint: Install a compatible Python version:
      fpm python install 3.13
      fpm python use 3.13
```

---

## Error 9: Disk Space

```bash
# When disk is nearly full
fpm install torch  # ~2GB package
```

**Expected Output:**

```
error: Insufficient disk space
  Required: 2.1 GB (torch 2.1.0)
  Available: 0.5 GB

hint: Free up space with: fpm cache gc
      Or clean wheel cache: fpm cache clean
```

---

## Error 10: Repair Suggestions

```bash
fpm repair
```

**Expected Output:**

```
Checking fpm health...

✓ Cache directory exists
✓ CAS integrity OK
✓ Python found at /usr/bin/python3
✗ fpm bin directory not in PATH

Fix: Add to your shell profile:
  export PATH="$HOME/.local/share/fpm/bin:$PATH"

1 issue found, 1 auto-fixable.
Run `fpm repair --fix` to apply fixes.
```

---

## Design Principles

fpm error messages follow these rules:

1. **What went wrong** — one line, plain English
2. **Why it happened** — context (if not obvious)
3. **How to fix it** — concrete command to run

Never:

- Show stack traces to end users
- Use jargon without explanation
- Leave the user wondering what to do next

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-errors
```

---

## Key Takeaway

> Good error messages save hours of debugging. fpm treats every error as an
> opportunity to guide the user to the solution. The "hint" line is not a
> suggestion — it's the actual command you should run. This is the UX difference
> between a tool built for humans and one built for scripts.
