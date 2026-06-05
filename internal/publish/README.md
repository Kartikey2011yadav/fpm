# publish

Upload Python distributions to PyPI and compatible registries.

## Key Types

- `Publisher` — upload client with authentication
- `PublishOptions` — repository URL, token, credentials

## Features

- Upload wheels and sdists via multipart POST
- Token-based authentication (`__token__` username)
- Support for PyPI, TestPyPI, and private registries
- Environment variable for token (`FPM_PUBLISH_TOKEN`)

## Files

- `publish.go` — `Upload()`, multipart file upload protocol
