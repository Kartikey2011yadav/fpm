# cli

Command-line interface definitions and implementations using
[Cobra](https://github.com/spf13/cobra).

## Structure

Each command has two files:

- `<command>.go` — command definition (flags, args, help text)
- `<command>_impl.go` — implementation logic

## Commands

| File          | Command        | Description                |
| ------------- | -------------- | -------------------------- |
| `install.go`  | `fpm install`  | Install packages           |
| `remove.go`   | `fpm remove`   | Remove packages            |
| `sync.go`     | `fpm sync`     | Sync from lockfile         |
| `lock.go`     | `fpm lock`     | Generate lockfile          |
| `run.go`      | `fpm run`      | Execute in environment     |
| `init.go`     | `fpm init`     | Create project             |
| `venv.go`     | `fpm venv`     | Create virtual environment |
| `python.go`   | `fpm python`   | Python version management  |
| `cache.go`    | `fpm cache`    | Cache management + GC      |
| `snapshot.go` | `fpm snapshot` | Environment snapshots      |
| `tree.go`     | `fpm tree`     | Dependency tree            |
| `build.go`    | `fpm build`    | Build wheel/sdist          |
| `publish.go`  | `fpm publish`  | Upload to PyPI             |
| `pip.go`      | `fpm pip`      | pip compatibility layer    |
| `tool.go`     | `fpm tool`     | Tool management            |

## Global Flags

Defined in `root.go`: `--verbose`, `--quiet`, `--color`, `--no-progress`,
`--json`, `--global`.
