# Virtual Environments

## What They Are

A virtual environment (venv) is an isolated Python installation. Packages
installed in one venv don't affect other projects or the system Python.

fpm creates and manages venvs automatically when you run `fpm init`.

## Project Layout

```
myproject/
├── pyproject.toml        # Project metadata + dependencies
├── fpm.toml              # fpm-specific configuration
├── fpm.lock              # Locked dependency versions
├── .python-version       # Pinned Python version
├── .venv/                # Virtual environment
│   ├── pyvenv.cfg        # Venv metadata
│   ├── bin/              # python, pip, activate scripts
│   └── lib/python3.12/site-packages/
│       ├── requests/     # ← linked from CAS
│       └── requests-2.31.0.dist-info/
│           └── INSTALLER # "fpm\n"
└── src/                  # Your code
```

## fpm's Venv Model

### Creation

```bash
fpm init myproject       # creates pyproject.toml + .venv
fpm venv                 # creates .venv in current directory
fpm venv --python 3.11   # uses specific Python version
```

### Detection

fpm finds the active venv by walking up the directory tree looking for
`pyvenv.cfg`. It also checks for `.venv/` in the current directory.

**`VIRTUAL_ENV` is intentionally ignored.** Unlike pip, fpm uses directory-based
detection only — just `cd` into your project and all commands target the venv.
Leaving the directory (`cd ..`) deactivates the project environment. This
matches how uv's project commands (`uv add`, `uv sync`) work.

### System vs Venv Installs

| Command                         | Target            | Behavior                |
| ------------------------------- | ----------------- | ----------------------- |
| `fpm install requests`          | .venv (if exists) | Errors if no venv found |
| `fpm install -s requests`       | System Python     | Installs globally       |
| `fpm install --system requests` | System Python     | Same as -s              |

This matches uv's model: explicit `--system` required for global installs.
Prevents accidental system Python pollution.

## Why Venvs Matter

### Isolation

Project A needs `requests==2.28`. Project B needs `requests==2.31`. Without
venvs, they'd conflict. With venvs, each project has its own copy.

### Reproducibility

`fpm.lock` + venv = exact same packages on every machine.

```bash
git clone project
cd project
fpm venv && fpm sync     # identical to the original developer's setup
```

### No sudo

System-wide installs may need root. Venvs are per-user, per-project — no
elevated permissions needed.

## How fpm Uses Venvs

- **install**: packages go into `.venv/lib/.../site-packages/` (via CAS links)
- **run**: executes commands with the venv's Python and PATH
- **lock/sync**: lockfile is scoped to the project's declared deps
- **snapshot**: captures the venv's complete state
- **tree**: shows the dependency tree from the lockfile
- **remove**: updates pyproject.toml (run `fpm sync` to apply)

## No Activation Needed

fpm does **not** require `source .venv/bin/activate`. Simply being in the
project directory is enough — all fpm commands auto-detect the `.venv`.

```bash
cd myproject     # ← this is all you need
fpm install ...  # targets .venv automatically
fpm list         # shows venv packages
cd ..            # ← leaving = deactivated
fpm list         # errors (no venv in scope)
```

If you still want shell-level access to the venv's Python (for `python`,
`pytest`, etc. without `fpm run`), activation still works:

```bash
source .venv/bin/activate    # bash/zsh
source .venv/bin/activate.fish  # fish
.venv\Scripts\activate.bat   # Windows cmd
```

Note: even after activation, fpm ignores `VIRTUAL_ENV` — it always uses
directory detection. This prevents accidentally installing to a project's venv
from outside its directory.

## Developer Reference

Key code:

- `internal/venv/create.go` — `Create()`, `Detect()`, `Venv` struct
- `internal/cli/run_impl.go` — `buildVenvEnv()` for PATH setup
- `internal/cli/install_impl.go` — venv detection + `--system` gating
- `internal/python/discovery.go` — `Probe()` for interpreter detection
