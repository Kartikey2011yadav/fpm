# audit

Vulnerability scanning for installed packages using the OSV database.

## Key Types

- `Vulnerability` — security issue (ID, summary, severity, fix version)
- `AuditResult` — per-package vulnerability findings
- `AuditReport` — full scan results

## Features

- Query OSV (Open Source Vulnerabilities) API for PyPI packages
- Report severity, affected versions, and fix versions
- Link to vulnerability details

## Usage

```bash
fpm audit
# Scanned 45 packages — found 2 vulnerabilities:
#   requests 2.28.0
#     GHSA-xxx [HIGH] SSRF via crafted URL
#       fix: upgrade to 2.31.0
```

## Files

- `audit.go` — scanning, OSV API client, report formatting
