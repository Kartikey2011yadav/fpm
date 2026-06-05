# Python Version Management

fpm can install, manage, and switch between multiple Python versions without
affecting your system Python.

## Listing Available Pythons

```bash
fpm python list
```

Output:

```
Installed Python versions:

  fpm-managed:
    * 3.12.1  (~/.local/share/fpm/python/cpython-3.12.1)
      3.11.8  (~/.local/share/fpm/python/cpython-3.11.8)

  System:
      3.14.5  (/usr/bin/python3)
      3.11.2  (/usr/local/bin/python3.11)
```

## Installing Python Versions

```bash
# Install one version
fpm python install 3.11

# Install multiple
fpm python install 3.11 3.12 3.13
```

fpm downloads pre-built Python from python-build-standalone (fast, no
compilation).

## Switching Versions

### Per-Project (recommended)

```bash
fpm python use 3.11
```

This writes `.python-version` in the current directory. Only this project uses
3.11.

### System-Wide

```bash
fpm python use 3.12 --global
```

Updates symlinks in `~/.local/share/fpm/bin/`. Add this to your PATH to use
fpm-managed Python globally.

## Creating Venvs with Specific Python

```bash
# System has 3.14, but you need 3.11 for this project
fpm venv --python 3.11

# If 3.11 isn't installed, fpm auto-downloads it
```

The venv is completely isolated — your global Python is unaffected.

## How It Works

- Python versions are stored in `~/.local/share/fpm/python/cpython-X.Y.Z/`
- `fpm python use` creates symlinks in `~/.local/share/fpm/bin/`
- Per-project switching uses `.python-version` file (compatible with pyenv)
- Venvs symlink to the specific Python binary, not a global path

## Uninstalling

```bash
fpm python uninstall 3.11
```
