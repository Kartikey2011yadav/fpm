# wheel

Wheel filename parsing and metadata extraction per
[PEP 427](https://peps.python.org/pep-0427/).

## Key Types

- `WheelFilename` — parsed wheel name (package, version, build,
  python/abi/platform tags)
- `Metadata` — extracted METADATA fields (name, version, requires-dist,
  requires-python)

## Features

- Parse wheel filenames into structured data
- Extract METADATA directly from zip without full extraction
- Determine pure vs platform-specific wheels
- Generate platform compatibility tags

## Files

- `wheel.go` — filename parsing, metadata extraction from zip
