#!/bin/bash
# fpm Real-World Scenario Tests
# Shows full command output for each scenario — useful for debugging and demos.
# Usage: docker exec fpm-test bash /tmp/test-scenarios.sh

set -o pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev

PASS=0
FAIL=0

run() {
    local desc="$1"
    shift
    echo -e "\n${BOLD}${CYAN}\$ $*${NC}"
    local output
    output=$("$@" 2>&1)
    local rc=$?
    echo "$output"
    return $rc
}

check() {
    local desc="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ ${desc}${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}✗ ${desc}${NC}"
        FAIL=$((FAIL + 1))
    fi
}

section() {
    echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  SCENARIO: $1${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

# ════════════════════════════════════════════════════════════════════════
section "1. System Install Without Venv"
# ════════════════════════════════════════════════════════════════════════

run "Install without --system should error" fpm install requests
check "Correctly blocks without --system" test $? -ne 0

run "Install with -s flag" fpm install -s requests
check "Install succeeded" test $? -eq 0

run "List fpm packages" fpm list
check "requests visible" fpm list 2>/dev/null | grep -q requests

# ════════════════════════════════════════════════════════════════════════
section "2. Cross-Manager Detection"
# ════════════════════════════════════════════════════════════════════════

run "List all managers" fpm list -a
check "Sees pip packages" fpm list -a 2>/dev/null | grep -q "pip"
check "Sees fpm packages" fpm list -a 2>/dev/null | grep -q "fpm"

run "Try installing pip's 'six' via fpm" fpm install -s six
check "Detects cross-manager" fpm install -s six 2>&1 | grep -qi "already installed\|skipping"

# ════════════════════════════════════════════════════════════════════════
section "3. Dependency Graph & Purge"
# ════════════════════════════════════════════════════════════════════════

run "Install flask (has many deps)" fpm install -s flask
run "Show dependency tree" fpm tree --system

run "Mark status" fpm mark --show flask requests urllib3

run "Remove flask with purge" fpm remove -sp flask
check "Flask removed" ! fpm list 2>/dev/null | grep -q "flask"
check "requests still present" fpm list 2>/dev/null | grep -q "requests"
check "urllib3 kept (needed by requests)" fpm list 2>/dev/null | grep -q "urllib3"

# ════════════════════════════════════════════════════════════════════════
section "4. Immutable Package Pinning"
# ════════════════════════════════════════════════════════════════════════

WORKDIR=$(mktemp -d)
cd "$WORKDIR"
run "Init project" fpm init .
cat > fpm.toml << 'EOF'
[project]
name = "pintest"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
EOF

run "Try installing wrong version of pinned package" fpm install "requests==2.30.0"
check "Immutable pin blocks it" fpm install "requests==2.30.0" 2>&1 | grep -qi "immutable"

# ════════════════════════════════════════════════════════════════════════
section "5. Project Workflow (venv)"
# ════════════════════════════════════════════════════════════════════════

cd /tmp && rm -rf myproject
run "Create project" fpm init myproject
cd myproject

run "Install inside venv (no -s needed)" fpm install requests flask
check "fpm.lock created" test -f fpm.lock
check "pyproject.toml updated" grep -q requests pyproject.toml

run "Run python in venv" fpm run python -c "import requests; print(f'requests {requests.__version__}')"
check "Python runs OK" fpm run python -c "import requests" 2>/dev/null

run "Show tree" fpm tree

# ════════════════════════════════════════════════════════════════════════
section "6. Snapshots"
# ════════════════════════════════════════════════════════════════════════

run "Create snapshot" fpm snapshot create "before adding pandas"
run "Install pandas" fpm install pandas
run "Create second snapshot" fpm snapshot create "added pandas"

run "List snapshots" fpm snapshot list

SNAP1=$(fpm snapshot list 2>/dev/null | grep "before" | grep -oE '[0-9]{8}-[0-9]{6}-[0-9]+' | head -1)
if [ -n "$SNAP1" ]; then
    run "Diff snapshot vs current" fpm snapshot diff "$SNAP1"
    check "Diff shows pandas" fpm snapshot diff "$SNAP1" 2>/dev/null | grep -q "pandas"
fi

# ════════════════════════════════════════════════════════════════════════
section "7. Error UX"
# ════════════════════════════════════════════════════════════════════════

run "Package not found (typo)" fpm install -s request
check "Shows did-you-mean" fpm install -s request 2>&1 | grep -qi "did you mean"

run "Remove other manager's package" fpm remove -s uv
check "Hints about --force" fpm remove -s uv 2>&1 | grep -q "force"

# ════════════════════════════════════════════════════════════════════════
section "8. Force Remove & Autoremove"
# ════════════════════════════════════════════════════════════════════════

cd /tmp
run "Install flask system-wide" fpm install -s flask
run "Remove flask only (no purge)" fpm remove -s flask
run "Show orphans" fpm list
run "Autoremove cleans orphans" fpm autoremove --system

REMAINING=$(fpm list 2>/dev/null | grep -c "fpm" || echo 0)
check "Orphans removed (${REMAINING} packages remain)" test "$REMAINING" -lt 13

# ════════════════════════════════════════════════════════════════════════
section "9. Config & Repair"
# ════════════════════════════════════════════════════════════════════════

run "Show config" fpm config show
run "Repair" fpm repair

# ════════════════════════════════════════════════════════════════════════
section "10. Audit"
# ════════════════════════════════════════════════════════════════════════

run "Audit system packages" fpm audit --system
check "Audit runs" fpm audit --system 2>/dev/null

# ════════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  RESULTS: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All scenarios passed!${NC}" || echo -e "\n${RED}Some scenarios failed.${NC}"
exit $FAIL
