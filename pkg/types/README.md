# types

Shared types used across the fpm codebase.

## Key Types

- `PackageName` — normalized Python package name (handles `-`, `_`, `.`
  equivalence)
- `ExtraName` — normalized extra/optional dependency name
- `HashDigest` — content hash (algorithm + hex value)

## Package Name Normalization

Per PEP 503, all of these refer to the same package:

- `My-Package`, `my_package`, `my.package` → normalized to `my-package`

## Files

- `package.go` — `PackageName`, `ExtraName`, `HashDigest`
