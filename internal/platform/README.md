# platform

Platform detection and [PEP 425](https://peps.python.org/pep-0425/) wheel tag
generation.

## Key Types

- `Platform` — current OS and architecture
- `Tag` — wheel compatibility tag (python-abi-platform)
- `TagSet` — ordered set of compatible tags with priority

## Supported Platforms

- Linux (x86_64, aarch64, armv7l, ppc64le, s390x) with manylinux versions
- macOS (x86_64, arm64) with version tags
- Windows (amd64, x86, arm64)

## Usage

```go
plat := platform.Current()
tags := platform.GenerateTags("311", plat)
compatible, priority := tags.Compatible(wheelTags)
```

## Files

- `platform.go` — OS/arch detection, platform tag strings
- `tags.go` — tag generation, wheel compatibility scoring
