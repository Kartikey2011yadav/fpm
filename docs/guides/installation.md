# Installation Guide

## Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash
```

This launches an interactive installer with:
- Platform detection (macOS/Linux, Intel/ARM)
- Default vs Custom path selection
- Disk space verification
- Shell PATH configuration

For CI/Docker (non-interactive):
```bash
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash -s -- --yes
```

---

## All Installation Methods

| Method | Command | Best for |
|--------|---------|----------|
| Install script | `curl ... \| bash` | Most users (interactive) |
| pip | `pip install fpm-cli` | Python developers |
| Go | `go install .../cmd/fpm@latest` | Go developers |
| Homebrew | `brew install kartikeyyadav/tap/fpm` | macOS users |
| Docker | `docker run ghcr.io/kartikey2011yadav/fpm` | Containers/CI |
| From source | `git clone && make build` | Contributors |

---

## pip

```bash
pip install fpm-cli
```

This installs a thin Python wrapper that automatically downloads the correct
native binary for your platform. After install, `fpm` is available in your PATH.

Works on: macOS, Linux, Windows (Intel and ARM).

---

## Go

```bash
go install github.com/kartikeyyadav/fpm/cmd/fpm@latest
```

Requires Go 1.25+. The binary is placed in `$GOPATH/bin/`.

---

## Homebrew (macOS/Linux)

```bash
brew install kartikeyyadav/tap/fpm
```

Updates:
```bash
brew upgrade fpm
```

---

## Docker

```bash
# Run directly
docker run --rm -v $(pwd):/io ghcr.io/kartikey2011yadav/fpm install -s requests

# Use in Dockerfile
COPY --from=ghcr.io/kartikey2011yadav/fpm:latest /fpm /usr/local/bin/fpm
```

Multi-arch: supports `linux/amd64` and `linux/arm64`.

---

## From Source

```bash
git clone https://github.com/Kartikey2011yadav/fpm.git
cd fpm
make build
./bin/fpm --version
```

Cross-compile for all platforms:
```bash
make build-all
# Outputs: bin/fpm-linux-amd64, bin/fpm-darwin-arm64, etc.
```

---

## Custom Installation Paths

### Via Interactive Installer

Run `curl ... | bash` and select "Custom" mode. You'll be prompted for:
- Binary path (`/usr/local/bin` vs `~/.local/bin` vs custom)
- Cache directory (package downloads, metadata)
- Data directory (Python versions, tools)
- Tool bin directory (tool executables)

### Via Environment Variables

```bash
# Set before running installer
FPM_INSTALL_DIR=/custom/bin curl ... | bash --yes

# Or configure after install
fpm config set cache.dir /data/fpm-cache
```

| Variable | Purpose | Default |
|----------|---------|---------|
| `FPM_INSTALL_DIR` | Binary location | `~/.local/bin` |
| `FPM_CACHE_DIR` | Cache directory | `~/.cache/fpm` |
| `FPM_DATA_DIR` | Data directory | `~/.local/share/fpm` |
| `FPM_CONFIG_DIR` | Config directory | `~/.config/fpm` |
| `FPM_TOOL_BIN_DIR` | Tool executables | `~/.local/bin` |

---

## Platform Compatibility

| Platform | Architecture | Status |
|----------|-------------|--------|
| macOS | Apple Silicon (M1/M2/M3/M4) | Supported |
| macOS | Intel | Supported |
| Linux | x86_64 | Supported |
| Linux | ARM64 (Raspberry Pi, AWS Graviton) | Supported |
| Windows | x86_64 | Supported |
| Docker | linux/amd64, linux/arm64 | Supported |
| JupyterHub | Per-user install | Supported |
| Multi-user systems | Per-user or system-wide | Supported |

---

## Special Environments

### JupyterHub / JupyterLab

The installer auto-detects JupyterHub (`$JUPYTERHUB_USER`) and installs per-user:

```bash
# In a Jupyter terminal
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash --yes
```

Binary goes to `~/.local/bin/fpm`, no sudo needed.

### Multi-user Systems

For shared systems (university clusters, shared servers):

**Per-user install (no sudo):**
```bash
FPM_INSTALL_DIR=~/.local/bin curl ... | bash --yes
```

**System-wide install (admin):**
```bash
sudo FPM_INSTALL_DIR=/usr/local/bin curl ... | bash --yes
```

### Docker / CI

```bash
# Minimal one-liner for Dockerfiles
RUN curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install-simple.sh | sh

# Or copy from the official image (faster, no network)
COPY --from=ghcr.io/kartikey2011yadav/fpm:latest /fpm /usr/local/bin/fpm
```

### VPN / Corporate Proxy

If TLS verification fails during install (VPN intercepting HTTPS):

```bash
# After install, configure fpm for VPN:
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
```

---

## Verifying Installation

```bash
fpm -v                # Print version
fpm repair            # Check all directories and configuration
fpm config show       # Display all paths and settings
```

---

## Uninstallation

```bash
# Remove binary
rm $(which fpm)

# Remove all data (cache, config, tools, managed Pythons)
rm -rf ~/.cache/fpm
rm -rf ~/.local/share/fpm
rm -rf ~/.config/fpm

# Remove PATH entry from shell profile (if added)
# Edit ~/.bashrc or ~/.zshrc and remove the fpm line
```

---

## Updating

```bash
# Self-update (downloads latest release)
fpm self update

# Or re-run installer
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | bash --yes

# Or via package manager
brew upgrade fpm               # Homebrew
pip install --upgrade fpm-cli  # pip
go install .../cmd/fpm@latest  # Go
```

---

For maintainer release workflows, see [docs/releasing.md](../releasing.md).

---

## Troubleshooting

### "Permission denied" during install

```bash
# Use per-user install (no sudo needed)
FPM_INSTALL_DIR=~/.local/bin curl ... | bash --yes
```

### "command not found" after install

Your shell doesn't see `~/.local/bin`. Add to PATH:

```bash
# bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc && source ~/.bashrc

# zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc && source ~/.zshrc

# fish
echo 'set -gx PATH "$HOME/.local/bin" $PATH' >> ~/.config/fish/config.fish
```

### pip install fails with "not found"

The package `fpm-cli` needs to be published to PyPI first (happens on first release).
Until then, use the install script or `go install`.

### Docker image not found

Published on first release. Until then, build locally:
```bash
docker build -t fpm:local .
docker run --rm fpm:local --version
```
