# Installation

## Quick Install (recommended)

### macOS / Linux

```bash
curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | sh
```

### With custom install directory

```bash
FPM_INSTALL_DIR=~/.local/bin curl -fsSL https://raw.githubusercontent.com/Kartikey2011yadav/fpm/main/install.sh | sh
```

## Package Managers

### Homebrew (macOS/Linux)

```bash
brew install kartikeyyadav/tap/fpm
```

### pip

```bash
pip install fpm-pkg
```

### go install

```bash
go install github.com/kartikeyyadav/fpm/cmd/fpm@latest
```

## From Source

```bash
git clone https://github.com/Kartikey2011yadav/fpm.git
cd fpm
make build
sudo mv bin/fpm /usr/local/bin/
```

## Docker

```bash
docker pull ghcr.io/kartikey2011yadav/fpm:latest
docker run --rm -v $(pwd):/io ghcr.io/kartikey2011yadav/fpm install requests
```

## Custom Paths

By default, fpm uses platform-standard directories. Override with environment
variables:

| Variable          | Default (Linux)      | Purpose                           |
| ----------------- | -------------------- | --------------------------------- |
| `FPM_CACHE_DIR`   | `~/.cache/fpm`       | Package cache (CAS, wheels, HTTP) |
| `FPM_DATA_DIR`    | `~/.local/share/fpm` | Python installs, tools, bin       |
| `FPM_CONFIG_DIR`  | `~/.config/fpm`      | User configuration                |
| `FPM_INSTALL_DIR` | `/usr/local/bin`     | Where to place the fpm binary     |

Set these in your shell profile to persist:

```bash
# ~/.bashrc or ~/.zshrc
export FPM_CACHE_DIR="/opt/fpm/cache"
export FPM_DATA_DIR="/opt/fpm/data"
```

## Verify Installation

```bash
fpm version
# fpm 0.1.0

fpm --help
```

## Updating

```bash
fpm self update
```

## Uninstalling

```bash
# Remove binary
rm $(which fpm)

# Remove data (optional)
rm -rf ~/.cache/fpm ~/.local/share/fpm ~/.config/fpm
```
