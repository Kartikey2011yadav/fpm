# CLI Tools

fpm can install and manage Python CLI tools in isolated environments (like
pipx).

## Install a Tool

```bash
fpm tool install black
fpm tool install ruff
fpm tool install httpie
```

Each tool gets its own virtual environment. Dependencies are isolated — tools
never conflict with your project.

## Run a Tool

```bash
# Run installed tool
fpm tool run black --check .

# Run tool without installing (ephemeral)
fpm tool run pyright .
```

Ephemeral runs create a temporary environment, execute, and clean up.

## List Installed Tools

```bash
fpm tool list
```

Output:

```
Installed tools:
  black 24.1.0 (Python 3.12.1)
    - black
    - blackd
  ruff 0.2.0 (Python 3.12.1)
    - ruff
```

## Uninstall

```bash
fpm tool uninstall black
```

## How It Works

Tools are stored in `~/.local/share/fpm/tools/<name>/`:

```
~/.local/share/fpm/tools/
├── black/
│   ├── .venv/           # isolated environment
│   └── receipt.json     # metadata (version, entrypoints)
├── ruff/
│   └── ...
```

Entrypoints are symlinked to `~/.local/share/fpm/bin/` which should be on your
PATH.
