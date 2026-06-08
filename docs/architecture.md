# fpm Architecture — Flows & Directories

## Directory Layout

### Cache (`~/.cache/fpm/` or `FPM_CACHE_DIR`)

```
~/.cache/fpm/
├── wheels/                     # Downloaded .whl files (raw)
│   └── requests-2.31.0-py3-none-any.whl
├── cas/sha256/                 # Content-Addressable Storage (extracted)
│   └── ab/                     # First 2 chars of hash (prefix)
│       └── ab3f7c8d...full_hash/  # Extracted wheel contents
│           ├── requests/
│           ├── requests-2.31.0.dist-info/
│           └── ...
├── http/                       # HTTP response cache (PyPI metadata)
│   └── requests.json           # Cached package index (10-min TTL)
├── refs/                       # Reference tracking (which envs use what)
│   ├── by-env/                 # env_hash → list of CAS keys
│   └── by-cas/                 # CAS key → list of environments
└── tmp/                        # Atomic staging area for extractions
```

### Installation Targets

| Scenario              | Target Directory                          |
| --------------------- | ----------------------------------------- |
| Inside venv (default) | `.venv/lib/python3.X/site-packages/`      |
| No venv (system-wide) | `/usr/local/lib/python3.X/dist-packages/` |
| `--system` flag       | Forces system-wide even inside a venv     |

### Project Files

```
my-project/
├── fpm.toml                    # Project config (deps, settings)
├── fpm.lock                    # Deterministic lockfile
├── pyproject.toml              # PEP 621 project metadata
└── .venv/                      # Virtual environment
    ├── pyvenv.cfg
    ├── bin/                    # Scripts and entry points
    └── lib/python3.X/site-packages/
        ├── requests/           # Package files (linked from CAS)
        ├── requests-2.31.0.dist-info/
        │   ├── METADATA
        │   ├── RECORD
        │   └── INSTALLER       # Contains "fpm\n"
        └── ...
```

---

## Flow: `fpm install requests`

```
┌─────────────────────────────────────────────────────────────────────┐
│ 1. PARSE                                                             │
│    Parse "requests" as PEP 508 requirement                          │
│    → {name: "requests", specifiers: [], extras: []}                 │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 2. DETECT ENVIRONMENT                                                │
│    Walk up from cwd looking for pyvenv.cfg                          │
│    Found .venv? → target = .venv/lib/python3.X/site-packages/       │
│    No venv?     → find system Python → target = its site-packages   │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 3. RESOLVE DEPENDENCIES                                              │
│    Fetch metadata: GET https://pypi.org/simple/requests/            │
│    → Parse JSON (PEP 691) or HTML (PEP 503)                        │
│    → Filter wheels by Python version + platform tags                │
│    → Resolve dependency graph (PubGrub algorithm)                   │
│    → Check immutable pins (fpm.toml [immutable])                    │
│    Result: [requests==2.31.0, urllib3==2.0.0, certifi==2023.7, ...] │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 4. DOWNLOAD                                                          │
│    For each resolved package:                                        │
│    a. Check cache: ~/.cache/fpm/wheels/requests-2.31.0-*.whl?       │
│       → Cache hit: skip download                                    │
│       → Cache miss: download from files.pythonhosted.org            │
│    b. Save to ~/.cache/fpm/wheels/                                  │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 5. STORE IN CAS                                                      │
│    a. SHA-256 hash the .whl file → key                              │
│    b. Check CAS: ~/.cache/fpm/cas/sha256/<prefix>/<hash>/           │
│       → Already exists: skip extraction                             │
│       → New: extract .whl to tmp/, atomic rename to CAS path        │
│    c. Each package stored ONCE regardless of how many envs use it   │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 6. LINK TO SITE-PACKAGES                                             │
│    Link files from CAS → target site-packages:                      │
│    Strategy (auto-selected):                                         │
│      macOS APFS → reflink (copy-on-write, instant, no space)        │
│      Linux btrfs/xfs → reflink                                      │
│      Other → hardlink (same inode, no space duplication)             │
│      Fallback → copy (works everywhere)                             │
│                                                                      │
│    Result: .venv/lib/.../requests/ ←link→ ~/.cache/fpm/cas/.../     │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 7. MARK OWNERSHIP                                                    │
│    Write "fpm\n" to <dist-info>/INSTALLER                           │
│    → Allows fpm and pip to identify who installed what              │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 8. TRACK REFERENCES                                                  │
│    Update refs/by-env/<env_hash>.json with new CAS keys             │
│    Update refs/by-cas/<cas_key>.json with this environment          │
│    → Enables garbage collection: unreferenced CAS entries removable │
└─────────────────────────────────┬───────────────────────────────────┘
                                  │
┌─────────────────────────────────▼───────────────────────────────────┐
│ 9. UPDATE PROJECT FILES                                              │
│    a. Add "requests" to pyproject.toml [project].dependencies       │
│    b. Write/update fpm.lock with resolved versions + hashes         │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Flow: `fpm remove requests`

1. Read `RECORD` file from `requests-*.dist-info/`
2. Delete all listed files from site-packages
3. Remove dist-info directory
4. Update pyproject.toml (remove from dependencies)
5. Update reference tracker (remove CAS reference for this env)
6. Note: CAS entry stays (other envs may use it; `fpm cache gc` cleans)

---

## Flow: `fpm list`

1. Detect environment (venv or system)
2. Scan site-packages directories for `*.dist-info/` folders
3. Parse `METADATA` for name + version
4. Read `INSTALLER` file to detect manager (fpm, pip, uv, conda, etc.)
5. Fallback: use path heuristics (`/usr/lib/` → system)
6. Filter: `fpm list` shows only fpm-managed; `fpm list -a` shows all

---

## Flow: `fpm cache gc`

1. Read all `refs/by-cas/` entries
2. For each CAS key, check if any referenced environment still exists
3. If no env references remain → delete CAS entry + wheel file
4. Report space reclaimed

---

## Manager Detection Logic

```
INSTALLER file exists?
├── Yes → read content → "fpm" | "pip" | "uv" | "conda" | "poetry" | "pdm"
└── No  → use path heuristics:
          /usr/lib/python3/ (not /usr/local/) → "system" (distro package)
          /opt/conda/ → "conda"
          Everything else → "pip" (most likely installed by pip without INSTALLER)
```

---

## TLS Certificate Chain

```
SSL_CERT_FILE / SSL_CERT_DIR set?
├── Yes → use ONLY those certs (complete override)
└── No  → System cert pool available?
          ├── Yes → use system certs
          └── No  → bundled Mozilla CAs (github.com/breml/rootcerts)

Per-host bypass: --allow-insecure-host / FPM_ALLOW_INSECURE_HOST
Global bypass:   FPM_INSECURE=1
```
