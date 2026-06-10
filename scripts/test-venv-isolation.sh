#!/bin/bash
# fpm Venv/System Isolation Tests
# Tests all permutations of venv active/inactive + --system flag behavior.
# Verifies that packages never leak between venv and system environments.
#
# Usage: docker exec fpm-test bash /workspace/scripts/test-venv-isolation.sh

set -o pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org

PASS=0
FAIL=0

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

check_fail() {
    local desc="$1"
    shift
    if ! "$@" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ ${desc}${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}✗ ${desc} (should have failed but succeeded)${NC}"
        FAIL=$((FAIL + 1))
    fi
}

section() {
    echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

cleanup() {
    unset VIRTUAL_ENV
    cd /tmp
    rm -rf /tmp/iso-test-*
    # Clean system packages installed by fpm for test isolation
    fpm autoremove --system <<< "a" 2>/dev/null || true
    # Remove specific test packages
    fpm remove -fs six 2>/dev/null || true
    fpm remove -fs requests 2>/dev/null || true
    fpm remove -fs flask 2>/dev/null || true
    fpm remove -fs click 2>/dev/null || true
}

cleanup

# ════════════════════════════════════════════════════════════════════════
section "1. INSTALL BOUNDARY: No venv + no --system = ERROR"
# ════════════════════════════════════════════════════════════════════════

cd /tmp
rm -rf /tmp/iso-test-1 && mkdir /tmp/iso-test-1 && cd /tmp/iso-test-1
# No .venv directory here, no VIRTUAL_ENV set

check_fail "Install without venv or --system errors" fpm install requests
check "Error message mentions venv" bash -c 'fpm install requests 2>&1 | grep -qi "virtual environment"'

# ════════════════════════════════════════════════════════════════════════
section "2. INSTALL BOUNDARY: --system flag installs to system"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-1

fpm install -s six >/dev/null 2>&1
check "six installed to system" fpm list -a --system 2>/dev/null | grep -q "six"

# ════════════════════════════════════════════════════════════════════════
section "3. INSTALL BOUNDARY: Inside project dir auto-targets venv"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/iso-test-3 && mkdir /tmp/iso-test-3 && cd /tmp/iso-test-3
fpm init . >/dev/null 2>&1

# Install WITHOUT --system, inside project with .venv — should go to venv
fpm install requests >/dev/null 2>&1
check "requests installed to venv" bash -c 'fpm list 2>/dev/null | grep -q "requests"'

# Verify it did NOT go to system
check "requests NOT in system" bash -c '! fpm list --system 2>/dev/null | grep -q "^requests "'

# ════════════════════════════════════════════════════════════════════════
section "4. INSTALL BOUNDARY: --system inside project targets system (not venv)"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3  # still in the project dir with .venv

fpm install -s click >/dev/null 2>&1
check "click installed to system" fpm list --system 2>/dev/null | grep -q "click"

# Verify it did NOT go to the venv
check "click NOT in venv" bash -c '! fpm list 2>/dev/null | grep -q "^click "'

# ════════════════════════════════════════════════════════════════════════
section "5. REMOVE BOUNDARY: Remove from venv does NOT touch system"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3  # has requests in venv, click in system

# Remove requests (from venv, no --system)
fpm remove requests >/dev/null 2>&1
check "requests removed from venv" bash -c '! fpm list 2>/dev/null | grep -q "requests"'

# System packages unaffected
check "click still in system" fpm list --system 2>/dev/null | grep -q "click"
check "six still in system" fpm list -a --system 2>/dev/null | grep -q "six"

# ════════════════════════════════════════════════════════════════════════
section "6. REMOVE BOUNDARY: Remove --system does NOT touch venv"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3

# Re-install something to venv
fpm install flask >/dev/null 2>&1
check "flask is in venv" fpm list 2>/dev/null | grep -q "flask"

# Remove click from system
fpm remove -s click >/dev/null 2>&1
check "click removed from system" bash -c '! fpm list --system 2>/dev/null | grep -q "^click "'

# Venv unaffected
check "flask still in venv" fpm list 2>/dev/null | grep -q "flask"

# ════════════════════════════════════════════════════════════════════════
section "7. DEACTIVATE BEHAVIOR: After deactivate in project dir, fpm still uses venv"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3  # project dir with .venv

# Simulate deactivate (unset VIRTUAL_ENV)
unset VIRTUAL_ENV

# fpm should still detect .venv by directory
check "fpm detects venv after deactivate (by dir)" fpm list 2>/dev/null | grep -q "flask"

# Install should still go to venv
fpm install requests >/dev/null 2>&1
check "install targets venv even after deactivate" fpm list 2>/dev/null | grep -q "requests"

# ════════════════════════════════════════════════════════════════════════
section "8. OUTSIDE PROJECT: After deactivate + cd away = needs --system"
# ════════════════════════════════════════════════════════════════════════

unset VIRTUAL_ENV
rm -rf /tmp/iso-test-8 && mkdir /tmp/iso-test-8 && cd /tmp/iso-test-8
# No .venv here, no VIRTUAL_ENV

check_fail "Install without venv/system errors" fpm install requests
check_fail "Remove without venv/system errors" fpm remove requests

# ════════════════════════════════════════════════════════════════════════
section "9. PURGE ISOLATION: Purge in venv doesn't affect system"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3  # back in project

# Ensure system has six
fpm install -s six >/dev/null 2>&1

# Purge flask from venv (non-interactive — auto-approve)
echo "a" | fpm remove -p flask >/dev/null 2>&1
check "flask removed from venv" bash -c '! fpm list 2>/dev/null | grep -q "^flask "'

# System packages must be unaffected
check "system six unaffected after venv purge" fpm list -a --system 2>/dev/null | grep -q "six"

# ════════════════════════════════════════════════════════════════════════
section "10. AUTOREMOVE ISOLATION: autoremove in venv doesn't touch system"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3

# Install flask again then remove it (leave deps)
fpm install flask >/dev/null 2>&1
fpm remove flask >/dev/null 2>&1

# Autoremove (venv) — should only clean venv orphans
echo "a" | fpm autoremove >/dev/null 2>&1
check "venv cleaned" bash -c '! fpm list 2>/dev/null | grep -q "werkzeug"'
check "system six still there" fpm list -a --system 2>/dev/null | grep -q "six"

# ════════════════════════════════════════════════════════════════════════
section "11. AUTOREMOVE --system: autoremove --system doesn't touch venv"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3

# Re-install to venv
fpm install flask >/dev/null 2>&1

# Install and remove from system to create system orphans
fpm install -s flask >/dev/null 2>&1
fpm remove -s flask >/dev/null 2>&1

# Autoremove --system
echo "a" | fpm autoremove --system >/dev/null 2>&1

# Venv flask must be unaffected
check "venv flask still present after system autoremove" fpm list 2>/dev/null | grep -q "flask"

# ════════════════════════════════════════════════════════════════════════
section "12. GRAPH ISOLATION: Venv and system graphs are independent"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-3

# Venv has flask, system should have nothing fpm-managed after cleanup
check "venv tree shows flask" fpm tree 2>/dev/null | grep -q "flask"

# System tree should NOT show venv packages
SYSTEM_TREE=$(fpm tree --system 2>&1)
check "system tree does not show venv flask" bash -c "echo '$SYSTEM_TREE' | grep -qv flask || test -z '$SYSTEM_TREE'"

# ════════════════════════════════════════════════════════════════════════
section "13. RECURSIVE PURGE: Sub-dependencies are also cleaned"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/iso-test-13 && mkdir /tmp/iso-test-13 && cd /tmp/iso-test-13
fpm init . >/dev/null 2>&1

fpm install flask >/dev/null 2>&1

# Purge flask — ALL deps should be cleaned (including markupsafe, zipp)
echo -e "a\na" | fpm remove -p flask >/dev/null 2>&1

# Count remaining fpm packages (exclude header line)
REMAINING=$(fpm list 2>/dev/null | grep -v "^Package\|^$\|^[0-9]" | grep -c "fpm" || true)
check "No orphaned sub-deps remain (got ${REMAINING} packages)" test "${REMAINING:-0}" -eq 0

# Specifically check the known sub-deps
check "markupsafe cleaned" bash -c '! fpm list 2>/dev/null | grep -qi "markupsafe"'
check "zipp cleaned" bash -c '! fpm list 2>/dev/null | grep -qi "zipp"'

# ════════════════════════════════════════════════════════════════════════
section "14. CROSS-PROJECT ISOLATION: Two projects don't interfere"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/iso-test-14a /tmp/iso-test-14b
mkdir /tmp/iso-test-14a /tmp/iso-test-14b

cd /tmp/iso-test-14a
fpm init . >/dev/null 2>&1
fpm install requests >/dev/null 2>&1

cd /tmp/iso-test-14b
fpm init . >/dev/null 2>&1
fpm install flask >/dev/null 2>&1

# Project A should NOT see flask
cd /tmp/iso-test-14a
check "project-a has requests" fpm list 2>/dev/null | grep -q "requests"
check "project-a does NOT have flask" bash -c '! fpm list 2>/dev/null | grep -q "^flask "'

# Project B should NOT see requests (unless flask depends on it)
cd /tmp/iso-test-14b
check "project-b has flask" fpm list 2>/dev/null | grep -q "flask"

# Remove in project-b doesn't affect project-a
echo "a" | fpm remove -p flask >/dev/null 2>&1
cd /tmp/iso-test-14a
check "project-a requests intact after project-b remove" fpm list 2>/dev/null | grep -q "requests"

# ════════════════════════════════════════════════════════════════════════
section "15. VIRTUAL_ENV IGNORED: cd out of project loses venv access"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/iso-test-14a  # this has a venv with requests

# Simulate: activate project-a's venv while cd'd elsewhere
export VIRTUAL_ENV="/tmp/iso-test-14a/.venv"

cd /tmp  # cd away from project-a

# fpm should NOT use VIRTUAL_ENV from outside the project directory
check_fail "VIRTUAL_ENV ignored outside project dir" fpm list 2>/dev/null | grep -q "requests"

# But back in the project dir, it works (regardless of VIRTUAL_ENV)
cd /tmp/iso-test-14a
check "In project dir, venv detected by directory" fpm list 2>/dev/null | grep -q "requests"

unset VIRTUAL_ENV

# ════════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  RESULTS: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

cleanup
[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All isolation tests passed!${NC}" || echo -e "\n${RED}Some isolation tests failed.${NC}"
exit $FAIL
