# build

PEP 517 build frontend for creating wheels and source distributions.

## Key Types

- `BuildFrontend` — build orchestrator (Python path, source dir, output dir)
- `BuildResult` — built artifacts (wheel path, sdist path)

## Features

- Invokes PEP 517 build backends via Python subprocess
- Supports setuptools, flit, hatch, maturin, and any PEP 517-compliant backend
- Falls back to `setup.py` for legacy projects

## Files

- `build.go` — `BuildWheel()`, `BuildSdist()`
