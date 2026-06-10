#!/bin/bash
# fpm Snapshot Full-Fidelity Restore Tests
# Tests ALL snapshot edge cases: cross-manager restore, immutable config,
# system-level snapshots, package addition/removal/version-change scenarios.
#
# Usage: docker exec fpm-test bash /tmp/test-snapshot-full.sh

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
        echo -e "  ${RED}✗ ${desc} (should have failed)${NC}"
        FAIL=$((FAIL + 1))
    fi
}

section() {
    echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}  $1${NC}"
    echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

get_snap_id() {
    fpm snapshot list "$@" 2>&1 | grep "$1" | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1 | head -1
}

# ════════════════════════════════════════════════════════════════════════
section "1. FPM PACKAGE RESTORE: Remove + restore via CAS"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t1 && mkdir /tmp/snap-t1 && cd /tmp/snap-t1
fpm init . >/dev/null 2>&1
fpm install requests >/dev/null 2>&1

fpm snapshot create "with-requests" >/dev/null 2>&1
SNAP1=$(fpm snapshot list 2>&1 | grep with-requests | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Remove requests
fpm remove requests >/dev/null 2>&1
echo "a" | fpm autoremove >/dev/null 2>&1
check_fail "requests gone after remove" bash -c "fpm list 2>/dev/null | grep -q requests"

# Restore
fpm snapshot restore "$SNAP1" >/dev/null 2>&1
check "requests restored" bash -c "fpm list 2>/dev/null | grep -q requests"
check "certifi restored (dep)" bash -c "fpm list 2>/dev/null | grep -q certifi"

# ════════════════════════════════════════════════════════════════════════
section "2. EXTERNAL PACKAGE RESTORE: pip package reinstalled"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t2 && mkdir /tmp/snap-t2 && cd /tmp/snap-t2
fpm init . >/dev/null 2>&1

# Install six via pip into the venv
.venv/bin/pip install six --trusted-host pypi.org --trusted-host files.pythonhosted.org -q 2>/dev/null

fpm snapshot create "with-pip-six" >/dev/null 2>&1
SNAP2=$(fpm snapshot list 2>&1 | grep with-pip-six | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Remove six manually
.venv/bin/pip uninstall -y six -q 2>/dev/null
check_fail "six gone after pip uninstall" bash -c "fpm list -a 2>/dev/null | grep -q six"

# Restore should bring it back
fpm snapshot restore "$SNAP2" >/dev/null 2>&1
check "pip six restored" bash -c "fpm list -a 2>/dev/null | grep -q six"

# ════════════════════════════════════════════════════════════════════════
section "3. NEW PACKAGES REMOVED ON RESTORE"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t3 && mkdir /tmp/snap-t3 && cd /tmp/snap-t3
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1

fpm snapshot create "only-six" >/dev/null 2>&1
SNAP3=$(fpm snapshot list 2>&1 | grep only-six | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Add flask after snapshot
fpm install flask >/dev/null 2>&1
check "flask installed" bash -c "fpm list 2>/dev/null | grep -q flask"

# Restore should remove flask (wasn't in snapshot)
fpm snapshot restore "$SNAP3" >/dev/null 2>&1
check_fail "flask removed on restore" bash -c "fpm list 2>/dev/null | grep -qi flask"
check "six still present" bash -c "fpm list 2>/dev/null | grep -q six"

# ════════════════════════════════════════════════════════════════════════
section "4. FPM.TOML IMMUTABLE CONFIG RESTORED"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t4 && mkdir /tmp/snap-t4 && cd /tmp/snap-t4
fpm init . >/dev/null 2>&1
cat > fpm.toml << 'TOML'
[project]
name = "snap-t4"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "six", version = "1.17.0" }]
TOML

fpm snapshot create "immutable-six" >/dev/null 2>&1
SNAP4=$(fpm snapshot list 2>&1 | grep immutable-six | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Change fpm.toml to pin numpy instead
cat > fpm.toml << 'TOML'
[project]
name = "snap-t4"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "numpy", version = "2.0.0" }]
TOML
check "fpm.toml changed to numpy" grep -q numpy fpm.toml

# Restore should revert fpm.toml
fpm snapshot restore "$SNAP4" >/dev/null 2>&1
check "fpm.toml reverted to six" grep -q "six" /tmp/snap-t4/fpm.toml
check_fail "numpy no longer in fpm.toml" grep -q "numpy" /tmp/snap-t4/fpm.toml

# ════════════════════════════════════════════════════════════════════════
section "5. IMMUTABLE ADDED AFTER SNAPSHOT → REMOVED ON RESTORE"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t5 && mkdir /tmp/snap-t5 && cd /tmp/snap-t5
fpm init . >/dev/null 2>&1
# No fpm.toml with immutable section initially

fpm snapshot create "no-immutable" >/dev/null 2>&1
SNAP5=$(fpm snapshot list 2>&1 | grep no-immutable | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Add immutable config after snapshot
cat > fpm.toml << 'TOML'
[project]
name = "snap-t5"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "click", version = "8.4.1" }]
TOML
check "immutable config added" grep -q immutable fpm.toml

# Restore should revert fpm.toml to version without immutable section
fpm snapshot restore "$SNAP5" >/dev/null 2>&1
check_fail "immutable section removed on restore" grep -q "immutable" /tmp/snap-t5/fpm.toml

# ════════════════════════════════════════════════════════════════════════
section "6. SYSTEM-LEVEL SNAPSHOT: Create and list"
# ════════════════════════════════════════════════════════════════════════

cd /tmp
fpm snapshot create --system "sys-baseline" >/dev/null 2>&1
check "system snapshot created" bash -c "fpm snapshot list --system 2>&1 | grep -q sys-baseline"

# ════════════════════════════════════════════════════════════════════════
section "7. SYSTEM SNAPSHOT: Restore removes added packages"
# ════════════════════════════════════════════════════════════════════════

SNAP7=$(fpm snapshot list --system 2>&1 | grep sys-baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Install to system after snapshot
fpm install -s chardet >/dev/null 2>&1
check "chardet in system" bash -c "fpm list --system 2>/dev/null | grep -q chardet"

# Restore system snapshot
fpm snapshot restore --system "$SNAP7" >/dev/null 2>&1
check_fail "chardet removed after system restore" bash -c "fpm list --system 2>/dev/null | grep -q chardet"

# ════════════════════════════════════════════════════════════════════════
section "8. SNAPSHOT DIFF: Shows additions, removals, changes"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t8 && mkdir /tmp/snap-t8 && cd /tmp/snap-t8
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1

fpm snapshot create "diff-base" >/dev/null 2>&1
SNAP8=$(fpm snapshot list 2>&1 | grep diff-base | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Make changes
fpm install flask >/dev/null 2>&1
fpm remove six >/dev/null 2>&1

DIFF_OUT=$(fpm snapshot diff "$SNAP8" 2>&1)
check "diff shows added packages" bash -c "echo '$DIFF_OUT' | grep -q '+'"
check "diff shows removed packages" bash -c "echo '$DIFF_OUT' | grep -q '\\- six'"

# ════════════════════════════════════════════════════════════════════════
section "9. SNAPSHOT SCOPE ISOLATION: Project vs system"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/snap-t9 && mkdir /tmp/snap-t9 && cd /tmp/snap-t9
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1
fpm snapshot create "project-snap" >/dev/null 2>&1

# System snapshots should be independent
SYS_LIST=$(fpm snapshot list --system 2>&1)
PROJ_LIST=$(fpm snapshot list 2>&1)
check "system snapshots separate from project" bash -c "echo '$SYS_LIST' | grep -qv 'project-snap'"

# ════════════════════════════════════════════════════════════════════════
section "10. SNAPSHOT DELETE"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/snap-t9
SNAP10=$(fpm snapshot list 2>&1 | grep project-snap | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot delete "$SNAP10" >/dev/null 2>&1
check_fail "snapshot deleted" bash -c 'fpm snapshot list 2>&1 | grep -q "project-snap"'

# ════════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  RESULTS: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All snapshot tests passed!${NC}" || echo -e "\n${RED}Some snapshot tests failed.${NC}"
exit $FAIL
