# Developer Reference: Docker Setup & Paths

Quick reference for setting up the test environment, paths, and configuration.

---

## Docker Container Setup

### Fresh container (clean slate)

```bash
# Remove old container if exists
docker stop fpm-test 2>/dev/null; docker rm fpm-test 2>/dev/null

# Create new container with TLS bypass for PyPI
docker run -d --name fpm-test \
  -e FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev \
  python:3.12-slim sleep infinity
```

### Install fpm binary

```bash
# Build for Linux (from macOS/host)
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/fpm-linux ./cmd/fpm

# Copy into container
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm

# Verify
docker exec fpm-test fpm --version
```

### Create test users

```bash
docker exec fpm-test bash -c '
  useradd -m -s /bin/bash alice
  useradd -m -s /bin/bash bob
  useradd -m -s /bin/bash charlie
'
```

### Configure pip trusted hosts (avoids TLS errors)

```bash
docker exec fpm-test bash -c '
  mkdir -p /root/.config/pip
  echo -e "[global]\ntrusted-host = pypi.org\n               files.pythonhosted.org" > /root/.config/pip/pip.conf
'
```

### Access the container

```bash
docker exec -it fpm-test bash              # root
docker exec -it -u alice fpm-test bash     # alice
docker exec -it -u bob fpm-test bash       # bob
docker exec -it -u charlie fpm-test bash   # charlie
```

---

## Paths Inside Container

### System paths

| Path                                       | Purpose              |
| ------------------------------------------ | -------------------- |
| `/usr/local/bin/fpm`                       | fpm binary           |
| `/usr/local/bin/python3.12`                | System Python        |
| `/usr/local/lib/python3.12/site-packages/` | System site-packages |

### Per-user paths (root)

| Path                                   | Purpose                                    |
| -------------------------------------- | ------------------------------------------ |
| `/root/.cache/fpm/`                    | Cache (CAS, wheels, HTTP, refs, snapshots) |
| `/root/.cache/fpm/cas/sha256/`         | Content-Addressable Storage                |
| `/root/.cache/fpm/wheels/`             | Downloaded .whl files                      |
| `/root/.cache/fpm/http/`               | HTTP response cache                        |
| `/root/.cache/fpm/refs/`               | Reference tracking (by-env, by-cas)        |
| `/root/.cache/fpm/snapshots/`          | Environment snapshots                      |
| `/root/.local/share/fpm/`              | Data (depgraph, python installs)           |
| `/root/.local/share/fpm/depgraph.json` | System dependency graph                    |
| `/root/.config/fpm/`                   | User configuration                         |
| `/root/.config/fpm/config.toml`        | User config file                           |

### Per-user paths (alice/bob/charlie)

Same structure under `/home/<user>/`:

- `/home/alice/.cache/fpm/`
- `/home/alice/.local/share/fpm/`
- `/home/alice/.config/fpm/`

### Per-project paths

| Path                                          | Purpose                                  |
| --------------------------------------------- | ---------------------------------------- |
| `project/.venv/`                              | Virtual environment                      |
| `project/.venv/pyvenv.cfg`                    | Venv marker (fpm detects this)           |
| `project/.venv/bin/python3`                   | Venv Python                              |
| `project/.venv/lib/python3.12/site-packages/` | Venv packages                            |
| `project/.fpm-depgraph.json`                  | Project dependency graph                 |
| `project/pyproject.toml`                      | PEP 621 project metadata                 |
| `project/fpm.toml`                            | fpm configuration (immutable pins, etc.) |
| `project/fpm.lock`                            | Lockfile (resolved versions + hashes)    |

### Multi-user mode paths

| Path                   | Purpose                              |
| ---------------------- | ------------------------------------ |
| `/var/cache/fpm/`      | Shared CAS (multi-user mode)         |
| `/etc/fpm/mode`        | System-wide mode file ("multi-user") |
| `/etc/fpm/config.toml` | System-wide config                   |

---

## Environment Variables

| Variable                  | Default                     | Purpose                                 |
| ------------------------- | --------------------------- | --------------------------------------- |
| `FPM_ALLOW_INSECURE_HOST` | (none)                      | Comma-separated hosts to skip TLS       |
| `FPM_CACHE_DIR`           | `~/.cache/fpm`              | Override cache directory                |
| `FPM_DATA_DIR`            | `~/.local/share/fpm`        | Override data directory                 |
| `FPM_CONFIG_DIR`          | `~/.config/fpm`             | Override config directory               |
| `FPM_MODE`                | (none)                      | Set to "multi-user" to force shared CAS |
| `FPM_SHARED_CACHE_DIR`    | `/var/cache/fpm`            | Custom shared cache path                |
| `FPM_PYTHON_INSTALL_DIR`  | `~/.local/share/fpm/python` | Managed Python installs                 |
| `FPM_TOOL_DIR`            | `~/.local/share/fpm/tools`  | Tool environments                       |
| `FPM_TOOL_BIN_DIR`        | `~/.local/bin`              | Tool binary symlinks                    |

---

## Running Test Suites

### Copy test scripts into container

```bash
docker cp scripts/test-features.sh fpm-test:/tmp/
docker cp scripts/test-scenarios.sh fpm-test:/tmp/
docker cp scripts/test-venv-isolation.sh fpm-test:/tmp/
docker cp scripts/test-snapshot-scenarios.sh fpm-test:/tmp/
docker cp scripts/test-snapshot-full.sh fpm-test:/tmp/
docker cp scripts/test-multiuser.sh fpm-test:/tmp/
```

### Run all tests

```bash
docker exec fpm-test bash /tmp/test-features.sh          # 76 tests
docker exec fpm-test bash /tmp/test-scenarios.sh         # 7 scenarios
docker exec fpm-test bash /tmp/test-venv-isolation.sh    # 16 tests
docker exec fpm-test bash /tmp/test-snapshot-scenarios.sh # 37 tests
docker exec fpm-test bash /tmp/test-multiuser.sh         # 16 tests
```

### Run specific test group

```bash
docker exec fpm-test bash /tmp/test-features.sh install   # just install tests
docker exec fpm-test bash /tmp/test-features.sh snapshot  # just snapshot tests
docker exec fpm-test bash /tmp/test-features.sh --list    # list all groups
```

---

## Building and Deploying

### Quick rebuild + deploy cycle

```bash
GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" -o bin/fpm-linux ./cmd/fpm && \
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm
```

### Run unit tests locally

```bash
make test           # go test ./...
go vet ./...        # static analysis
make build          # build for host platform
```

---

## fpm Configuration

### User config (`~/.config/fpm/config.toml`)

```toml
[tool]
cross-manager-policy = "ask"    # ask | install | skip
link-mode = "auto"              # auto | hardlink | copy | symlink
concurrency = 50
mode = "single-user"            # single-user | multi-user

[cache]
dir = "/custom/cache/path"      # override cache location

[[index]]
name = "pypi"
url = "https://pypi.org/simple"
default = true

[network]
allow-insecure-host = ["internal-pypi.corp.example.com"]

[log]
level = "off"                   # debug | info | warn | error | off
```

### Project config (`fpm.toml`)

```toml
[project]
name = "myproject"
requires-python = ">=3.10"
dependencies = ["requests", "flask"]

[immutable]
packages = [
    { name = "requests", version = "2.34.2" },
    { name = "cryptography", version = "41.0.0" }
]

[tool]
cross-manager-policy = "skip"   # project-level override
```

### Config priority (highest wins)

1. CLI flags (`--system`, `--allow-insecure-host`)
2. Environment variables (`FPM_*`)
3. Project `fpm.toml`
4. User `~/.config/fpm/config.toml`
5. System `/etc/fpm/config.toml`
6. Defaults

---

## Key Commands Quick Reference

```bash
# Project lifecycle
fpm init .                          # create project + venv
fpm install requests flask          # install packages
fpm remove flask                    # remove package
fpm remove -p flask                 # remove + purge deps
fpm autoremove                      # clean orphans
fpm list                            # list fpm packages
fpm list -a                         # list ALL (pip/uv/conda too)
fpm list -m                         # show mutable/immutable status
fpm list --system                   # system packages
fpm tree                            # dependency tree
fpm run python main.py              # run in managed env
fpm lock                            # generate lockfile
fpm sync                            # install from lockfile

# Snapshots
fpm snapshot create "message"       # capture state
fpm snapshot list                   # show history
fpm snapshot diff <id>              # compare vs current
fpm snapshot restore <id>           # rollback
fpm snapshot delete <id>            # remove snapshot
fpm snapshot create --system "msg"  # system-level snapshot

# System
fpm install -s numpy                # install to system
fpm remove -s numpy                 # remove from system
fpm audit                           # vulnerability scan
fpm audit --system                  # scan system packages
fpm repair                          # diagnose issues
fpm config show                     # display config
fpm config set tool.mode multi-user # enable multi-user
fpm cache size                      # disk usage
fpm cache gc                        # garbage collect
```

---

## Debugging Tips

```bash
# Check what fpm sees
fpm list -a --system 2>&1           # all system packages
fpm config show                     # current config
fpm repair                          # health check

# Check file ownership
ls -la /usr/local/lib/python3.12/site-packages/*.dist-info/INSTALLER

# Check depgraph
cat ~/.local/share/fpm/depgraph.json | python3 -m json.tool

# Check snapshots
ls ~/.cache/fpm/snapshots/
cat ~/.cache/fpm/snapshots/*/snap-*.json | python3 -m json.tool

# Check CAS
du -sh ~/.cache/fpm/cas/
ls ~/.cache/fpm/cas/sha256/ | wc -l

# Check locks (should be empty when idle)
find ~/.cache/fpm -name "*.lock" -ls
find ~/.local/share/fpm -name "*.lock" -ls
```
