# Testing Guide

fpm has two levels of testing: Go unit tests and Docker integration tests.

## Unit Tests (Go)

Run the full Go test suite:

```bash
make test          # All packages, verbose
go test ./...      # Quick run
go test ./internal/cli/ -v       # Single package
go test ./internal/tls/ -run TestNormalizeHost  # Single test
```

### Test Coverage

| Package              | Tests | What's covered                               |
| -------------------- | ----- | -------------------------------------------- |
| `internal/auth`      | 4     | Credential sources, netrc parsing            |
| `internal/cache`     | 4     | CAS store/retrieve, reference tracking       |
| `internal/cli`       | 9     | Commands exist, flags, help groups, aliases  |
| `internal/config`    | 5     | Config loading, env overrides, platform dirs |
| `internal/env`       | 6     | Package scanning, cross-manager detection    |
| `internal/fs`        | 5     | Linking strategies, fallbacks                |
| `internal/git`       | 2     | Git URL parsing, URL detection               |
| `internal/installer` | 3     | Install from CAS, uninstall, entry points    |
| `internal/lock`      | 5     | Generate, read/write, diff, export           |
| `internal/pep440`    | 7     | Version parsing, specifiers, comparison      |
| `internal/pep508`    | 6     | Requirement parsing, markers, extras         |
| `internal/platform`  | 5     | Platform tags, wheel compatibility           |
| `internal/script`    | 3     | PEP 723 inline metadata                      |
| `internal/snapshot`  | 4     | Capture, list, diff, delete                  |
| `internal/tls`       | 9     | Cert loading, host normalize, transport      |
| `internal/wheel`     | 2     | Filename parsing                             |
| `internal/workspace` | 4     | pyproject.toml read/write, discovery         |
| `pkg/errors`         | 5     | Error types, wrapping, hints                 |
| `pkg/types`          | 4     | Package name normalization                   |

**Total: 19 test suites, 90+ test cases**

### Writing Tests

Follow Go conventions (per CLAUDE.md):

- Table-driven tests for multiple inputs
- Use `t.TempDir()` for filesystem isolation
- Co-locate test files with source (`*_test.go`)
- Test both happy path and error cases

```go
func TestSomething(t *testing.T) {
    tests := []struct {
        name string
        input string
        want  string
    }{
        {"basic", "input", "expected"},
        {"edge case", "", ""},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := DoSomething(tt.input)
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

## Integration Tests (Docker)

The `scripts/test-features.sh` script tests all features end-to-end in a real
Docker container with Python + uv + fpm.

### Setup

```bash
# Create container
docker run -d --name fpm-test python:3.12-slim sleep infinity
docker exec fpm-test pip install uv \
    --trusted-host pypi.org --trusted-host files.pythonhosted.org

# Build and deploy fpm
GOOS=linux GOARCH=amd64 go build -o bin/fpm-linux ./cmd/fpm
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm

# Copy and run tests
docker cp scripts/test-features.sh fpm-test:/tmp/
docker exec fpm-test bash /tmp/test-features.sh
```

### Running Tests

```bash
# All tests
docker exec fpm-test bash /tmp/test-features.sh

# List available groups
docker exec fpm-test bash /tmp/test-features.sh --list

# Run specific groups
docker exec fpm-test bash /tmp/test-features.sh cli install errors

# With logging
docker exec fpm-test bash /tmp/test-features.sh --log /tmp/fpm-test.log

# View log
docker exec fpm-test cat /tmp/fpm-test.log
```

### Test Groups (17 total, 52 tests)

| Group          | Description                             |
| -------------- | --------------------------------------- |
| `cli`          | CLI basics — version, help, flags       |
| `install`      | System install with --system flag       |
| `list`         | List variants (fpm-only, all, filtered) |
| `pip`          | Pip compatibility layer                 |
| `errors`       | Error messages, hints, did-you-mean     |
| `audit`        | Vulnerability scanning + caching        |
| `config`       | Configuration and repair commands       |
| `project`      | Full project workflow (init → run)      |
| `tree`         | Dependency tree display                 |
| `snapshot`     | Environment snapshots                   |
| `remove`       | Package removal + aliases               |
| `python`       | Python version discovery                |
| `venv`         | Virtual environment creation            |
| `cache`        | Cache management                        |
| `crossmanager` | Cross-manager conflict detection        |
| `immutable`    | Immutable package pins                  |
| `version`      | Version flag variations                 |

### Rebuilding After Changes

```bash
# Quick rebuild + test cycle
GOOS=linux GOARCH=amd64 go build -o bin/fpm-linux ./cmd/fpm \
    && docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm \
    && docker exec fpm-test bash /tmp/test-features.sh
```

### VPN/Proxy Environments

The test script automatically sets:

```bash
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev
```

If you need additional hosts, modify the script or set before running:

```bash
docker exec -e FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,custom.host \
    fpm-test bash /tmp/test-features.sh
```

## CI/CD

The project uses GitHub Actions. Key checks:

- `go vet ./...` — static analysis
- `go build ./...` — compilation
- `go test ./...` — unit tests

Pre-commit hooks run automatically:

- Prettier for markdown formatting
- `go vet` and `go build` validation

## Adding New Tests

### Unit Test

Add to the relevant `*_test.go` file in the package directory.

### Integration Test

Add a new section to `scripts/test-features.sh`:

```bash
if should_run "myfeature"; then
    section "My Feature"
    # Test commands here
    fpm mycommand >/dev/null 2>&1 && pass "mycommand works" || fail "mycommand"
fi
```

Then add `myfeature` to the `ALL_GROUPS` variable at the top of the script and
add it to `show_list()`.
