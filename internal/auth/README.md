# auth

Authentication for PyPI and private package registries.

## Key Types

- `Authenticator` — credential resolution engine
- `Credential` — auth credential (Basic, Bearer, Token)
- `AuthTransport` — HTTP middleware that attaches auth to requests

## Credential Resolution Order

1. URL-embedded (`https://user:pass@index.example.com`)
2. Environment variables (`FPM_INDEX_TOKEN`, `FPM_INDEX_USERNAME`)
3. Config file credentials (`fpm.toml [[index]]` with token)
4. System keyring (planned)
5. `~/.netrc` file

## Files

- `auth.go` — `Authenticator`, credential types, resolution
- `netrc.go` — `.netrc` file parsing
- `middleware.go` — HTTP transport with auth injection
