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
- Directory-based venv detection (`Detect()` walks up directory tree)
- `VIRTUAL_ENV` env var is intentionally ignored — like uv project commands, fpm
  only detects venvs by being in (or under) the project directory
- Supports any Python version (can use fpm-managed Python different from system)

## Files

- `create.go` — venv creation, Python linking, detection
- `activate.go` — activation script generation for all shells
