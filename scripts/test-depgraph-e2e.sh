#!/bin/bash
# End-to-end integration tests for dependency graph correctness.
# Tests real-world scenarios where install/remove/purge/snapshot/sync interact.
#
# Requirements: fpm binary in PATH, network access to PyPI
# Usage: ./scripts/test-depgraph-e2e.sh [--insecure]
set -e

INSECURE_FLAGS=""
if [ "$1" = "--insecure" ] || [ "$FPM_INSECURE" = "1" ]; then
    INSECURE_FLAGS="--allow-insecure-host pypi.org --allow-insecure-host files.pythonhosted.org"
fi

FPM="${FPM:-fpm}"
WORKDIR=$(mktemp -d)
PASS=0
FAIL=0
ERRORS=""

cleanup() {
    rm -rf "$WORKDIR"
}
trap cleanup EXIT

GREEN='\033[0;32m'
RED='\033[0;31m'
BOLD='\033[1m'
DIM='\033[2m'
NC='\033[0m'

pass() { PASS=$((PASS+1)); echo -e "  ${GREEN}PASS${NC} $1"; }
fail() { FAIL=$((FAIL+1)); ERRORS="$ERRORS\n  - $1"; echo -e "  ${RED}FAIL${NC} $1"; }

assert_installed() {
    local pkg="$1" dir="$2"
    if $FPM list 2>&1 | grep -qi "$pkg"; then
        return 0
    fi
    return 1
}

assert_not_installed() {
    local pkg="$1"
    if $FPM list 2>&1 | grep -qi "$pkg"; then
        return 1
    fi
    return 0
}

assert_importable() {
    local pkg="$1"
    if $FPM run python -c "import $pkg" 2>&1 | grep -q "Error\|error\|Traceback"; then
        return 1
    fi
    return 0
}

count_packages() {
    $FPM list 2>&1 | grep -c "fpm" || echo "0"
}

echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  FPM Dependency Graph E2E Test Suite${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  ${DIM}Working directory: $WORKDIR${NC}"
echo -e "  ${DIM}FPM binary: $($FPM --version 2>&1)${NC}"
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 1: Install → Remove → Purge (basic lifecycle)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s1 && mkdir s1 && cd s1
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install click $INSECURE_FLAGS >/dev/null 2>&1
if assert_installed "click"; then pass "install click"; else fail "install click"; fi

$FPM remove click >/dev/null 2>&1
if assert_not_installed "click"; then pass "remove click"; else fail "remove click"; fi

$FPM install flask $INSECURE_FLAGS >/dev/null 2>&1
BEFORE=$(count_packages)
$FPM remove flask -p >/dev/null 2>&1
AFTER=$(count_packages)
if [ "$AFTER" = "0" ] || [ "$AFTER" -lt "$BEFORE" ]; then
    pass "remove -p purges deps (was $BEFORE, now $AFTER)"
else
    fail "remove -p should reduce package count (was $BEFORE, now $AFTER)"
fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 2: Shared Dependencies Protection${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s2 && mkdir s2 && cd s2
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install flask requests $INSECURE_FLAGS >/dev/null 2>&1
$FPM install httpx $INSECURE_FLAGS >/dev/null 2>&1

# Remove httpx with purge — flask and requests deps must survive
$FPM remove httpx -p >/dev/null 2>&1

if assert_installed "flask"; then pass "flask survives httpx purge"; else fail "flask survives httpx purge"; fi
if assert_installed "requests"; then pass "requests survives httpx purge"; else fail "requests survives httpx purge"; fi
if assert_importable "flask"; then pass "flask importable after purge"; else fail "flask importable after purge"; fi
if assert_importable "requests"; then pass "requests importable after purge"; else fail "requests importable after purge"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 3: Snapshot → Modify → Restore${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s3 && mkdir s3 && cd s3
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install flask $INSECURE_FLAGS >/dev/null 2>&1
SNAP_ID=$($FPM snapshot create "with flask" 2>&1 | grep -o '[0-9]\{8\}-[0-9]\{6\}-[0-9]\{3\}')

$FPM install httpx $INSECURE_FLAGS >/dev/null 2>&1
if assert_installed "httpx"; then pass "httpx installed after snapshot"; else fail "httpx installed after snapshot"; fi

# Restore to before httpx
$FPM snapshot restore "$SNAP_ID" >/dev/null 2>&1
if assert_installed "flask"; then pass "flask restored from snapshot"; else fail "flask restored from snapshot"; fi
if assert_not_installed "httpx"; then pass "httpx gone after restore"; else fail "httpx gone after restore"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 4: Mark Workflow (protect dep from purge)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s4 && mkdir s4 && cd s4
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install flask $INSECURE_FLAGS >/dev/null 2>&1

# Mark click as requested (was transitive dep of flask)
$FPM mark --requested click >/dev/null 2>&1

# Remove flask with purge — click should survive because it's marked requested
$FPM remove flask -p >/dev/null 2>&1

if assert_installed "click"; then pass "marked click survives flask purge"; else fail "marked click survives flask purge"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 5: Lock & Sync Cycle${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s5 && mkdir s5 && cd s5
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install click $INSECURE_FLAGS >/dev/null 2>&1
$FPM lock $INSECURE_FLAGS >/dev/null 2>&1

if [ -f "fpm.lock" ]; then pass "lockfile created"; else fail "lockfile created"; fi

# Manually remove click from disk (simulate corruption)
rm -rf .venv/lib/*/site-packages/click* 2>/dev/null

# Sync should reinstall
$FPM sync $INSECURE_FLAGS >/dev/null 2>&1
if assert_importable "click"; then pass "sync reinstalled missing click"; else fail "sync reinstalled missing click"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 6: Error Cases (graceful handling)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s6 && mkdir s6 && cd s6
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

# Install nonexistent package
OUTPUT=$($FPM install this-does-not-exist-xyz $INSECURE_FLAGS 2>&1 || true)
if echo "$OUTPUT" | grep -q "not found"; then pass "nonexistent package: clear error"; else fail "nonexistent package: clear error"; fi

# Install typo
OUTPUT=$($FPM install requst $INSECURE_FLAGS 2>&1 || true)
if echo "$OUTPUT" | grep -qi "did you mean"; then pass "typo: suggests correction"; else fail "typo: suggests correction"; fi

# Remove package not installed
OUTPUT=$($FPM remove something-not-here 2>&1 || true)
if echo "$OUTPUT" | grep -qi "not installed\|not found\|removed"; then pass "remove non-installed: graceful"; else fail "remove non-installed: graceful"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 7: Multiple Snapshots (time travel)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s7 && mkdir s7 && cd s7
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install click $INSECURE_FLAGS >/dev/null 2>&1
SNAP1=$($FPM snapshot create "v1-click" 2>&1 | grep -o '[0-9]\{8\}-[0-9]\{6\}-[0-9]\{3\}')

$FPM install httpx $INSECURE_FLAGS >/dev/null 2>&1
SNAP2=$($FPM snapshot create "v2-httpx" 2>&1 | grep -o '[0-9]\{8\}-[0-9]\{6\}-[0-9]\{3\}')

# List shows both
SNAP_COUNT=$($FPM snapshot list 2>&1 | grep -c "[0-9]\{8\}-[0-9]\{6\}")
if [ "$SNAP_COUNT" -ge 2 ]; then pass "multiple snapshots listed"; else fail "multiple snapshots listed ($SNAP_COUNT)"; fi

# Restore to v1 (before httpx)
$FPM snapshot restore "$SNAP1" >/dev/null 2>&1
if assert_installed "click"; then pass "restore v1: click present"; else fail "restore v1: click present"; fi
if assert_not_installed "httpx"; then pass "restore v1: httpx absent"; else fail "restore v1: httpx absent"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 8: Audit (vulnerability scanning)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s8 && mkdir s8 && cd s8
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install click $INSECURE_FLAGS >/dev/null 2>&1
OUTPUT=$($FPM audit $INSECURE_FLAGS 2>&1 || true)
if echo "$OUTPUT" | grep -qi "scan\|audit\|vulnerabilit"; then pass "audit runs successfully"; else fail "audit runs successfully"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 9: Autoremove (orphan cleanup)${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s9 && mkdir s9 && cd s9
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install flask $INSECURE_FLAGS >/dev/null 2>&1
BEFORE=$(count_packages)
$FPM remove flask >/dev/null 2>&1
# Flask's deps are now orphaned
$FPM autoremove >/dev/null 2>&1
AFTER=$(count_packages)
if [ "$AFTER" = "0" ]; then pass "autoremove cleaned all orphans"; else fail "autoremove cleaned all orphans ($AFTER remaining)"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}▶ SCENARIO 10: Cache Operations${NC}"
# ═══════════════════════════════════════════════════════════════════════════════
cd "$WORKDIR" && rm -rf s10 && mkdir s10 && cd s10
$FPM init proj $INSECURE_FLAGS >/dev/null 2>&1 && cd proj

$FPM install click $INSECURE_FLAGS >/dev/null 2>&1
OUTPUT=$($FPM cache gc 2>&1 || true)
if echo "$OUTPUT" | grep -qi "clean\|unreferenced\|No "; then pass "cache gc runs"; else fail "cache gc runs"; fi
echo ""

# ═══════════════════════════════════════════════════════════════════════════════
# RESULTS
# ═══════════════════════════════════════════════════════════════════════════════
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
TOTAL=$((PASS+FAIL))
if [ "$FAIL" -eq 0 ]; then
    echo -e "  ${GREEN}${BOLD}ALL $TOTAL TESTS PASSED${NC}"
else
    echo -e "  ${RED}${BOLD}$FAIL/$TOTAL TESTS FAILED${NC}"
    echo -e "${RED}$ERRORS${NC}"
fi
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
exit $FAIL
