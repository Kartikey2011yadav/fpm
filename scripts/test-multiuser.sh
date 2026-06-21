#!/bin/bash
# Multi-user simulation test
# Usage: docker exec fpm-test bash /tmp/test-multiuser.sh

set -o pipefail
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev

GREEN='\033[0;32m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

PASS=0; FAIL=0
check() {
    local d="$1"; shift
    if "$@" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} $d"; PASS=$((PASS+1))
    else
        echo -e "  ${RED}✗${NC} $d"; FAIL=$((FAIL+1))
    fi
}

echo -e "${BOLD}MULTI-USER SHARED SYSTEM SIMULATION${NC}"
echo ""

# Create users
for u in user1 user2 user3; do
    useradd -m -s /bin/bash $u 2>/dev/null || true
done

echo -e "${BOLD}━━━ 1. Per-user project isolation ━━━${NC}"
for u in user1 user2 user3; do
    su -s /bin/bash - $u -c "
export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev
rm -rf ~/proj && mkdir ~/proj && cd ~/proj
fpm init . >/dev/null 2>&1
" 2>/dev/null
done
check "user1 project created" test -d /home/user1/proj/.venv
check "user2 project created" test -d /home/user2/proj/.venv
check "user3 project created" test -d /home/user3/proj/.venv

echo ""
echo -e "${BOLD}━━━ 2. Different packages per user ━━━${NC}"
su -s /bin/bash - user1 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev; cd ~/proj && fpm install requests >/dev/null 2>&1" 2>/dev/null
su -s /bin/bash - user2 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev; cd ~/proj && fpm install flask >/dev/null 2>&1" 2>/dev/null
su -s /bin/bash - user3 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev; cd ~/proj && fpm install six >/dev/null 2>&1" 2>/dev/null

check "user1 has requests" bash -c "su -s /bin/bash - user1 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q requests'" 2>/dev/null
check "user2 has flask" bash -c "su -s /bin/bash - user2 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -qi flask'" 2>/dev/null
check "user3 has six" bash -c "su -s /bin/bash - user3 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q six'" 2>/dev/null

echo ""
echo -e "${BOLD}━━━ 3. No package leakage between users ━━━${NC}"
check "user1 does NOT have flask" bash -c "! su -s /bin/bash - user1 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -qi flask'" 2>/dev/null
check "user2 does NOT have requests" bash -c "! su -s /bin/bash - user2 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q requests'" 2>/dev/null
check "user3 does NOT have flask" bash -c "! su -s /bin/bash - user3 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -qi flask'" 2>/dev/null

echo ""
echo -e "${BOLD}━━━ 4. Shared system packages ━━━${NC}"
# user1 installs to system
su -s /bin/bash - user1 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; fpm install -s chardet >/dev/null 2>&1" 2>/dev/null
# All users should see it
check "user2 sees system chardet" bash -c "su -s /bin/bash - user2 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; fpm list -a --system 2>/dev/null | grep -q chardet'" 2>/dev/null
check "user3 sees system chardet" bash -c "su -s /bin/bash - user3 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; fpm list -a --system 2>/dev/null | grep -q chardet'" 2>/dev/null

echo ""
echo -e "${BOLD}━━━ 5. Per-user snapshots (isolated) ━━━${NC}"
su -s /bin/bash - user1 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm snapshot create 'u1-snap' >/dev/null 2>&1" 2>/dev/null
su -s /bin/bash - user2 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm snapshot create 'u2-snap' >/dev/null 2>&1" 2>/dev/null

check "user1 sees own snap" bash -c "su -s /bin/bash - user1 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm snapshot list 2>&1 | grep -q u1-snap'" 2>/dev/null
check "user1 doesn't see user2 snap" bash -c "! su -s /bin/bash - user1 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm snapshot list 2>&1 | grep -q u2-snap'" 2>/dev/null

echo ""
echo -e "${BOLD}━━━ 6. Concurrent installs (race condition test) ━━━${NC}"
# All 3 users install to their own venvs simultaneously
su -s /bin/bash - user1 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm install chardet >/dev/null 2>&1" 2>/dev/null &
PID1=$!
su -s /bin/bash - user2 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm install chardet >/dev/null 2>&1" 2>/dev/null &
PID2=$!
su -s /bin/bash - user3 -c "export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm install chardet >/dev/null 2>&1" 2>/dev/null &
PID3=$!
wait $PID1 $PID2 $PID3
check "concurrent: user1 has chardet" bash -c "su -s /bin/bash - user1 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q chardet'" 2>/dev/null
check "concurrent: user2 has chardet" bash -c "su -s /bin/bash - user2 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q chardet'" 2>/dev/null
check "concurrent: user3 has chardet" bash -c "su -s /bin/bash - user3 -c 'export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org; cd ~/proj && fpm list 2>/dev/null | grep -q chardet'" 2>/dev/null

echo ""
echo -e "${BOLD}━━━ 7. Per-user CAS caches ━━━${NC}"
U1_CAS=$(su -s /bin/bash - user1 -c "du -sh ~/.cache/fpm 2>/dev/null | cut -f1" 2>/dev/null)
U2_CAS=$(su -s /bin/bash - user2 -c "du -sh ~/.cache/fpm 2>/dev/null | cut -f1" 2>/dev/null)
U3_CAS=$(su -s /bin/bash - user3 -c "du -sh ~/.cache/fpm 2>/dev/null | cut -f1" 2>/dev/null)
echo "  user1 cache: $U1_CAS"
echo "  user2 cache: $U2_CAS"
echo "  user3 cache: $U3_CAS"

echo ""
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BOLD}  RESULTS: ${GREEN}${PASS} passed${NC}, ${RED}${FAIL} failed${NC}"
echo -e "${BOLD}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""
echo "ARCHITECTURE SUMMARY:"
echo "  Per-user isolation:"
echo "    - Each user's ~/.cache/fpm/ is independent"
echo "    - Each user's ~/project/.venv is independent"
echo "    - Each user's snapshots are per-venv"
echo "  Shared resources:"
echo "    - System site-packages (/usr/local/lib/python3.12/site-packages)"
echo "    - Python interpreter (/usr/local/bin/python3)"
echo "  Concurrency:"
echo "    - Parallel installs to separate venvs: SAFE"
echo "    - Parallel installs to system: needs file locking (not yet implemented)"
echo "  Scaling:"
echo "    - No limit on number of users (each has own venv + cache)"
echo "    - CAS deduplication works per-user (not cross-user yet)"
echo "    - System packages shared across all users"

[ "$FAIL" -eq 0 ] && exit 0 || exit $FAIL
