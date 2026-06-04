# Contributing to fpm

## Setup

[Go 1.22+](https://go.dev/dl/) is required to build fpm.

```bash
git clone https://github.com/kartikeyyadav/fpm.git
cd fpm
make build
```

## Testing

Run the full test suite:

```bash
make test
```

Run tests for a specific package:

```bash
go test ./internal/pep440/ -v
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` for additional checks
- Write table-driven tests
- Keep functions small and focused
- Prefer returning errors over panicking

## Pull Requests

- Include a clear description of the change
- Add tests for new functionality
- Ensure `make test` and `make lint` pass
- Keep PRs focused — one feature or fix per PR

## Architecture

See the plan file for detailed architecture documentation. Key packages:

- `internal/cli/` — Command definitions (Cobra)
- `internal/config/` — Configuration parsing
- `internal/pep440/` — Version parsing (PEP 440)
- `internal/pep508/` — Dependency specifiers (PEP 508)
- `internal/platform/` — Platform tags (PEP 425)
- `internal/resolver/` — PubGrub dependency resolver
- `internal/cache/` — Content-addressable cache + GC
- `internal/env/` — Environment scanning + cross-manager inspection
