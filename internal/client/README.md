# client

HTTP client for PyPI and compatible package registries using the PEP 691 JSON
Simple API.

## Key Types

- `RegistryClient` — HTTP client with caching, concurrency control, retry
- `SimpleProjectDetail` — PEP 691 response (list of available files)
- `SimpleFile` — individual file entry (URL, hashes, requires-python, yanked
  status)

## Features

- PEP 691 JSON Simple API (`application/vnd.pypi.simple.v1+json`)
- HTTP response caching (10-minute TTL matching PyPI's Cache-Control)
- Configurable concurrency (default: 50 parallel requests)
- Multiple index support (PyPI + private registries)
- User-Agent for good citizenship

## Files

- `client.go` — `RegistryClient`, fetch versions, download wheels, HTTP cache
