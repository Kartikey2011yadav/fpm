# cache

Content-addressable storage (CAS) with SHA-256 hashing, reference tracking, and
garbage collection.

## Key Types

- `Cache` — root cache interface (store, retrieve, has, clean)
- `CASKey` — content-addressed identifier (algorithm + digest)
- `RefTracker` — bidirectional reference tracking (env ↔ CAS key)
- `GCResult` — garbage collection results

## Cache Layout

```
~/.cache/fpm/
├── cas/sha256/<prefix>/<hash>/   # Extracted wheel contents
├── wheels/                       # Raw .whl downloads
├── http/                         # HTTP response cache
├── refs/by-env/                  # env → CAS keys mapping
├── refs/by-cas/                  # CAS key → environments mapping
└── tmp/                          # Atomic staging area
```

## Garbage Collection

Packages are only GC'd when no environment or snapshot references them:

```go
cache.GC(GCOptions{OlderThan: 30 * 24 * time.Hour})
```

## Files

- `cache.go` — CAS store/retrieve, hash computation
- `extract.go` — zip extraction with path traversal protection
- `reference.go` — bidirectional reference tracking
- `gc.go` — garbage collection, prune, size calculation
