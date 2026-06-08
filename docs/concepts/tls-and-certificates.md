# TLS & Certificate Handling

## The Problem

Python package indexes (PyPI) use HTTPS. In many environments — Docker
containers, corporate VPNs, air-gapped networks — the system certificate
store is either missing or contains a proxy CA that isn't trusted by default.

This causes: `x509: certificate signed by unknown authority`

pip solves this by bundling `certifi`. uv bundles `webpki-root-certs`.
fpm bundles Mozilla's root certificates the same way.

## Certificate Resolution

fpm uses a layered approach (highest priority first):

```
1. SSL_CERT_FILE / SSL_CERT_DIR  → complete override (only these certs used)
2. System certificate pool        → platform's native cert store
3. Bundled Mozilla CAs            → fallback (via github.com/breml/rootcerts)
```

### Layer 1: Custom Certificates

If `SSL_CERT_FILE` or `SSL_CERT_DIR` is set, those are the **only** trusted
roots. System certs and bundled certs are ignored.

```bash
# Single file (PEM bundle)
SSL_CERT_FILE=/etc/ssl/custom-ca.pem fpm install requests

# Directory of PEM files
SSL_CERT_DIR=/etc/ssl/corp-certs/ fpm install requests
```

Use case: corporate environments with a specific CA that must be trusted.

### Layer 2: System Certificates

The default. Uses whatever certificates the OS provides:
- Linux: `/etc/ssl/certs/`, `/etc/pki/tls/certs/`
- macOS: Keychain
- Windows: Certificate Store

### Layer 3: Bundled Mozilla CAs

If the system cert pool is unavailable (Docker scratch image, minimal
container), fpm falls back to bundled Mozilla root certificates. These
are compiled into the binary via `github.com/breml/rootcerts`.

Updated by running `go get -u github.com/breml/rootcerts` and rebuilding.

## Per-Host Insecure Bypass

For VPN/proxy environments where a corporate proxy MITMs HTTPS:

```bash
# CLI flag (per-command)
fpm install --allow-insecure-host pypi.org --allow-insecure-host files.pythonhosted.org -s requests

# Environment variable (persistent)
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org

# Config file (per-project or user)
[network]
allow-insecure-host = ["pypi.org", "files.pythonhosted.org"]
```

This disables certificate verification **only for the specified hosts**.
All other HTTPS connections remain fully verified.

## Global Bypass (Nuclear Option)

```bash
FPM_INSECURE=1 fpm install -s requests
```

Disables ALL TLS verification. Use only for debugging — never in production.

## mTLS (Client Certificates)

For private package indexes that require mutual TLS:

```bash
SSL_CLIENT_CERT=/path/to/client-cert-and-key.pem fpm install -s private-package
```

Or in config:
```toml
[network]
client-cert = "/path/to/cert.pem"
client-key = "/path/to/key.pem"
```

## How It Works Internally

```
HTTP Request → InsecureHostTransport (per-host check)
                    ↓
              Matching host? → InsecureSkipVerify = true
              Other host?   → Normal TLS with configured cert pool
                                   ↓
                             RootCAs set? → use custom pool (SSL_CERT_FILE)
                             RootCAs nil? → Go uses system + bundled fallback
```

## Certificate Updates

Mozilla updates their CA store quarterly. The update flow:

1. Mozilla adds/removes CAs
2. `github.com/breml/rootcerts` package updates (automated CI)
3. fpm runs `go get -u github.com/breml/rootcerts`
4. Next fpm build includes fresh certs
5. Users update fpm (`fpm self update`)

## Developer Reference

Key code:
- `internal/tls/certs.go` — `LoadCertPool()`, `SSL_CERT_FILE`/`SSL_CERT_DIR` loading
- `internal/tls/transport.go` — `InsecureHostTransport`, per-host bypass
- `internal/client/client.go` — TLS config wiring in `New()`
- `cmd/fpm/main.go` — `import _ "github.com/breml/rootcerts"` (bundled CAs)
- `internal/config/config.go` — `NetworkConfig` struct
