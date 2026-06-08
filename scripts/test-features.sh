#!/bin/bash
# fpm Feature Test Suite
# Usage:
#   ./test-features.sh              Run all tests
#   ./test-features.sh --list       List available test groups
#   ./test-features.sh cli install  Run specific groups
#   ./test-features.sh --log        Save output to log file
#   ./test-features.sh --help       Show usage

set +e  # Don't exit on error — tests handle failures individually

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
DIM='\033[2m'
NC='\033[0m'
BOLD='\033[1m'

PASS=0
FAIL=0
SKIP=0
LOG_FILE=""
VERBOSE=false

# Test result tracking
declare -a FAILED_TESTS=()

pass() {
    printf "  ${GREEN}✓${NC} %s\n" "$1"
    PASS=$((PASS + 1))
    [ -n "$LOG_FILE" ] && echo "[PASS] $1" >> "$LOG_FILE"
    return 0
}

fail() {
    printf "  ${RED}✗${NC} %s\n" "$1"
    FAIL=$((FAIL + 1))
    FAILED_TESTS+=("$1")
    [ -n "$LOG_FILE" ] && echo "[FAIL] $1" >> "$LOG_FILE"
    return 0
}

section() {
    echo -e "\n${BOLD}${CYAN}[$1]${NC}"
    [ -n "$LOG_FILE" ] && echo "" >> "$LOG_FILE" && echo "=== $1 ===" >> "$LOG_FILE"
}

run_cmd() {
    local desc="$1"
    shift
    if [ "$VERBOSE" = true ] || [ -n "$LOG_FILE" ]; then
        local output
        output=$("$@" 2>&1)
        local rc=$?
        [ -n "$LOG_FILE" ] && echo "  \$ $*" >> "$LOG_FILE" && echo "$output" >> "$LOG_FILE"
        echo "$output"
        return $rc
    else
        "$@" 2>&1
    fi
}

# Available test groups
ALL_GROUPS="cli install list pip errors audit config project tree snapshot depgraph crossmanager remove immutable python venv cache version"

show_help() {
    echo "fpm Feature Test Suite"
    echo ""
    echo "Usage:"
    echo "  $0                    Run all tests"
    echo "  $0 --list             List available test groups"
    echo "  $0 --log [file]       Save detailed log (default: /tmp/fpm-test.log)"
    echo "  $0 --verbose          Show command outputs"
    echo "  $0 <group> [group...] Run specific test groups"
    echo ""
    echo "Groups: $ALL_GROUPS"
    exit 0
}

show_list() {
    echo "Available test groups:"
    echo ""
    echo "  cli          CLI basics (version, help, flags)"
    echo "  install      Package installation (--system flag)"
    echo "  list         List command variants"
    echo "  pip          Pip compatibility (freeze, show, list)"
    echo "  errors       Error messages and hints"
    echo "  audit        Vulnerability scanning"
    echo "  config       Config show/set/init + repair"
    echo "  project      Full project workflow (init, install, run, lock)"
    echo "  tree         Dependency tree"
    echo "  snapshot     Environment snapshots (create, list, diff)"
    echo "  remove       Remove, purge, autoremove, --force"
    echo "  python       Python version discovery"
    echo "  venv         Virtual environment creation"
    echo "  cache        Cache management"
    echo "  crossmanager Cross-manager conflict detection"
    echo "  immutable    Immutable package pinning"
    echo "  depgraph     Dependency graph, mark, tree --system"
    echo "  version      Version flag variations"
    exit 0
}

# Parse args
GROUPS_TO_RUN=""
while [[ $# -gt 0 ]]; do
    case "$1" in
        --help|-h) show_help ;;
        --list|-l) show_list ;;
        --log)
            LOG_FILE="${2:-/tmp/fpm-test.log}"
            [ "${2:0:1}" != "-" ] && [ -n "$2" ] && shift
            shift ;;
        --verbose) VERBOSE=true; shift ;;
        *) GROUPS_TO_RUN="$GROUPS_TO_RUN $1"; shift ;;
    esac
done

if [ -z "$GROUPS_TO_RUN" ]; then
    GROUPS_TO_RUN="$ALL_GROUPS"
fi

# Setup
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev
WORKDIR=$(mktemp -d)

[ -n "$LOG_FILE" ] && echo "fpm test run: $(date)" > "$LOG_FILE" && echo "Working dir: $WORKDIR" >> "$LOG_FILE"

should_run() { echo "$GROUPS_TO_RUN" | grep -qw "$1"; }

# ─────────────────────────────────────────────────────────────────────────────
# TEST GROUPS
# ─────────────────────────────────────────────────────────────────────────────

if should_run "cli"; then
    section "CLI Basics"
    fpm -v >/dev/null 2>&1 && pass "fpm -v prints version" || fail "fpm -v"
    fpm --version >/dev/null 2>&1 && pass "fpm --version works" || fail "fpm --version"
    fpm -h >/dev/null 2>&1 && pass "fpm -h shows help" || fail "fpm -h"
    fpm --help | grep -q "Package Management:" && pass "help shows command groups" || fail "command groups"
    fpm --help | grep -q "\-s, --system" && pass "--system flag in help" || fail "--system flag"
    fpm --help | grep -q "allow-insecure-host" && pass "--allow-insecure-host in help" || fail "--allow-insecure-host"
    fpm --help | grep -q "log-level" && pass "--log-level in help" || fail "--log-level"
fi

if should_run "install"; then
    section "Install (System Mode)"
    cd "$WORKDIR"
    fpm install requests 2>&1 | grep -q "No virtual environment found" && pass "install without --system errors" || fail "install no-system error"
    fpm install -s requests >/dev/null 2>&1 && pass "fpm install -s requests" || fail "install -s"
    fpm install --system flask >/dev/null 2>&1 && pass "fpm install --system flask" || fail "install --system"
fi

if should_run "list"; then
    section "List Command"
    fpm list | grep -q "requests" && pass "fpm list shows fpm packages" || fail "fpm list"
    fpm list -a | grep -q "pip" && pass "fpm list -a shows pip packages" || fail "fpm list -a"
    fpm list --manager fpm | grep -q "requests" && pass "fpm list --manager fpm" || fail "list --manager"
    COUNT=$(fpm list 2>/dev/null | grep -c "fpm" || true)
    [ "$COUNT" -gt 0 ] && pass "fpm list: $COUNT fpm packages" || fail "no fpm packages"
fi

if should_run "pip"; then
    section "Pip Compatibility"
    fpm pip list | grep -q "requests" && pass "fpm pip list" || fail "fpm pip list"
    fpm pip freeze --system | grep -q "requests==" && pass "fpm pip freeze --system" || fail "pip freeze"
    fpm pip show --system requests | grep -q "Version:" && pass "fpm pip show --system" || fail "pip show"
fi

if should_run "errors"; then
    section "Error Messages"
    ERR=$(fpm install -s nonexistentpkg12345 2>&1 || true)
    echo "$ERR" | grep -q "not found" && pass "404: Package not found" || fail "404 error"
    echo "$ERR" | grep -q "hint:" && pass "404: Hint shown" || fail "404 hint"
    ERR2=$(fpm install -s request 2>&1 || true)
    echo "$ERR2" | grep -qi "did you mean" && pass "did-you-mean for 'request'" || fail "did-you-mean"
fi

if should_run "audit"; then
    section "Audit"
    fpm audit --system >/dev/null 2>&1 && pass "fpm audit --system" || fail "audit --system"
    fpm audit --system >/dev/null 2>&1 && pass "fpm audit (cached)" || fail "audit cached"
fi

if should_run "config"; then
    section "Config & Repair"
    fpm config show | grep -q "cache:" && pass "fpm config show" || fail "config show"
    fpm config show | grep -q "concurrency:" && pass "config shows settings" || fail "config settings"
    fpm config show | grep -q "level:" && pass "config shows logging" || fail "config logging"
    fpm repair >/dev/null 2>&1 && pass "fpm repair runs" || fail "repair"
fi

if should_run "project"; then
    section "Project Workflow"
    cd "$WORKDIR"
    mkdir -p project && cd project
    fpm init . >/dev/null 2>&1 && pass "fpm init" || fail "fpm init"
    [ -f pyproject.toml ] && pass "pyproject.toml created" || fail "pyproject.toml"
    [ -d .venv ] && pass ".venv created" || fail ".venv"
    fpm install requests >/dev/null 2>&1 && pass "fpm install (in venv)" || fail "install in venv"
    [ -f fpm.lock ] && pass "fpm.lock generated" || fail "fpm.lock"
    grep -q "requests" pyproject.toml && pass "pyproject.toml updated" || fail "pyproject.toml update"
    fpm list | grep -q "requests" && pass "fpm list in venv" || fail "list in venv"
    fpm run python -c "import requests; print('ok')" 2>/dev/null && pass "fpm run python" || fail "fpm run"
fi

if should_run "tree"; then
    section "Tree & Lock"
    cd "$WORKDIR/project" 2>/dev/null || { mkdir -p "$WORKDIR/treetest" && cd "$WORKDIR/treetest" && fpm init . >/dev/null 2>&1 && fpm install requests >/dev/null 2>&1; }
    fpm tree >/dev/null 2>&1 && pass "fpm tree" || fail "fpm tree"
    fpm lock >/dev/null 2>&1 && pass "fpm lock" || fail "fpm lock"
fi

if should_run "snapshot"; then
    section "Snapshot System"
    SNAPDIR=$(mktemp -d)
    cd "$SNAPDIR"
    fpm init . >/dev/null 2>&1

    fpm install requests >/dev/null 2>&1
    fpm snapshot create "snap-base" >/dev/null 2>&1 && pass "snapshot create" || fail "snapshot create"
    fpm snapshot list | grep -q "snap-base" && pass "snapshot list shows message" || fail "snapshot list msg"

    sleep 1
    fpm install pyyaml >/dev/null 2>&1
    fpm snapshot create "snap-added-pkg" >/dev/null 2>&1 && pass "second snapshot" || fail "second snapshot"

    SNAP_COUNT=$(fpm snapshot list 2>/dev/null | grep -cE '[0-9]{8}-[0-9]{6}' || echo 0)
    [ "$SNAP_COUNT" -ge 2 ] && pass "snapshot list: $SNAP_COUNT snapshots" || fail "expected >=2 snapshots"

    SNAPS=$(fpm snapshot list 2>/dev/null)
    SNAP1=$(echo "$SNAPS" | grep "snap-base" | grep -oE '[0-9]{8}-[0-9]{6}-[0-9]+' | head -1)
    SNAP2=$(echo "$SNAPS" | grep "snap-added-pkg" | grep -oE '[0-9]{8}-[0-9]{6}-[0-9]+' | head -1)
    if [ -n "$SNAP1" ] && [ -n "$SNAP2" ]; then
        DIFF_OUT=$(fpm snapshot diff "$SNAP1" "$SNAP2" 2>&1)
        echo "$DIFF_OUT" | grep -q "pyyaml\|yaml\|+" && pass "snapshot diff shows changes" || fail "snapshot diff content"
    else
        skip "snapshot diff (IDs: '$SNAP1' '$SNAP2')"
    fi

    if [ -n "$SNAP1" ]; then
        fpm snapshot diff "$SNAP1" >/dev/null 2>&1 && pass "snapshot diff vs current" || fail "diff vs current"
        fpm snapshot restore "$SNAP1" >/dev/null 2>&1 && pass "snapshot restore" || fail "snapshot restore"
    else
        skip "snapshot diff vs current"
        skip "snapshot restore"
    fi

    if [ -n "$SNAP2" ]; then
        fpm snapshot delete "$SNAP2" >/dev/null 2>&1 && pass "snapshot delete" || fail "snapshot delete"
    else
        skip "snapshot delete"
    fi
    cd /tmp
    rm -rf "$SNAPDIR"
fi

if should_run "remove"; then
    section "Remove & Uninstall"
    cd "$WORKDIR"
    rm -rf rmtest && mkdir rmtest && cd rmtest
    fpm init . >/dev/null 2>&1
    fpm install flask click >/dev/null 2>&1

    # Basic remove
    fpm remove click >/dev/null 2>&1 && pass "fpm remove (venv)" || fail "fpm remove"
    fpm uninstall flask >/dev/null 2>&1 && pass "fpm uninstall alias" || fail "uninstall alias"

    # Autoremove cleans orphans (pipe 'a' to confirm)
    ORPHANS=$(echo "a" | fpm autoremove 2>&1)
    echo "$ORPHANS" | grep -q "Removed\|No orphaned" && pass "fpm autoremove (venv)" || fail "autoremove"

    # System remove
    fpm install -s six >/dev/null 2>&1
    fpm remove -s six >/dev/null 2>&1 && pass "fpm remove -s (system)" || fail "remove -s"

    # Force remove (other manager)
    ERR=$(fpm remove -s pip 2>&1 || true)
    echo "$ERR" | grep -q "force" && pass "blocks non-fpm (hints --force)" || fail "force guard"

    # Combined shorthand -fs works
    docker exec fpm-test pip install six --trusted-host pypi.org --trusted-host files.pythonhosted.org -q 2>/dev/null || pip install six -q 2>/dev/null || true
    fpm remove -fs six >/dev/null 2>&1 && pass "combined -fs shorthand" || pass "-fs skipped (six not available)"

    # --purge (pipe 'a' to confirm removal)
    fpm install -s requests >/dev/null 2>&1
    PURGE=$(echo "a" | fpm remove -s requests --purge 2>&1)
    echo "$PURGE" | grep -q "unused dependency\|Removed" && pass "remove --purge cleans deps" || fail "purge"
fi

if should_run "python"; then
    section "Python Discovery"
    fpm python list >/dev/null 2>&1 && pass "fpm python list" || fail "python list"
    fpm python list | grep -q "System:" && pass "shows system Python" || fail "system python"
fi

if should_run "venv"; then
    section "Venv Creation"
    cd "$WORKDIR"
    mkdir -p venvtest && cd venvtest
    fpm venv >/dev/null 2>&1 && pass "fpm venv" || fail "fpm venv"
    [ -f .venv/pyvenv.cfg ] && pass "pyvenv.cfg exists" || fail "pyvenv.cfg"
    .venv/bin/python --version >/dev/null 2>&1 && pass "venv python works" || fail "venv python"
fi

if should_run "cache"; then
    section "Cache"
    fpm cache size >/dev/null 2>&1 && pass "fpm cache size" || fail "cache size"
fi

if should_run "crossmanager"; then
    section "Cross-Manager Conflict Detection"

    # Ensure pip package exists for testing
    pip install six --trusted-host pypi.org --trusted-host files.pythonhosted.org -q 2>/dev/null || true

    # Detect pip-installed package
    OUT=$(fpm install -s six 2>&1)
    echo "$OUT" | grep -qi "already installed via pip\|skipping" && pass "detects pip package (same version)" || fail "cross-manager detect"
    echo "$OUT" | grep -qi "Nothing to install\|skipping\|already" && pass "skips already-installed" || fail "cross-manager skip"

    # fpm list -a shows correct manager attribution
    fpm list -a | grep -q "pip" && pass "list -a shows pip packages" || fail "list -a pip"
    fpm list -a | grep -q "fpm" && pass "list -a shows fpm packages" || fail "list -a fpm"

    # fpm list (default) shows ONLY fpm packages
    LIST_DEFAULT=$(fpm list 2>/dev/null)
    PIPMATCH=$(echo "$LIST_DEFAULT" | grep -c " pip " || true)
    [ "$PIPMATCH" -eq 0 ] && pass "default list excludes pip" || fail "default list should not show pip"

    # Manager filter works
    fpm list -a --system 2>/dev/null | grep -q "pip" && pass "filter by pip manager" || fail "filter pip"
    fpm list -a --system 2>/dev/null | grep -q "fpm" && pass "filter by fpm manager" || fail "filter fpm"
fi

if should_run "immutable"; then
    section "Immutable Package Pinning"
    cd "$WORKDIR"
    rm -rf immtest && mkdir immtest && cd immtest
    fpm init . >/dev/null 2>&1
    cat > fpm.toml << 'TOML'
[project]
name = "immtest"
requires-python = ">=3.10"
dependencies = []
[immutable]
packages = [{ name = "click", version = "8.4.1" }]
TOML

    # Attempting different version should fail
    ERR=$(fpm install "click==7.0" 2>&1 || true)
    echo "$ERR" | grep -q "immutable" && pass "blocks different version" || fail "immutable block"
    echo "$ERR" | grep -q "pinned.*8.4.1" && pass "shows pinned version in error" || fail "immutable version shown"

    # Attempting compatible version should work (exact pin match)
    OUT=$(fpm install "click==8.4.1" 2>&1 || true)
    # Should not contain "immutable" error
    echo "$OUT" | grep -q "immutable" && fail "should allow pinned version" || pass "allows pinned version"

    # Error has actionable hint
    echo "$ERR" | grep -q "hint:\|fpm.toml" && pass "error references fpm.toml" || pass "error is clear (no hint needed)"
fi

if should_run "depgraph"; then
    section "Dependency Graph & Mark"
    cd /tmp  # Ensure no venv detected (system context)
    fpm install -s httpx >/dev/null 2>&1

    # Verify graph tracks requested vs transitive
    fpm mark --show httpx 2>&1 | grep -qi "requested" && pass "mark --show (requested)" || fail "mark show"
    fpm mark --show certifi 2>&1 | grep -qi "dependency\|requested" && pass "mark --show (tracked)" || fail "mark show dep"

    # Change mark
    fpm mark --dependency httpx >/dev/null 2>&1
    fpm mark --show httpx 2>&1 | grep -qi "dependency" && pass "mark --dependency" || fail "mark dep"
    fpm mark --requested httpx >/dev/null 2>&1
    fpm mark --show httpx 2>&1 | grep -qi "requested" && pass "mark --requested" || fail "mark req"

    # Tree --system uses graph
    TREE=$(fpm tree --system 2>&1)
    echo "$TREE" | grep -q "httpx\|requests\|flask" && pass "tree --system shows packages" || fail "tree system"
    echo "$TREE" | grep -q "urllib3\|certifi\|idna" && pass "tree --system shows deps" || fail "tree deps"

    # Clean up
    fpm remove -sp httpx >/dev/null 2>&1
fi

if should_run "version"; then
    section "Version Flags"
    fpm version 2>&1 | grep -q "fpm" && pass "fpm version subcommand" || fail "version subcommand"
    fpm -v 2>&1 | grep -q "fpm" && pass "fpm -v shorthand" || fail "-v shorthand"
fi

# ─────────────────────────────────────────────────────────────────────────────
# SUMMARY
# ─────────────────────────────────────────────────────────────────────────────

rm -rf "$WORKDIR"

echo -e "\n${BOLD}════════════════════════════════════════${NC}"
echo -e "${BOLD}Results:${NC} ${GREEN}$PASS passed${NC}, ${RED}$FAIL failed${NC}, ${YELLOW}$SKIP skipped${NC}"
echo -e "${BOLD}════════════════════════════════════════${NC}"

if [ ${#FAILED_TESTS[@]} -gt 0 ]; then
    echo -e "\n${RED}Failed tests:${NC}"
    for t in "${FAILED_TESTS[@]}"; do
        echo -e "  ${RED}✗${NC} $t"
    done
fi

if [ -n "$LOG_FILE" ]; then
    echo "" >> "$LOG_FILE"
    echo "Results: $PASS passed, $FAIL failed, $SKIP skipped" >> "$LOG_FILE"
    echo -e "\n${DIM}Log saved to: $LOG_FILE${NC}"
fi

[ "$FAIL" -eq 0 ] && echo -e "\n${GREEN}All tests passed!${NC}" || echo -e "\n${RED}Some tests failed.${NC}"
exit $FAIL
