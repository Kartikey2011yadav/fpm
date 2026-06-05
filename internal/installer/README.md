# installer

Package installation and uninstallation engine.

## Key Types

- `InstallPlan` — what to install (package, CAS key, target dir, link mode)
- `InstallResult` — outcome (file count, dist-info path)
- `UninstallResult` — outcome (removed files/dirs)
- `ConsoleScript` — entry point definition (name, module, function)

## Features

- Install wheels from CAS to site-packages via linking
- Write INSTALLER marker file (identifies fpm-managed packages)
- Uninstall by reading RECORD and removing all listed files
- Generate console_script entry point wrappers (Unix + Windows)
- Editable installs via .egg-link and .pth files (PEP 660)

## Files

- `install.go` — wheel installation from CAS
- `uninstall.go` — package removal via RECORD
- `scripts.go` — console_script wrapper generation
- `editable.go` — editable/development installs
