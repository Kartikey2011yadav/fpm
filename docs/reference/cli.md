# CLI Reference

## Global Flags

All commands support these flags:

| Flag                           | Short | Description                                |
| ------------------------------ | ----- | ------------------------------------------ |
| `--version`                    | `-v`  | Print version and exit                     |
| `--system`                     | `-s`  | Install into system Python instead of venv |
| `--verbose`                    |       | Enable verbose output                      |
| `--quiet`                      | `-q`  | Suppress output except errors              |
| `--json`                       |       | Output in JSON format                      |
| `--no-progress`                |       | Disable progress bars                      |
| `--color <mode>`               |       | Color output: auto, always, never          |
| `--allow-insecure-host <host>` |       | Skip TLS verification for specific hosts   |
| `--log-level <level>`          |       | Log level: debug, info, warn, error, off   |

## Package Commands

### `fpm install <packages...>`

Install packages. Aliases: `fpm add`.

- `--system` / `-s` — install into system Python (required when no venv)
- Without a venv and without `--system`, errors (like uv)

### `fpm remove <packages...>`

Remove packages from the environment. Aliases: `fpm uninstall`, `fpm rm`.

- `-f`, `--force` — remove even if installed by another manager (pip, uv, conda)
- `-p`, `--purge` — also remove unused dependencies
- Combine flags: `fpm remove -sfp numpy` (system + force + purge)

### `fpm autoremove`

Find and remove orphaned packages not required by anything else.

### `fpm list`

List installed packages with manager attribution. Alias: `fpm ls`.

- `--all` — include system packages
- `--manager <name>` — filter by manager (fpm, pip, uv, conda, poetry, pdm,
  system)
- `--json` — output as JSON array

### `fpm sync`

Sync environment to match lockfile exactly.

### `fpm lock`

Generate or update fpm.lock from pyproject.toml.

### `fpm tree`

Display dependency tree.

- `--depth <n>` — limit depth
- `--invert` — show reverse dependencies

### `fpm audit`

Scan installed packages for known vulnerabilities (OSV database).

## Project Commands

### `fpm init [path]`

Create a new Python project with pyproject.toml, .python-version, and .venv.

### `fpm run <command> [args...]`

Run command in the managed environment. Supports PEP 723 scripts.

### `fpm build [path]`

Build wheel and/or source distribution.

- `--wheel` — only wheel
- `--sdist` — only source distribution
- `-o <dir>` — output directory (default: dist/)

### `fpm publish [files...]`

Upload distributions to PyPI.

- `--repository <name>` — target (pypi, testpypi, or URL)
- `--token <token>` — authentication token

## Environment Commands

### `fpm venv [path]`

Create a virtual environment.

- `--python <version>` — use specific Python (downloads if needed)
- `--system-packages` — allow system site-packages
- `--prompt <string>` — custom activation prompt

### `fpm python list`

Show installed Python versions (managed and system).

### `fpm python install <versions...>`

Download and install Python versions.

### `fpm python use <version>`

Switch active Python. Local by default, `--system` for system-wide.

### `fpm python pin <version>`

Pin Python version for current project (.python-version).

### `fpm python uninstall <versions...>`

Remove managed Python installations.

### `fpm snapshot create [message]`

Capture current environment state. Alias: `fpm snapshot save`.

### `fpm snapshot list`

Show snapshot history. Alias: `fpm snapshot log`.

### `fpm snapshot restore <id>`

Restore environment to a previous snapshot.

### `fpm snapshot diff <id1> [id2]`

Compare two snapshots or snapshot vs current.

### `fpm snapshot delete <id>`

Remove a snapshot.

## Advanced Commands

### `fpm tool install <package>`

Install a CLI tool in an isolated environment.

### `fpm tool run <tool> [args...]`

Run a tool (installed or ephemeral).

### `fpm tool list`

List installed tools.

### `fpm tool uninstall <tool>`

Remove an installed tool.

### `fpm cache size`

Show cache disk usage breakdown.

### `fpm cache gc`

Remove unreferenced packages.

- `--older-than <duration>` — only GC entries older than duration (e.g., `30d`)

### `fpm cache list-unused`

Show cached packages not referenced by any environment.

### `fpm cache prune`

Remove corrupted cache entries.

### `fpm cache clean`

Remove all cached data.

### `fpm pip list`

List packages with manager attribution (same as `fpm list`).

- `--all` — include system packages
- `--manager <name>` — filter by manager

### `fpm pip freeze`

Output installed packages in requirements.txt format.

### `fpm pip show <packages...>`

Show detailed package information.

### `fpm self update`

Update fpm to the latest version.

### `fpm version`

Show fpm version. Also available as `fpm --version` or `fpm -v`.

### `fpm repair`

Diagnose and fix common installation issues. Checks:

- Missing directories (auto-creates)
- Cache integrity
- Python availability
- Bin directory in PATH
- Broken symlinks (auto-removes)
- Configuration file validity

### `fpm config show`

Display all current settings: directories, concurrency, link mode, logging,
indexes, and which config files are active.

### `fpm config set <key> <value>`

Set a user-level configuration value. Saved to `~/.config/fpm/config.toml`.

```bash
fpm config set log.level debug
fpm config set tool.concurrency 100
fpm config set cache.dir /tmp/fpm-cache
```

### `fpm config init`

Create a default user configuration file with documented options. Will not
overwrite an existing file.
