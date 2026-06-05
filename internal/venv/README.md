# venv

Virtual environment creation with activation scripts for all shells.

## Key Types

- `Venv` — created virtual environment (path, bin dir, site-packages, Python
  path)
- `CreateOptions` — creation parameters (path, interpreter, system-packages,
  prompt)

## Features

- Standard venv structure (bin/, lib/, pyvenv.cfg)
- Activation scripts: bash, zsh, fish, csh, PowerShell, cmd.exe
- Symlinks Python binary (falls back to copy on Windows)
- Auto-detection of existing venvs (`Detect()` walks up directory tree)
- Supports any Python version (can use fpm-managed Python different from system)

## Files

- `create.go` — venv creation, Python linking, detection
- `activate.go` — activation script generation for all shells
