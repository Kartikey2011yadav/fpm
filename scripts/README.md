# scripts/

Development and testing scripts for fpm.

## Scripts

| Script                    | Purpose                            | Usage                                             |
| ------------------------- | ---------------------------------- | ------------------------------------------------- |
| `test-features.sh`        | Feature integration tests (Docker) | `./test-features.sh [groups...] [--log] [--list]` |
| `check-all.sh`            | Run all CI checks locally          | `./check-all.sh`                                  |
| `dev-setup.sh`            | Set up development environment     | `./dev-setup.sh`                                  |
| `benchmark.sh`            | Benchmark fpm operations           | `./benchmark.sh [binary]`                         |
| `generate-completions.sh` | Generate shell completions         | `./generate-completions.sh [binary]`              |
| `release.sh`              | Create a new release               | `./release.sh <version>`                          |

## test-features.sh

The primary integration test script. Runs inside a Docker container to test all
fpm features end-to-end against a real Python environment.

### Quick Start

```bash
# Setup container (pass env var via -e, not .bashrc)
docker run -d --name fpm-test \
  -e FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev \
  python:3.12-slim sleep infinity
docker exec fpm-test pip install uv --trusted-host pypi.org --trusted-host files.pythonhosted.org
GOOS=linux GOARCH=amd64 go build -o bin/fpm-linux ./cmd/fpm
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm

# Run all tests
docker cp scripts/test-features.sh fpm-test:/tmp/
docker exec fpm-test bash /tmp/test-features.sh
```

> **Note:** `docker exec` doesn't source `.bashrc`. Env vars must be set via
> `docker run -e` or passed inline: `docker exec -e VAR=val fpm-test ...`

### Options

```
./test-features.sh                    Run all 52 tests (17 groups)
./test-features.sh --list             List available test groups
./test-features.sh --help             Show usage
./test-features.sh cli install        Run specific groups only
./test-features.sh --log [file]       Save log (default: /tmp/fpm-test.log)
./test-features.sh --verbose          Show command output inline
./test-features.sh errors --log       Combine: specific group + logging
```

### Test Groups

| Group          | Tests | What it covers                            |
| -------------- | ----- | ----------------------------------------- |
| `cli`          | 7     | Version, help, flags visibility           |
| `install`      | 3     | System install, --system flag enforcement |
| `list`         | 4     | fpm list, -a, --manager filter            |
| `pip`          | 3     | pip list, freeze, show compatibility      |
| `errors`       | 3     | 404 messages, did-you-mean, hints         |
| `audit`        | 2     | Vulnerability scanning + cache            |
| `config`       | 4     | config show, settings, logging, repair    |
| `project`      | 8     | init, install, lock, run, pyproject.toml  |
| `tree`         | 2     | Dependency tree + lock command            |
| `snapshot`     | 4     | Create, list, diff snapshots              |
| `remove`       | 7     | Remove, purge, autoremove, --force, -s    |
| `python`       | 2     | Python version discovery                  |
| `venv`         | 3     | Virtual environment creation              |
| `cache`        | 1     | Cache size reporting                      |
| `crossmanager` | 2     | Detects pip packages, skip existing       |
| `immutable`    | 1     | Blocks pinned version changes             |
| `version`      | 2     | Version subcommand + -v flag              |

### Log Output

When `--log` is used, the log file contains:

- Timestamp of test run
- Working directory
- PASS/FAIL/SKIP status for each test
- Final result summary

### Environment Variables

The script automatically sets
`FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev` to handle
VPN/proxy environments.

## check-all.sh

Runs the same checks as CI: `go vet`, `go build`, `go test ./...`. Use before
pushing to catch issues early.

```bash
./scripts/check-all.sh
```

## dev-setup.sh

One-time setup for new developers. Installs dependencies, builds the binary, and
verifies the development environment is ready.

```bash
./scripts/dev-setup.sh
```

## benchmark.sh

Times common fpm operations for performance tracking.

```bash
./scripts/benchmark.sh ./bin/fpm
```

## generate-completions.sh

Generates shell completion scripts for bash, zsh, fish, and PowerShell.

```bash
./scripts/generate-completions.sh ./bin/fpm
# Outputs to completions/
```

## release.sh

Creates a tagged release with cross-compiled binaries.

```bash
./scripts/release.sh v0.2.0
```
