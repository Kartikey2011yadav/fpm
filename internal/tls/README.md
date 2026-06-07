# internal/tls

TLS certificate configuration for fpm's HTTP client. Implements uv-style
certificate resolution with bundled Mozilla CA fallback.

## Certificate Precedence

1. **SSL_CERT_FILE / SSL_CERT_DIR** — if set, these are the ONLY trusted roots
2. **System certificate pool** — platform's native cert store
3. **Bundled Mozilla CAs** — fallback via `github.com/breml/rootcerts`

## Key Types

- `LoadCertPool() (*x509.CertPool, error)` — loads custom certs from env vars
- `InsecureHostTransport` — per-host TLS bypass (wraps `http.RoundTripper`)
- `NormalizeHost(host string) string` — normalizes host for comparison

## Environment Variables

| Variable                  | Effect                                          |
| ------------------------- | ----------------------------------------------- |
| `SSL_CERT_FILE`           | PEM bundle file path (overrides system certs)   |
| `SSL_CERT_DIR`            | Directory of PEM certs (overrides system certs) |
| `FPM_INSECURE=1`          | Skip ALL TLS verification                       |
| `FPM_ALLOW_INSECURE_HOST` | Comma-separated hosts to skip TLS for           |

## Usage

```go
import fpmtls "github.com/kartikeyyadav/fpm/internal/tls"

pool, err := fpmtls.LoadCertPool()
if pool != nil {
    transport.TLSClientConfig = &tls.Config{RootCAs: pool}
}

// Per-host bypass
rt := fpmtls.NewInsecureHostTransport(transport, []string{"internal.corp.com"})
client := &http.Client{Transport: rt}
```
