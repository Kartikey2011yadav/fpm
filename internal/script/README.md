# script

PEP 723 inline script metadata parsing for single-file Python scripts.

## Key Types

- `ScriptMetadata` — parsed inline metadata (dependencies, requires-python)

## PEP 723 Format

```python
# /// script
# dependencies = ["requests>=2.28", "rich"]
# requires-python = ">=3.10"
# ///
import requests
```

## Features

- Parse TOML metadata from `# /// script` blocks
- Detect if a script has inline dependencies
- Used by `fpm run script.py` to auto-install deps in ephemeral environments

## Files

- `script.go` — `ParseInlineMetadata()`, `HasInlineMetadata()`
