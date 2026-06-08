# Release Guide

How to publish a new fpm release. Each release automatically distributes to:
GitHub Releases, Docker (ghcr.io), PyPI, and Homebrew.

## Prerequisites (One-Time Setup)

### 1. Homebrew Tap Repository

Create `github.com/Kartikey2011yadav/homebrew-tap`:

```bash
mkdir homebrew-tap && cd homebrew-tap
git init
mkdir Formula
cp /path/to/fpm/homebrew/fpm.rb Formula/
git add . && git commit -m "Initial fpm formula"
git remote add origin https://github.com/Kartikey2011yadav/homebrew-tap.git
git push -u origin main
```

### 2. GitHub Secrets

Go to: fpm repo → Settings → Secrets and variables → Actions → New repository secret

| Secret | Value | Purpose |
|--------|-------|---------|
| `TAP_GITHUB_TOKEN` | Personal Access Token (repo scope) | Push formula updates to homebrew-tap |

`GITHUB_TOKEN` is automatic — no setup needed.

### 3. PyPI Trusted Publisher

1. Go to https://pypi.org → Account → Publishing → Add new pending publisher
2. Fill in:
   - PyPI project name: `fpm-pkg`
   - Owner: `Kartikey2011yadav`
   - Repository name: `fpm`
   - Workflow name: `pypi.yml`
   - Environment: (leave blank)
3. The first release that triggers `pypi.yml` will claim the name.

### 4. GoReleaser

Already configured in `.goreleaser.yml`. No setup needed — runs via GitHub Actions.

---

## Publishing a Release

### Step 1: Verify everything passes

```bash
# Clean working tree
git status

# All tests pass
make test

# All integration tests pass
docker exec fpm-test bash /tmp/test-features.sh

# go vet clean
go vet ./...
```

### Step 2: Run the release script

```bash
./scripts/release.sh 0.2.0
```

This script:
1. Verifies clean git state
2. Runs full test suite
3. Builds all platform binaries
4. Creates annotated git tag `v0.2.0`

### Step 3: Push the tag

```bash
git push origin main
git push origin v0.2.0
```

### Step 4: Verify CI completes

Check GitHub Actions for these workflows:

| Workflow | Trigger | What it does |
|----------|---------|--------------|
| `release.yml` | `v*` tag | GoReleaser → GitHub Release with binaries + checksums |
| `docker.yml` | `v*` tag | Multi-arch Docker image → ghcr.io |
| `pypi.yml` | `v*` tag | Publishes `fpm-pkg` to PyPI |
| `homebrew.yml` | Release published | Updates Formula in homebrew-tap with SHA256 |

### Step 5: Verify distribution

```bash
# GitHub Release
gh release view v0.2.0

# Docker
docker pull ghcr.io/kartikey2011yadav/fpm:0.2.0
docker run --rm ghcr.io/kartikey2011yadav/fpm:0.2.0 --version

# PyPI
pip install fpm-pkg==0.2.0
fpm --version

# Homebrew (after tap updates)
brew update
brew install kartikeyyadav/tap/fpm
fpm --version
```

---

## Release Artifacts

GoReleaser produces these files per release:

```
fpm-0.2.0-linux-amd64.tar.gz
fpm-0.2.0-linux-arm64.tar.gz
fpm-0.2.0-darwin-amd64.tar.gz
fpm-0.2.0-darwin-arm64.tar.gz
fpm-0.2.0-windows-amd64.zip
checksums.txt
```

Each `.tar.gz` contains a single `fpm` binary. The `checksums.txt` has SHA256 for all files.

---

## Version Numbering

Follow semver: `MAJOR.MINOR.PATCH`

| Change | Version bump | Example |
|--------|-------------|---------|
| Breaking CLI change | Major | `--global` → `--system` |
| New feature | Minor | Add `fpm repair` command |
| Bug fix | Patch | Fix TLS error message |

Pre-releases: `0.2.0-beta.1` (GoReleaser auto-marks as prerelease)

---

## CI Workflows Reference

### `ci.yml` — Continuous Integration

- **Trigger:** Push to main, PRs to main
- **Matrix:** Ubuntu, macOS, Windows
- **Jobs:** Build, test (with race detector), lint, coverage

### `release.yml` — Binary Distribution

- **Trigger:** Push tag `v*`
- **Tool:** GoReleaser
- **Output:** GitHub Release with multi-platform binaries + checksums + changelog

### `docker.yml` — Container Image

- **Trigger:** Push tag `v*`
- **Platforms:** linux/amd64, linux/arm64
- **Registry:** ghcr.io/kartikey2011yadav/fpm
- **Tags:** `:{version}` and `:latest`

### `pypi.yml` — Python Package

- **Trigger:** Push tag `v*`
- **Package:** `fpm-pkg` (wrapper that downloads Go binary)
- **Auth:** PyPI trusted publisher (OIDC, no token needed)

### `homebrew.yml` — Homebrew Formula

- **Trigger:** Release published
- **Action:** Computes SHA256 for all tarballs, generates formula, pushes to tap repo
- **Auth:** `TAP_GITHUB_TOKEN` secret

---

## Hotfix Release

For urgent fixes without releasing unreleased features:

```bash
# Branch from the release tag
git checkout -b hotfix/0.2.1 v0.2.0

# Make the fix
# ...

# Tag and push
git tag v0.2.1
git push origin hotfix/0.2.1
git push origin v0.2.1

# Merge back to main
git checkout main
git merge hotfix/0.2.1
```

---

## Troubleshooting Releases

### GoReleaser fails

- Check Go version in `release.yml` (should be `stable`)
- Ensure `.goreleaser.yml` is valid: `goreleaser check`
- Verify tag format: must be `v{semver}` (e.g., `v0.2.0`)

### Docker push fails

- Check `packages: write` permission in workflow
- Verify image name matches repo: `ghcr.io/kartikey2011yadav/fpm`

### PyPI publish fails

- Verify trusted publisher is configured on pypi.org
- Check workflow name matches (`pypi.yml`)
- First publish must claim the name `fpm-pkg`

### Homebrew tap not updating

- Verify `TAP_GITHUB_TOKEN` secret is set and not expired
- Check homebrew-tap repo exists with `Formula/` directory
- Verify release assets are downloadable (checksums depend on them)

---

## Rollback

If a release has a critical bug:

```bash
# Delete the GitHub release (keeps the tag)
gh release delete v0.2.0

# Or mark as draft/prerelease
gh release edit v0.2.0 --draft

# Users on old version are unaffected
# Fix, then re-release as v0.2.1
```

Docker `:latest` tag will remain on the broken version until the next push.
Pin specific versions in production: `ghcr.io/kartikey2011yadav/fpm:0.1.0`
