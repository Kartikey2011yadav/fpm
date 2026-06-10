# FPM Demo Scenarios

Step-by-step demo guides covering every major feature of fpm. Each scenario is
self-contained and can be presented independently.

## Prerequisites

```bash
# Build fpm from source
make build

# Ensure fpm binary is in PATH
export PATH="$PWD/bin:$PATH"

# Verify
fpm --version
```

> All scenarios assume a clean environment. Run commands in order within each
> scenario. Use `--system` (`-s`) flag when outside a virtual environment.

---

## Scenarios

| #   | Scenario                                                                         | What It Proves                                                            |
| --- | -------------------------------------------------------------------------------- | ------------------------------------------------------------------------- |
| 1   | [Installation & Environment Detection](01-installation-and-env-detection.md)     | fpm enforces venv/system boundaries, prevents accidental system pollution |
| 2   | [Dependency Graph & Intelligent Cleanup](02-dependency-graph-and-cleanup.md)     | fpm tracks requested vs transitive deps, enables smart autoremove         |
| 3   | [Immutable Package Pinning](03-immutable-packages.md)                            | Production-grade version locks that block conflicting installs            |
| 4   | [Snapshots & Rollback](04-snapshots-and-rollback.md)                             | Full-fidelity restore: all packages, all managers, config included        |
| 5   | [Cross-Manager Awareness](05-cross-manager-awareness.md)                         | fpm sees packages from pip/uv/conda/poetry and handles conflicts          |
| 6   | [Conflict Resolution Policies](06-conflict-resolution.md)                        | Three strategies (ask/install/skip) for cross-manager conflicts           |
| 7   | [Content-Addressable Storage & Zero Duplication](07-cas-and-zero-duplication.md) | One copy on disk shared across projects via reflinks                      |
| 8   | [Lockfile & Reproducible Environments](08-lockfile-and-sync.md)                  | Deterministic installs via fpm lock + fpm sync                            |
| 9   | [Vulnerability Auditing](09-vulnerability-audit.md)                              | Scan all packages (any manager) against OSV database                      |
| 10  | [Cache Management & Garbage Collection](10-cache-management.md)                  | Reference-tracked GC that only removes truly unused packages              |
| 11  | [Error UX & Helpful Hints](11-error-ux.md)                                       | Typo correction, clear diagnostics, actionable suggestions                |
| 12  | [Project Initialization & Workflow](12-project-workflow.md)                      | Full project lifecycle from init to build                                 |

---

## Running as Automated Demo

All scenarios can also be run via the test script in Docker:

```bash
# Build test container
docker build -t fpm-test -f Dockerfile.test .

# Run all scenarios
docker exec fpm-test bash /tmp/test-scenarios.sh
```

---

## Tips for Presenting

1. **Start with Scenario 1** — establishes baseline behavior
2. **Scenario 2 + 3** — core differentiators (dep graph + immutable)
3. **Scenario 4** — the "wow" moment (snapshots)
4. **Scenario 5 + 6** — real-world coexistence story
5. **Scenario 7** — technical depth (CAS architecture)
6. **End with Scenario 9** — security angle (audit)

Each scenario has a "What This Proves" section at the top and a "Key Takeaway"
at the bottom — useful for framing during a live demo.
