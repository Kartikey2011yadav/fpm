- Run `make test` to run the test suite
- Run `make build` to build the binary
- Run `make lint` to check for issues
- ALWAYS add tests for new functionality
- PREFER table-driven tests following Go conventions
- AVOID unnecessary abstractions — keep it simple
- PREFER `internal/` packages for non-public code
- NEVER use `panic()` in library code, return errors
- ALWAYS handle errors explicitly
- PREFER early returns over deep nesting
- ALWAYS use `filepath.Join` for path construction (cross-platform)
- PREFER goroutines + errgroup for parallel operations
- RUN `go vet ./...` before committing
- ALWAYS keep docs in sync: when adding/changing features, update the relevant
  README.md in the module directory, root README.md features list, and any
  affected per-module docs
- EVERY internal/ and pkg/ directory MUST have a README.md explaining its
  purpose, key types, and usage
- When a new command is added, update the root README.md CLI reference table
