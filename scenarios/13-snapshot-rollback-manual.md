# Scenario 13: Snapshot & Rollback — Comprehensive Manual Test

## What This Proves

fpm's snapshot system handles every real-world rollback scenario: project-level,
system-level, config changes, multi-user environments, and edge cases. This
document is a step-by-step manual test guide.

---

## Prerequisites

```bash
# Fresh Docker container with multiple users
docker run -d --name fpm-test \
  -e FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org,api.osv.dev \
  python:3.12-slim sleep infinity

# Install fpm
docker cp bin/fpm-linux fpm-test:/usr/local/bin/fpm

# Create test users
docker exec fpm-test bash -c '
  useradd -m -s /bin/bash alice
  useradd -m -s /bin/bash bob
  mkdir -p /root/.config/pip
  echo -e "[global]\ntrusted-host = pypi.org\n               files.pythonhosted.org" > /root/.config/pip/pip.conf
'
```

---

## Part 1: Project-Level Rollback

### Test 1.1: Multi-Point Time Travel

```bash
cd /tmp && rm -rf proj1 && mkdir proj1 && cd proj1
fpm init .

# Create 3 states
fpm install six
fpm snapshot create "state-1: just six"

fpm install requests
fpm snapshot create "state-2: + requests"

fpm install flask
fpm snapshot create "state-3: + flask"

# List all snapshots
fpm snapshot list
```

**Expected:** 3 snapshots with increasing package counts.

```bash
# Jump back to state 1
SNAP1=$(fpm snapshot list | grep "state-1" | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP1
fpm list
```

**Expected:** Only six present. requests and flask gone.

```bash
# Jump forward to state 3
SNAP3=$(fpm snapshot list | grep "state-3" | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP3
fpm list
```

**Expected:** six, requests, flask all present.

---

### Test 1.2: Packages Added After Snapshot Get Removed

```bash
cd /tmp && rm -rf proj2 && mkdir proj2 && cd proj2
fpm init .
fpm install six
fpm snapshot create "baseline"

# Add more packages AFTER the snapshot
fpm install flask requests
fpm list   # shows six + flask + requests + deps

# Restore removes everything not in snapshot
SNAP=$(fpm snapshot list | grep baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
fpm list
```

**Expected:** Only six remains. flask, requests, and all their deps are removed.

---

### Test 1.3: Removed Packages Get Restored

```bash
cd /tmp && rm -rf proj3 && mkdir proj3 && cd proj3
fpm init .
fpm install requests flask
fpm snapshot create "full"

# Remove everything
echo "a" | fpm remove -p requests
echo "a" | fpm remove -p flask
echo "a" | fpm autoremove
fpm list   # should be empty

# Restore brings everything back
SNAP=$(fpm snapshot list | grep full | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
fpm list
```

**Expected:** All packages restored (requests, flask + all deps).

---

## Part 2: Configuration Rollback

### Test 2.1: Immutable Config Captured and Restored

```bash
cd /tmp && rm -rf proj4 && mkdir proj4 && cd proj4
fpm init .
fpm install requests

# Set up immutable config
cat > fpm.toml << 'EOF'
[project]
name = "test"
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
EOF

fpm snapshot create "with-immutable"

# Change the config
cat > fpm.toml << 'EOF'
[project]
name = "test"
[immutable]
packages = [{ name = "numpy", version = "2.0.0" }]
EOF

# Verify immutable was changed
cat fpm.toml | grep -A1 "packages"

# Restore
SNAP=$(fpm snapshot list | grep with-immutable | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
cat fpm.toml | grep -A1 "packages"
```

**Expected:** After restore, fpm.toml shows `requests` pinned (not numpy).

---

### Test 2.2: Immutable Added After Snapshot Gets Removed

```bash
cd /tmp && rm -rf proj5 && mkdir proj5 && cd proj5
fpm init .
fpm install six
fpm snapshot create "no-config"

# Add immutable config AFTER snapshot
cat > fpm.toml << 'EOF'
[project]
name = "test"
[immutable]
packages = [{ name = "six", version = "1.17.0" }]
EOF

# Restore should remove fpm.toml (didn't exist at snapshot time)
SNAP=$(fpm snapshot list | grep no-config | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
ls fpm.toml 2>&1
```

**Expected:** `fpm.toml` no longer exists (or has no immutable section).

---

### Test 2.3: Verify Immutable Enforcement After Restore

```bash
cd /tmp/proj4
# After restoring the "with-immutable" snapshot (from test 2.1)
fpm install requests==3.0.0
```

**Expected:** Error — `pinned as immutable at version 2.34.2 in fpm.toml`

---

## Part 3: System-Level Rollback

### Test 3.1: System Snapshot Create and Restore

```bash
cd /tmp

# Clean system state
fpm snapshot create --system "clean-system"

# Install to system
fpm install -s chardet
fpm install -s six
fpm list --system   # shows chardet + six

# Rollback
SNAP=$(fpm snapshot list --system | grep clean-system | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore --system $SNAP
fpm list --system
```

**Expected:** chardet and six are removed from system.

---

### Test 3.2: System Snapshot Doesn't Affect Projects

```bash
cd /tmp && rm -rf proj6 && mkdir proj6 && cd proj6
fpm init .
fpm install requests
fpm install -s chardet

# Take system snapshot
cd /tmp
fpm snapshot create --system "with-chardet"

# Restore system to clean (removes chardet)
fpm snapshot restore --system $SNAP   # use SNAP from test 3.1

# Project should be unaffected
cd /tmp/proj6
fpm list
```

**Expected:** Project still has requests. System chardet is gone.

---

### Test 3.3: System Diff

```bash
cd /tmp
fpm install -s six
fpm install -s chardet

SNAP=$(fpm snapshot list --system | grep clean | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot diff --system $SNAP
```

**Expected:** Shows `+ six` and `+ chardet` as additions since snapshot.

---

## Part 4: Cross-Scope Isolation

### Test 4.1: Project Restore Doesn't Touch System

```bash
cd /tmp && rm -rf proj7 && mkdir proj7 && cd proj7
fpm init .
fpm install flask

# Also have something in system
fpm install -s six

# Snapshot project
fpm snapshot create "proj-baseline"

# Add more to project
fpm install requests

# Restore project
SNAP=$(fpm snapshot list | grep proj-baseline | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP

# Verify
fpm list            # flask only (requests removed)
fpm list --system   # six still there
```

**Expected:** System packages untouched by project restore.

---

### Test 4.2: System Restore Doesn't Touch Projects

```bash
cd /tmp
fpm snapshot create --system "sys-base"
fpm install -s chardet

# Restore system
SNAP=$(fpm snapshot list --system | grep sys-base | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore --system $SNAP

# Project should be untouched
cd /tmp/proj7
fpm list   # still has flask
```

---

## Part 5: Multi-User Scenarios

### Test 5.1: Per-User Project Isolation

```bash
# Alice creates her project
docker exec -u alice fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  rm -rf ~/proj && mkdir ~/proj && cd ~/proj
  fpm init . && fpm install requests
  fpm snapshot create "alice-v1"
  fpm list
'

# Bob creates his project
docker exec -u bob fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  rm -rf ~/proj && mkdir ~/proj && cd ~/proj
  fpm init . && fpm install flask
  fpm snapshot create "bob-v1"
  fpm list
'
```

**Expected:** Alice has requests. Bob has flask. No leakage.

---

### Test 5.2: Per-User Snapshot Isolation

```bash
# Alice's snapshots
docker exec -u alice fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm snapshot list
'

# Bob's snapshots
docker exec -u bob fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm snapshot list
'
```

**Expected:** Alice sees "alice-v1", Bob sees "bob-v1". They don't see each
other's snapshots.

---

### Test 5.3: Shared System Packages

```bash
# Alice installs to system
docker exec -u root fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  fpm install -s chardet
'

# Bob can see it
docker exec -u bob fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  fpm list -a --system | grep chardet
'
```

**Expected:** System packages visible to all users.

---

### Test 5.4: Concurrent Installs

```bash
# Alice and Bob install different packages simultaneously
docker exec -u alice fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm install chardet
' &
docker exec -u bob fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm install chardet
' &
wait

# Both should succeed without corruption
docker exec -u alice fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm list | grep chardet
'
docker exec -u bob fpm-test bash -c '
  export FPM_ALLOW_INSECURE_HOST=pypi.org,files.pythonhosted.org
  cd ~/proj && fpm list | grep chardet
'
```

**Expected:** Both users have chardet. No file corruption.

---

## Part 6: Edge Cases

### Test 6.1: Restore to Empty State

```bash
cd /tmp && rm -rf proj8 && mkdir proj8 && cd proj8
fpm init .
fpm snapshot create "empty"

fpm install flask requests pandas
fpm list   # lots of packages

SNAP=$(fpm snapshot list | grep empty | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
fpm list
```

**Expected:** Zero fpm packages after restore (back to empty venv).

---

### Test 6.2: Snapshot With pip-Installed Packages

```bash
cd /tmp && rm -rf proj9 && mkdir proj9 && cd proj9
fpm init .
.venv/bin/python3 -m ensurepip 2>/dev/null
.venv/bin/python3 -m pip install chardet -q

fpm list -a   # shows chardet (pip manager)
fpm snapshot create "with-pip-pkg"

.venv/bin/python3 -m pip uninstall -y chardet -q
fpm list -a   # chardet gone

SNAP=$(fpm snapshot list | grep with-pip | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
fpm list -a   # chardet back
```

**Expected:** pip-installed package restored via `pip install --target`.

---

### Test 6.3: Multiple Restores in Sequence

```bash
cd /tmp && rm -rf proj10 && mkdir proj10 && cd proj10
fpm init .
fpm install six && fpm snapshot create "v1"
fpm install requests && fpm snapshot create "v2"
fpm install flask && fpm snapshot create "v3"

# Rapid restore sequence
for label in v1 v3 v2 v1 v3; do
  SNAP=$(fpm snapshot list | grep "$label" | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
  fpm snapshot restore $SNAP >/dev/null 2>&1
done

# Final state should be v3
fpm list
```

**Expected:** After restoring v1→v3→v2→v1→v3, final state matches v3 (flask
present).

---

### Test 6.4: Delete Snapshot Then Try Restore

```bash
cd /tmp/proj10
SNAP=$(fpm snapshot list | grep v2 | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot delete $SNAP
fpm snapshot restore $SNAP
```

**Expected:** Error — snapshot not found.

---

### Test 6.5: Snapshot List Shows Correct Order

```bash
cd /tmp/proj10
fpm snapshot list
```

**Expected:** Newest first. v3 at top, v1 at bottom. v2 gone (deleted).

---

## Part 7: Using `--mutable` with Snapshots

### Test 7.1: Pin Status Visible Before/After Restore

```bash
cd /tmp && rm -rf proj11 && mkdir proj11 && cd proj11
fpm init .
fpm install requests flask

# Pin requests
cat > fpm.toml << 'EOF'
[project]
name = "test"
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
EOF

fpm list --mutable   # requests shows 🔒, flask shows mutable

fpm snapshot create "pinned"

# Remove pin
rm fpm.toml
fpm list --mutable   # all show mutable

# Restore
SNAP=$(fpm snapshot list | grep pinned | grep -oE "[0-9]{8}-[0-9]{6}-[0-9]+" | head -1)
fpm snapshot restore $SNAP
fpm list --mutable   # requests shows 🔒 again
```

**Expected:** `--mutable` flag correctly reflects fpm.toml state after restore.

---

## Cleanup

```bash
docker stop fpm-test && docker rm fpm-test
```

---

## Key Takeaway

> fpm's snapshot system provides complete environment versioning. You can jump
> to any point in history — packages, config, and policies all roll back
> together. Multi-user systems are safe thanks to file locking and per-user
> isolation. The only shared state is system site-packages, which requires
> explicit `--system`.
