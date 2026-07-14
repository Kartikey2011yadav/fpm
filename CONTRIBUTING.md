# Contributing to fpm

## Setup

[Go 1.25+](https://go.dev/dl/) is required to build fpm.

```bash
git clone https://github.com/kartikeyyadav/fpm.git
cd fpm
make build
make setup-hooks   # configure pre-commit hooks (vet + lint + tests)
```

Or run the full dev setup script:

```bash
./scripts/dev-setup.sh
```

## Pre-Commit Hooks

The project uses git hooks (`.githooks/`) that run automatically before each
commit:

1. `go vet ./...`
2. `golangci-lint run` (skipped if not installed)
3. `go test ./... -short`

To run these checks manually:

```bash
make pre-commit
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

Run short tests only (used by pre-commit hook):

```bash
make test-short
```

Run with race detector (especially for depgraph/concurrency tests):

```bash
go test ./internal/depgraph/ -race -v
```

Run the E2E integration test script:

```bash
./scripts/test-depgraph-e2e.sh --insecure  # if behind corporate proxy
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Use `golangci-lint` for additional checks (config in `.golangci.yml`)
- Write table-driven tests
- Keep functions small and focused
- Prefer returning errors over panicking

## Pull Requests

- Include a clear description of the change
- Add tests for new functionality
- Ensure `make pre-commit` passes locally before pushing
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
