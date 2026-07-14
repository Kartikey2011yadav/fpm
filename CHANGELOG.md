# Changelog

All notable changes to fpm will be documented in this file.

## [0.3.0] - 2026-07-14

### Added
- **Git-like workflow commands:** `fpm status`, `fpm log`, `fpm tag`, `fpm stash`,
  `fpm revert`, `fpm blame`, `fpm cherry-pick`, `fpm bisect`, `fpm branch`
- **Shell completion:** `fpm completion bash/zsh/fish/powershell`
- **Documentation command:** `fpm docs [topic]` with 8 inline topics
- **Operation journal:** all install/remove operations recorded for history/revert
- **Pre-flight permission checks** with user-friendly error hints
- **Streaming SHA256 verification** during wheel downloads
- **Windows PowerShell installer** (`install.ps1`)
- **Comprehensive test suite:** 34 depgraph tests, 10 snapshot tests, 10 E2E scenarios
- **Git-inspired feature roadmap** (`docs/roadmap-git-features.md`)

### Fixed
- **TLS on corporate networks:** removed `breml/rootcerts` that broke Zscaler/enterprise proxies
- **`fpm python install` 404s:** now queries GitHub API dynamically for python-build-standalone
- **`remove -p` purging all dependencies:** fallback orphan detection no longer runs when graph has data
- **Windows `UnlockFileEx` crash:** fixed API argument order (access violation)
- **Depgraph race condition:** lock held during entire Load (read+parse) and Save (write lock)
- **Dangling references blocking GC:** validates environment paths exist before considering "in use"
- **Branch switch race:** acquires venv lock during switch
- **Snapshot restore race:** acquires venv lock during CleanSitePackages

### Changed
- HTTP timeout increased from 30s to 5min (large wheels like numpy/torch)
- CI uses `go-version-file: "go.mod"` instead of `"stable"`
- golangci-lint action upgraded to v7 (supports golangci-lint v2)
- Docker image tags forced lowercase
- GoReleaser action upgraded to v6
- All gofmt issues resolved across codebase
- PyPI package renamed from `fpm-pkg` to `fpm-cli`

## [0.2.1] - 2026-07-13

### Fixed
- CI pipeline fixes (golangci-lint version, Go version)
- Homebrew workflow trigger (tag push instead of release event)
- Docker lowercase tags

## [0.2.0] - 2026-07-13

### Added
- Pre-commit hooks (`.githooks/pre-commit`)
- `.golangci.yml` configuration
- Security hardening (crypto/rand, response limits, atomic cache write)
- Self-update SHA256 verification
- Install script SHA256 verification

## [0.1.0] - 2026-06-07

### Added
- Initial release
- Package install/remove/sync/lock
- PubGrub dependency resolver
- Content-addressable storage with reflink/hardlink
- Environment snapshots (create/restore/diff)
- Cross-manager awareness (pip, uv, conda, poetry, pdm)
- Immutable version pinning
- Smart dependency tracking (requested vs transitive)
- Autoremove orphaned dependencies
- Vulnerability scanning (OSV)
- Multi-platform support (macOS, Linux, Windows)
