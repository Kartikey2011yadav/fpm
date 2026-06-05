# python

Python interpreter discovery, version management, and managed installations.

## Key Types

- `Interpreter` — discovered Python (path, version, prefix, site-packages,
  implementation)
- `Finder` — discovers Python installations from PATH, pyenv, conda, managed
- `ManagedVersion` — fpm-installed Python version

## Features

- PATH scanning for system Pythons
- Probes interpreters for metadata (runs `python -c` to extract sys info)
- Downloads pre-built Python from python-build-standalone
- Manages multiple versions simultaneously
- Switches via symlink manipulation (local .python-version or global)
- `.python-version` file support

## Files

- `interpreter.go` — `Interpreter` struct, install scheme, tags
- `discovery.go` — `Finder`, PATH scanning, probing
- `managed.go` — download, install, switch, uninstall Python versions
