#!/bin/bash
# fpm Snapshot Complex Scenario Tests
# Comprehensive edge-case testing for the snapshot system covering:
# - Cross-manager restore (fpm + pip packages)
# - Immutable config capture and restore
# - System conflict detection
# - Project-level override strategy
# - Mutable/immutable listing
# - Snapshot scope isolation
#
# Usage: docker exec fpm-test bash /tmp/test-snapshot-scenarios.sh

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
    local desc="$1"; shift
    if "$@" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ ${desc}${NC}"
        PASS=$((PASS + 1))
    else
        echo -e "  ${RED}✗ ${desc}${NC}"
        FAIL=$((FAIL + 1))
    fi
}

check_fail() {
    local desc="$1"; shift
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

# Clean all previous state
rm -rf /tmp/ss-* /root/.cache/fpm/snapshots

# ════════════════════════════════════════════════════════════════════════
section "1. BASIC: Create → Modify → Restore fpm packages"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-1 && mkdir /tmp/ss-1 && cd /tmp/ss-1
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1
fpm install requests >/dev/null 2>&1

fpm snapshot create "baseline" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Remove both packages
echo "a" | fpm remove -p requests >/dev/null 2>&1
fpm remove six >/dev/null 2>&1
echo "a" | fpm autoremove >/dev/null 2>&1
check_fail "packages removed" bash -c "fpm list 2>/dev/null | grep -q requests"

# Restore
fpm snapshot restore "$SNAP" >/dev/null 2>&1
check "requests restored" bash -c "fpm list 2>/dev/null | grep -q requests"
check "six restored" bash -c "fpm list 2>/dev/null | grep -q six"
check "certifi restored (dep)" bash -c "fpm list 2>/dev/null | grep -q certifi"

# ════════════════════════════════════════════════════════════════════════
section "2. CROSS-MANAGER: Snapshot captures pip, restore brings it back"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-2 && mkdir /tmp/ss-2 && cd /tmp/ss-2
fpm init . >/dev/null 2>&1

# Install pip into venv, then install chardet via pip
.venv/bin/python3 -m ensurepip >/dev/null 2>&1
.venv/bin/python3 -m pip install chardet --trusted-host pypi.org --trusted-host files.pythonhosted.org -q 2>/dev/null
check "chardet installed via pip" bash -c "fpm list -a 2>/dev/null | grep -qi chardet"

fpm snapshot create "with-pip-chardet" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep with-pip | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Remove chardet
.venv/bin/python3 -m pip uninstall -y chardet -q 2>/dev/null
check_fail "chardet removed" bash -c "fpm list -a 2>/dev/null | grep -qi chardet"

# Restore
fpm snapshot restore "$SNAP" >/dev/null 2>&1
check "chardet restored from pip" bash -c "fpm list -a 2>/dev/null | grep -qi chardet"

# ════════════════════════════════════════════════════════════════════════
section "3. NEW PACKAGES CLEANED: Packages added after snapshot get removed"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-3 && mkdir /tmp/ss-3 && cd /tmp/ss-3
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1

fpm snapshot create "only-six" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep only-six | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Add more packages
fpm install flask >/dev/null 2>&1
check "flask added" bash -c "fpm list 2>/dev/null | grep -qi flask"

# Restore removes flask
fpm snapshot restore "$SNAP" >/dev/null 2>&1
check_fail "flask removed (not in snapshot)" bash -c "fpm list 2>/dev/null | grep -qi flask"
check "six still present" bash -c "fpm list 2>/dev/null | grep -q six"

# ════════════════════════════════════════════════════════════════════════
section "4. IMMUTABLE CONFIG: fpm.toml restored to snapshot state"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-4 && mkdir /tmp/ss-4 && cd /tmp/ss-4
fpm init . >/dev/null 2>&1
cat > fpm.toml << 'TOML'
[project]
name = "ss-4"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
TOML

fpm snapshot create "immutable-requests" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep immutable-requests | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Change immutable config
cat > fpm.toml << 'TOML'
[project]
name = "ss-4"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "numpy", version = "2.0.0" }]
TOML
check "fpm.toml changed" grep -q numpy /tmp/ss-4/fpm.toml

# Restore reverts fpm.toml
fpm snapshot restore "$SNAP" >/dev/null 2>&1
check "fpm.toml restored to requests pin" grep -q "requests" /tmp/ss-4/fpm.toml
check_fail "numpy pin removed" grep -q "numpy" /tmp/ss-4/fpm.toml

# ════════════════════════════════════════════════════════════════════════
section "5. IMMUTABLE ADDED AFTER SNAPSHOT: Gets removed on restore"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-5 && mkdir /tmp/ss-5 && cd /tmp/ss-5
fpm init . >/dev/null 2>&1

fpm snapshot create "no-immutable" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep no-immutable | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Add immutable after snapshot
cat > fpm.toml << 'TOML'
[project]
name = "ss-5"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "click", version = "8.4.1" }]
TOML
check "immutable added" grep -q immutable /tmp/ss-5/fpm.toml

# Restore removes immutable section
fpm snapshot restore "$SNAP" >/dev/null 2>&1
check_fail "immutable section gone after restore" grep -q "immutable" /tmp/ss-5/fpm.toml

# ════════════════════════════════════════════════════════════════════════
section "6. MUTABLE FLAG: Shows pinned status"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-6 && mkdir /tmp/ss-6 && cd /tmp/ss-6
fpm init . >/dev/null 2>&1
fpm install requests >/dev/null 2>&1
cat > fpm.toml << 'TOML'
[project]
name = "ss-6"
requires-python = ">=3.10"
dependencies = ["requests"]
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
TOML

LIST_OUT=$(fpm list -m 2>&1)
check "mutable flag shows Pinned column" bash -c "echo '$LIST_OUT' | grep -q Pinned"
check "requests shows lock icon" bash -c "echo '$LIST_OUT' | grep -q '🔒'"
check "certifi shows mutable" bash -c "echo '$LIST_OUT' | grep certifi | grep -q mutable"

# ════════════════════════════════════════════════════════════════════════
section "7. SYSTEM SNAPSHOT: Create and list with --system"
# ════════════════════════════════════════════════════════════════════════

cd /tmp
fpm snapshot create --system "system-base" >/dev/null 2>&1
check "system snapshot created" bash -c "fpm snapshot list --system 2>&1 | grep -q system-base"

# Project snapshots don't show system ones
rm -rf /tmp/ss-7 && mkdir /tmp/ss-7 && cd /tmp/ss-7
fpm init . >/dev/null 2>&1
fpm snapshot create "project-only" >/dev/null 2>&1
check_fail "project list doesn't show system snap" bash -c "fpm snapshot list 2>&1 | grep -q system-base"
check "project list shows own snap" bash -c "fpm snapshot list 2>&1 | grep -q project-only"

# ════════════════════════════════════════════════════════════════════════
section "8. SNAPSHOT DIFF: Detects additions, removals, version changes"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-8 && mkdir /tmp/ss-8 && cd /tmp/ss-8
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1

fpm snapshot create "diff-baseline" >/dev/null 2>&1
SNAP=$(fpm snapshot list 2>&1 | grep diff-baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# Make changes
fpm install flask >/dev/null 2>&1
fpm remove six >/dev/null 2>&1

DIFF=$(fpm snapshot diff "$SNAP" 2>&1)
check "diff shows additions (+)" bash -c "echo '$DIFF' | grep -q '+'"
check "diff shows removals (-)" bash -c "echo '$DIFF' | grep -q '\\-'"

# ════════════════════════════════════════════════════════════════════════
section "9. SNAPSHOT DELETE: Removes snapshot"
# ════════════════════════════════════════════════════════════════════════

cd /tmp/ss-8
fpm snapshot delete "$SNAP" >/dev/null 2>&1
check_fail "snapshot no longer in list" bash -c "fpm snapshot list 2>&1 | grep -q diff-baseline"

# ════════════════════════════════════════════════════════════════════════
section "10. MULTIPLE SNAPSHOTS: Restore to any point in history"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-10 && mkdir /tmp/ss-10 && cd /tmp/ss-10
fpm init . >/dev/null 2>&1

# State 1: just six
fpm install six >/dev/null 2>&1
fpm snapshot create "state-1-six" >/dev/null 2>&1
SNAP1=$(fpm snapshot list 2>&1 | grep state-1-six | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# State 2: six + requests
fpm install requests >/dev/null 2>&1
fpm snapshot create "state-2-requests" >/dev/null 2>&1
SNAP2=$(fpm snapshot list 2>&1 | grep state-2-requests | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)

# State 3: six + requests + flask
fpm install flask >/dev/null 2>&1
fpm snapshot create "state-3-flask" >/dev/null 2>&1

# Restore to state 1 (just six)
fpm snapshot restore "$SNAP1" >/dev/null 2>&1
check "restore to state-1: six present" bash -c "fpm list 2>/dev/null | grep -q six"
check_fail "restore to state-1: requests gone" bash -c "fpm list 2>/dev/null | grep -q requests"
check_fail "restore to state-1: flask gone" bash -c "fpm list 2>/dev/null | grep -qi flask"

# Restore to state 2 (six + requests)
fpm snapshot restore "$SNAP2" >/dev/null 2>&1
check "restore to state-2: requests present" bash -c "fpm list 2>/dev/null | grep -q requests"
check_fail "restore to state-2: flask gone" bash -c "fpm list 2>/dev/null | grep -qi flask"

# ════════════════════════════════════════════════════════════════════════
section "11. CROSS-PROJECT ISOLATION: Snapshots don't leak between projects"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-11a /tmp/ss-11b
mkdir /tmp/ss-11a /tmp/ss-11b

cd /tmp/ss-11a
fpm init . >/dev/null 2>&1
fpm install six >/dev/null 2>&1
fpm snapshot create "project-a" >/dev/null 2>&1

cd /tmp/ss-11b
fpm init . >/dev/null 2>&1
fpm install requests >/dev/null 2>&1
fpm snapshot create "project-b" >/dev/null 2>&1

# Each project only sees its own snapshots
cd /tmp/ss-11a
check "project-a sees own snapshot" bash -c "fpm snapshot list 2>&1 | grep -q project-a"
check_fail "project-a doesn't see project-b" bash -c "fpm snapshot list 2>&1 | grep -q project-b"

cd /tmp/ss-11b
check "project-b sees own snapshot" bash -c "fpm snapshot list 2>&1 | grep -q project-b"
check_fail "project-b doesn't see project-a" bash -c "fpm snapshot list 2>&1 | grep -q project-a"

# ════════════════════════════════════════════════════════════════════════
section "12. OUTSIDE PROJECT: Snapshot commands require --system or project"
# ════════════════════════════════════════════════════════════════════════

rm -rf /tmp/ss-12 && mkdir /tmp/ss-12 && cd /tmp/ss-12
# No .venv here

check_fail "snapshot create errors without project" fpm snapshot create "test"
check_fail "snapshot list errors without project" fpm snapshot list
check "snapshot create --system works anywhere" bash -c "fpm snapshot create --system 'anywhere' 2>/dev/null"
check "snapshot list --system works anywhere" bash -c "fpm snapshot list --system 2>&1 | grep -q anywhere"

# ════════════════════════════════════════════════════════════════════════
echo -e "\n${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  RESULTS: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All snapshot scenario tests passed!${NC}" || echo -e "\n${RED}Some tests failed.${NC}"
exit $FAIL
