#!/bin/bash
# fpm Feature Test Suite
# Run inside Docker: docker exec -it fpm-test bash /path/to/test-features.sh
# Or: docker cp scripts/test-features.sh fpm-test:/tmp/ && docker exec fpm-test bash /tmp/test-features.sh

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'
BOLD='\033[1m'

PASS=0
FAIL=0
SKIP=0

pass() { echo -e "  ${GREEN}✓${NC} $1"; PASS=$((PASS + 1)); }
fail() { echo -e "  ${RED}✗${NC} $1"; FAIL=$((FAIL + 1)); }
skip() { echo -e "  ${YELLOW}○${NC} $1 (skipped)"; SKIP=$((SKIP + 1)); }
section() { echo -e "\n${BOLD}${CYAN}[$1]${NC}"; }

# Setup
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev
WORKDIR=$(mktemp -d)
cd "$WORKDIR"

section "1. CLI Basics"
fpm -v >/dev/null 2>&1 && pass "fpm -v prints version" || fail "fpm -v"
fpm --version >/dev/null 2>&1 && pass "fpm --version works" || fail "fpm --version"
fpm -h >/dev/null 2>&1 && pass "fpm -h shows help" || fail "fpm -h"
fpm --help | grep -q "Package Management:" && pass "help shows command groups" || fail "command groups in help"
fpm --help | grep -q "\-s, --system" && pass "--system flag in help" || fail "--system flag"
fpm --help | grep -q "allow-insecure-host" && pass "--allow-insecure-host in help" || fail "--allow-insecure-host"

section "2. Install (System Mode)"
fpm install requests 2>&1 | grep -q "No virtual environment found" && pass "install without --system errors" || fail "install without --system should error"
fpm install -s requests >/dev/null 2>&1 && pass "fpm install -s requests" || fail "install -s requests"
fpm install --system flask >/dev/null 2>&1 && pass "fpm install --system flask" || fail "install --system flask"

section "3. List Command"
fpm list | grep -q "requests" && pass "fpm list shows fpm packages" || fail "fpm list"
fpm list -a | grep -q "pip" && pass "fpm list -a shows pip packages" || fail "fpm list -a"
fpm list --manager fpm | grep -q "requests" && pass "fpm list --manager fpm" || fail "list --manager"
COUNT=$(fpm list 2>/dev/null | grep -c "fpm" || true)
[ "$COUNT" -gt 0 ] && pass "fpm list: $COUNT fpm packages found" || fail "no fpm packages"

section "4. Pip Compatibility"
fpm pip list | grep -q "requests" && pass "fpm pip list" || fail "fpm pip list"
fpm pip freeze --system | grep -q "requests==" && pass "fpm pip freeze --system" || fail "pip freeze"
fpm pip show --system requests | grep -q "Version:" && pass "fpm pip show --system" || fail "pip show"

section "5. Error Messages"
ERR=$(fpm install nonexistentpkg12345 -s 2>&1 || true)
echo "$ERR" | grep -q "not found" && pass "404: Package not found message" || fail "404 error message"
echo "$ERR" | grep -q "hint:" && pass "404: Hint shown" || fail "404 hint"

ERR2=$(fpm install request -s 2>&1 || true)
echo "$ERR2" | grep -qi "did you mean" && pass "did-you-mean suggestion for 'request'" || fail "did-you-mean"

section "6. Audit"
fpm audit --system >/dev/null 2>&1 && pass "fpm audit --system runs" || fail "audit --system"
# Second run should be faster (cached)
fpm audit --system >/dev/null 2>&1 && pass "fpm audit (cached run)" || fail "audit cached"

section "7. Config & Repair"
fpm config show | grep -q "cache:" && pass "fpm config show" || fail "config show"
fpm config show | grep -q "concurrency:" && pass "config shows settings" || fail "config settings"
fpm repair >/dev/null 2>&1 && pass "fpm repair runs" || fail "repair"

section "8. Project Workflow (venv)"
cd "$WORKDIR"
mkdir -p project && cd project
fpm init . >/dev/null 2>&1 && pass "fpm init" || fail "fpm init"
[ -f pyproject.toml ] && pass "pyproject.toml created" || fail "pyproject.toml missing"
[ -d .venv ] && pass ".venv created" || fail ".venv missing"

fpm install requests >/dev/null 2>&1 && pass "fpm install (in venv)" || fail "install in venv"
[ -f fpm.lock ] && pass "fpm.lock generated" || fail "fpm.lock missing"
grep -q "requests" pyproject.toml && pass "pyproject.toml updated" || fail "pyproject.toml not updated"

fpm list | grep -q "requests" && pass "fpm list in venv" || fail "list in venv"
fpm run python -c "import requests; print('ok')" 2>/dev/null && pass "fpm run python" || fail "fpm run"

section "9. Tree & Lock"
fpm tree >/dev/null 2>&1 && pass "fpm tree" || fail "fpm tree"
fpm lock >/dev/null 2>&1 && pass "fpm lock" || fail "fpm lock"

section "10. Snapshot System"
fpm snapshot create "test snapshot" >/dev/null 2>&1 && pass "snapshot create" || fail "snapshot create"
fpm snapshot list | grep -q "test snapshot" && pass "snapshot list shows entry" || fail "snapshot list"

# Install more, create second snapshot
fpm install click >/dev/null 2>&1
fpm snapshot create "added click" >/dev/null 2>&1 && pass "second snapshot" || fail "second snapshot"

SNAPS=$(fpm snapshot list 2>/dev/null)
SNAP1=$(echo "$SNAPS" | grep "test snapshot" | grep -oE '[0-9]{8}-[0-9]{6}-[0-9]+' | head -1)
SNAP2=$(echo "$SNAPS" | grep "added click" | grep -oE '[0-9]{8}-[0-9]{6}-[0-9]+' | head -1)

if [ -n "$SNAP1" ] && [ -n "$SNAP2" ]; then
    fpm snapshot diff "$SNAP1" "$SNAP2" >/dev/null 2>&1 && pass "snapshot diff" || fail "snapshot diff"
else
    skip "snapshot diff (couldn't parse IDs: SNAP1='$SNAP1' SNAP2='$SNAP2')"
fi

section "11. Remove & Uninstall"
fpm remove click >/dev/null 2>&1 && pass "fpm remove click" || fail "fpm remove"
fpm uninstall requests >/dev/null 2>&1 && pass "fpm uninstall (alias)" || fail "uninstall alias"

section "12. Python Discovery"
fpm python list >/dev/null 2>&1 && pass "fpm python list" || fail "python list"
fpm python list | grep -q "System:" && pass "shows system Python" || fail "system python in list"

section "13. Venv Creation"
cd "$WORKDIR"
mkdir -p venvtest && cd venvtest
fpm venv >/dev/null 2>&1 && pass "fpm venv" || fail "fpm venv"
[ -f .venv/pyvenv.cfg ] && pass "pyvenv.cfg exists" || fail "pyvenv.cfg missing"
.venv/bin/python --version >/dev/null 2>&1 && pass "venv python works" || fail "venv python broken"

section "14. Cache"
fpm cache size >/dev/null 2>&1 && pass "fpm cache size" || fail "cache size"

section "15. Cross-Manager Conflict Detection"
# uv is installed by pip — fpm should detect and skip
OUT=$(fpm install -s uv 2>&1)
echo "$OUT" | grep -q "already installed via pip" && pass "cross-manager: detects pip package" || fail "cross-manager detection"
echo "$OUT" | grep -q "Nothing to install\|skipping" && pass "cross-manager: skips existing" || fail "cross-manager skip"

section "16. Immutable Package Pinning"
cd "$WORKDIR"
mkdir -p immtest && cd immtest
fpm init . >/dev/null 2>&1
cat > fpm.toml << 'TOML'
[project]
name = "immtest"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "click", version = "8.4.1" }]
TOML
ERR=$(fpm install "click==7.0" 2>&1 || true)
echo "$ERR" | grep -q "immutable" && pass "immutable pin blocks wrong version" || fail "immutable pin"

section "17. Version Flag Variations"
fpm version 2>&1 | grep -q "fpm" && pass "fpm version subcommand" || fail "version subcommand"

# Cleanup
rm -rf "$WORKDIR"

# Summary
echo -e "\n${BOLD}════════════════════════════════════════${NC}"
echo -e "${BOLD}Results:${NC} ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}, ${YELLOW}$SKIP skipped${NC}"
echo -e "${BOLD}════════════════════════════════════════${NC}"

[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All tests passed!${NC}" || echo -e "\n${RED}Some tests failed.${NC}"
exit $FAIL
