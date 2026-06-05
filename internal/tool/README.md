# tool

Isolated tool management (similar to pipx).

## Key Types

- `Tool` — installed tool metadata (name, version, entrypoints)
- `Registry` — tool installation tracking and lifecycle

## Features

- Install tools in isolated virtual environments
- Symlink entrypoints to shared bin directory
- Run tools ephemerally (create temp env, run, cleanup)
- Track installed tools with receipt files
- Per-tool isolation prevents dependency conflicts

## Storage Layout

```
~/.local/share/fpm/tools/
├── black/
│   ├── .venv/           # isolated environment
│   └── receipt.json     # tool metadata
├── ruff/
│   └── ...
~/.local/share/fpm/bin/
├── black -> ../tools/black/.venv/bin/black
├── ruff -> ../tools/ruff/.venv/bin/ruff
```

## Files

- `tool.go` — install, run, list, uninstall, registry
