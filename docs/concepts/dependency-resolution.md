# Dependency Resolution

## Overview

When you run `fpm install requests`, fpm must figure out which exact version
of `requests` to install AND which versions of all its transitive dependencies
(urllib3, certifi, charset-normalizer, idna) are compatible with each other.

This is the dependency resolution problem.

## The PubGrub Algorithm

fpm uses **PubGrub**, the same algorithm used by Dart (pub) and Rust (cargo).
It's a SAT-solver approach that:

1. Starts with the user's requirements
2. Picks the highest compatible version of each package
3. Fetches that version's dependencies from PyPI
4. Checks for conflicts
5. If conflict found: backtracks intelligently (not brute-force)
6. Repeats until all packages are resolved or proven unsatisfiable

### Why PubGrub over pip's resolver?

| Aspect | pip (backtracking) | fpm (PubGrub) |
|--------|-------------------|---------------|
| Conflict detection | Trial and error | Learns from conflicts |
| Backtracking | Random | Targeted (knows which package caused it) |
| Error messages | "Could not find compatible version" | Explains exactly why |
| Speed | Can take minutes for complex trees | Usually seconds |

## Resolution Flow

```
User: fpm install "flask>=2.0"

1. Fetch flask metadata from PyPI (all versions + their deps)
2. Pick highest compatible: flask 3.1.3
3. flask 3.1.3 requires: werkzeug>=3.0, jinja2>=3.1, click>=8.0, ...
4. For each dep, repeat: fetch metadata, pick version
5. Check all constraints are satisfied simultaneously
6. Result: exact version for every package in the tree
```

## Immutable Package Pins

fpm supports **immutable pins** in `fpm.toml`:

```toml
[immutable]
packages = [
    { name = "numpy", version = "1.24.0" }
]
```

During resolution, if any package (or transitive dependency) would require
a different version of numpy, the resolver immediately fails with a clear
error rather than silently changing the pinned version.

Use case: production systems where a specific library version has been
validated and must never change, even if a new dependency wants a newer version.

## Lockfile

After resolution succeeds, fpm writes `fpm.lock` — a deterministic record
of every package, version, source URL, and content hash:

```toml
[[package]]
name = "requests"
version = "2.31.0"
source = { registry = "https://pypi.org/simple" }
hashes = ["sha256:abc123..."]

[[package.dependencies]]
name = "urllib3"
version = ">=2.0"
```

The lockfile ensures:
- `fpm sync` installs the exact same versions on any machine
- CI gets the same packages as development
- `fpm lock` only changes when pyproject.toml changes

## Cross-Manager Awareness

Before downloading, fpm checks if the resolved package is already installed
by another manager (pip, uv, conda):

- **Same version exists** → skip (no download needed)
- **Different version exists** → depends on `cross-manager-policy`:
  - `ask`: prompt user (skip / install / abort)
  - `install`: install fpm's version (shadows the other)
  - `skip`: keep the existing version

## Developer Reference

Key code:
- `internal/resolver/resolver.go` — PubGrub implementation, `Resolve()`
- `internal/lock/lockfile.go` — `Generate()`, `Read()`, `Write()`, `Diff()`
- `internal/client/client.go` — `FetchPackageVersions()` (PyPI metadata)
- `internal/pep440/` — Version parsing and specifier matching
- `internal/pep508/` — Dependency requirement parsing (name, extras, markers)
