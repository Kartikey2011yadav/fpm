# Working with Other Package Managers

fpm is designed to coexist with pip, uv, conda, poetry, and pdm without
conflicts.

## How Coexistence Works

fpm uses **path precedence** — packages installed by fpm sit first in Python's
search path, shadowing (but not removing) packages from other managers.

```
sys.path priority:
1. .venv/lib/python3.11/site-packages  ← fpm manages this
2. /usr/lib/python3.11/site-packages   ← pip/system
3. /opt/conda/lib/python3.11/site-packages  ← conda
```

## Cross-Manager Detection

When you install a package, fpm checks all other managers first:

### Same version exists elsewhere

```bash
$ fpm install numpy
  numpy 1.24.0 is already installed via pip — skipping download
```

No duplicate download. fpm knows it's available.

### Different version exists

```bash
$ fpm install numpy==2.0.0
  numpy 1.24.0 is installed via pip, but you're requesting 2.0.0.
  After installation, fpm's version will take priority based on path order.

  [1] Skip installation (keep pip's 1.24.0)
  [2] Install anyway (fpm's 2.0.0 will shadow pip's)
  [3] Abort

  Choice [1/2/3]:
```

### Non-interactive Mode (CI/CD)

In `fpm.toml`:

```toml
[tool.fpm]
cross-manager-policy = "install"  # always install (skip prompts)
# or "skip" to always skip, or "ask" (default)
```

Or via environment:

```bash
export FPM_CROSS_MANAGER_POLICY=install
```

## Viewing All Packages

```bash
# Show everything from all managers
fpm pip list --all

# Filter by manager
fpm pip list --manager pip
fpm pip list --manager conda
fpm pip list --manager uv
```

## Supported Managers

| Manager | Detection Method                         |
| ------- | ---------------------------------------- |
| pip     | `INSTALLER` file = "pip"                 |
| uv      | `INSTALLER` file = "uv"                  |
| conda   | `INSTALLER` file = "conda" or conda path |
| poetry  | `INSTALLER` file = "poetry"              |
| pdm     | `INSTALLER` file = "pdm"                 |
| system  | path heuristics (/usr/lib/...)           |
| fpm     | `INSTALLER` file = "fpm"                 |

## Using fpm Alongside pip

This is perfectly fine:

```bash
pip install some-package    # pip manages it
fpm install other-package   # fpm manages it
fpm pip list --all          # shows both
```

## Using fpm Alongside uv

fpm detects uv-installed packages:

```bash
uv pip install black        # uv manages it
fpm install ruff            # fpm manages it
fpm pip list --all          # shows both with correct attribution
```
