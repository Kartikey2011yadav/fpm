# pep508

Implementation of
[PEP 508 — Dependency specification for Python Software Packages](https://peps.python.org/pep-0508/).

## Key Types

- `Requirement` — parsed dependency (name, extras, version specifiers, URL,
  markers)
- `MarkerTree` — environment marker expression tree
- `MarkerEnvironment` — runtime environment values for marker evaluation

## Supported Syntax

```
requests[security] >= 2.28.0; python_version >= "3.8" and sys_platform == "linux"
package @ https://example.com/package-1.0.tar.gz
```

## Usage

```go
req, _ := pep508.ParseRequirement(`requests[security]>=2.28; python_version>="3.8"`)
req.Name.Normalized()        // "requests"
req.Extras                   // ["security"]
req.EvaluateMarkers(env)     // true/false based on environment
```

## Files

- `requirement.go` — `Requirement` struct
- `parse.go` — recursive descent parser
- `marker.go` — marker expression types and evaluation
