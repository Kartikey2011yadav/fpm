# Contributing to fpm

Thanks for your interest in contributing! fpm is an open-source Python package
manager that aims to be the best tool for managing Python environments. Every
contribution matters — from fixing typos to implementing features.

## Quick Start

```bash
# Fork and clone
git clone https://github.com/<your-username>/fpm.git
cd fpm

# Setup (installs deps, builds, configures git hooks)
./scripts/dev-setup.sh

# Or manually:
make build
make setup-hooks
```

**Requirements:** [Go 1.25+](https://go.dev/dl/) and optionally
[golangci-lint](https://golangci-lint.run/welcome/install/) for linting.

## Ways to Contribute

### For First-Time Contributors

Look for issues tagged [`good first issue`](https://github.com/Kartikey2011yadav/fpm/labels/good%20first%20issue):

- Add missing tests for existing functionality
- Improve error messages and hints
- Fix typos in docs
- Add shell completion improvements

### For Experienced Go Developers

- Implement features from the [roadmap](docs/roadmap-git-features.md)
- Fix bugs from the issue tracker
- Improve performance (parallel operations, caching)
- Add new platform support

### Non-Code Contributions

- Report bugs with reproducible steps
- Suggest features in [Discussions](https://github.com/Kartikey2011yadav/fpm/discussions)
- Write tutorials or blog posts
- Test on different platforms (Windows, Linux ARM, etc.)
- Improve documentation

## Development Workflow

### 1. Pick an Issue

- Comment on the issue saying you're working on it
- If no issue exists, open one first to discuss the approach

### 2. Create a Branch

```bash
git checkout -b feature/your-feature-name
```

### 3. Write Code

- Follow existing patterns in the codebase
- Add tests for new functionality
- Keep commits focused and well-described

### 4. Run Checks

```bash
make pre-commit          # vet + lint + short tests (same as git hook)
make test                # full test suite
go test ./internal/depgraph/ -race -v  # race detector for concurrency code
```

### 5. Submit PR

- Fill out the PR template
- Link the related issue
- Keep PRs small and focused (one feature/fix per PR)

## Pre-Commit Hooks

The project has git hooks (`.githooks/`) that run before each commit:

1. `go vet ./...` — catch common mistakes
2. `golangci-lint run` — style and bug checks
3. `go test ./... -short` — fast tests

These are set up automatically by `make setup-hooks`. If a hook fails, fix the
issue before committing.

## Testing

```bash
make test                    # full test suite
make test-short              # quick tests only
go test ./internal/pep440/ -v  # specific package
go test ./internal/depgraph/ -race -v  # with race detector
./scripts/test-depgraph-e2e.sh --insecure  # E2E integration (needs network)
```

**Rule:** All new functionality must have tests. Table-driven tests preferred.

## Code Style

- `gofmt` (enforced by linter)
- `go vet` (enforced by pre-commit hook)
- No comments unless the "why" is non-obvious
- Early returns over deep nesting
- Errors returned, never panicked
- `filepath.Join` for all path construction
- No unnecessary abstractions

## Project Structure

```
cmd/fpm/          Entry point
internal/
  cli/            Commands (name.go = definition, name_impl.go = logic)
  cache/          CAS storage, reference tracking, GC
  resolver/       PubGrub dependency resolver
  depgraph/       Dependency graph (requested vs transitive)
  snapshot/       Environment snapshots (capture/restore)
  installer/      Wheel installation (reflink/hardlink/copy)
  client/         PyPI HTTP client
  journal/        Operation history (append-only log)
  fs/             File locking, linking, permissions
  env/            Package scanning, cross-manager detection
  config/         Hierarchical configuration
  pep440/         Version parsing (PEP 440)
  pep508/         Dependency specifiers (PEP 508)
  python/         Python version management
  ...
pkg/
  types/          Public types (PackageName, HashDigest)
  errors/         Structured errors with hints
docs/             User documentation
scripts/          Dev scripts, test scenarios
```

## Architecture Decisions

- **CAS (Content-Addressable Storage):** Every package stored once by SHA256,
  linked into environments via reflink/hardlink. Enables zero-duplication.
- **Dependency Graph:** Tracks requested vs transitive packages. Enables
  autoremove, blame, and orphan detection.
- **File Locking:** All shared state (graph, refs, cache) uses advisory file
  locks for multi-user safety.
- **Git-like Workflow:** Operations recorded in journal, enabling revert/blame.
  Snapshots enable branch/bisect/stash.

## Release Process

Releases are automated via GitHub Actions. Only maintainers create releases:

```bash
git tag v0.X.0
git push origin v0.X.0
```

This builds binaries, Docker images, publishes to PyPI, and updates Homebrew.

## Getting Help

- Open a [Discussion](https://github.com/Kartikey2011yadav/fpm/discussions) for questions
- Read `docs/` for architecture details
- Check `docs/roadmap-git-features.md` for planned features
- Run `fpm docs <topic>` for inline help

## Code of Conduct

Be respectful, constructive, and inclusive. We're building something together.
