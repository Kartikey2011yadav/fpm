# env

Environment scanning, cross-manager package detection, and path precedence
management.

## Key Types

- `InstalledPackage` — package found in site-packages (name, version, manager,
  location)
- `PackageManager` — detected source (fpm, pip, uv, conda, poetry, pdm, system)
- `ScanResult` — complete inventory of all installed packages
- `CrossManagerChecker` — conflict detection and user prompting
- `PthConfig` — .pth file generation for path priority

## Supported Managers

fpm detects packages installed by: **pip, uv, conda, poetry, pdm, system, fpm**

Detection uses the `INSTALLER` file in `.dist-info` directories and path
heuristics.

## Cross-Manager Behavior

- Same version exists → inform user, skip download
- Different version exists → prompt user (skip/install/abort)
- Non-interactive mode → configurable policy via `fpm.toml`

## Files

- `scanner.go` — scan site-packages, parse .dist-info/METADATA
- `crossmanager.go` — conflict detection, user prompts, policies
- `pth.go` — .pth file generation for path precedence layering
