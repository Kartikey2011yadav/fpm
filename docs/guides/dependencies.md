# Managing Dependencies

## Installing Packages

```bash
# Latest version
fpm install requests

# Specific version
fpm install numpy==1.24.0

# Version range
fpm install "flask>=2.0,<3.0"

# Multiple packages
fpm install requests numpy pandas scipy

# With extras
fpm install "requests[security]"

# From requirements.txt
fpm install -r requirements.txt

# Editable/development install
fpm install -e ./my-local-package

# From git
fpm install "git+https://github.com/user/repo.git@v1.0"
```

## Removing Packages

```bash
fpm remove requests
```

This removes the package files, updates pyproject.toml, and adjusts the
lockfile.

## Updating Packages

```bash
# Update specific package
fpm install numpy --upgrade

# Re-resolve everything
fpm lock
fpm sync
```

## Dependency Groups

In `fpm.toml`:

```toml
[project]
dependencies = ["requests>=2.28"]

[tool.fpm.dependency-groups]
test = ["pytest>=7.0", "coverage"]
docs = ["sphinx", "furo"]
dev = ["ruff", "mypy"]
```

Install a group:

```bash
fpm install --group test
```

## Immutable Packages

Pin packages that must never change (e.g., due to ABI compatibility):

```toml
[immutable]
packages = [
    { name = "numpy", version = "1.24.0" },
    { name = "torch", version = "2.1.0" },
]
```

If any dependency tries to pull a different version, fpm will backtrack or error
with a clear message.

## Global vs Local

```bash
# Install into project venv (default, safe)
fpm install requests

# Install system-wide
fpm install requests --global
```

## Viewing Dependencies

```bash
# Dependency tree
fpm tree

# Flat list
fpm pip list

# Freeze format
fpm pip freeze

# Package details
fpm pip show requests
```
