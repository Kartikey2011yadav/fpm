# pep440

Implementation of
[PEP 440 — Version Identification and Dependency Specification](https://peps.python.org/pep-0440/).

## Key Types

- `Version` — parsed version (epoch, release, pre, post, dev, local)
- `Specifier` — single version constraint (e.g., `>=1.0`)
- `VersionSpecifiers` — multiple constraints combined (e.g., `>=1.0, <2.0`)

## Supported Operators

`==`, `!=`, `<`, `<=`, `>`, `>=`, `~=` (compatible), `===` (arbitrary equality),
wildcards (`==1.0.*`)

## Usage

```go
v, _ := pep440.Parse("1.24.0")
specs, _ := pep440.ParseSpecifiers(">=1.0, <2.0")
specs.Contains(v) // true
```

## Files

- `version.go` — `Version` struct and methods
- `parse.go` — version string parser
- `compare.go` — version comparison (PEP 440 ordering)
- `specifier.go` — version specifier parsing and evaluation
