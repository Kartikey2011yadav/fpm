# fpm-cli

**Fast Package Manager for Python** — binary distribution via pip.

This package provides the `fpm` command by downloading the pre-built binary for
your platform (macOS, Linux, Windows — Intel and ARM).

## Installation

```bash
pip install fpm-cli
```

## Usage

After installation, `fpm` is available:

```bash
fpm --help
fpm init myproject
fpm install requests
```

## What this package does

This is a thin Python wrapper that:

1. Downloads the correct `fpm` binary for your OS/architecture
2. Places it alongside your Python scripts
3. Proxies all commands to the native binary

The actual `fpm` tool is written in Go for maximum performance.

## Platforms

| OS      | Architecture | Supported |
| ------- | ------------ | --------- |
| Linux   | x86_64       | ✓         |
| Linux   | arm64        | ✓         |
| macOS   | x86_64       | ✓         |
| macOS   | arm64 (M1+)  | ✓         |
| Windows | x86_64       | ✓         |

## More information

- [GitHub](https://github.com/Kartikey2011yadav/fpm)
- [Documentation](https://github.com/Kartikey2011yadav/fpm/tree/main/docs)
