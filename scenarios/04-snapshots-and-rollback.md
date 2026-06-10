# Scenario 4: Snapshots & Rollback

## What This Proves

fpm provides git-like history for your package environment. You can snapshot the
current state, experiment freely, then instantly roll back if something breaks.
Snapshots capture EVERYTHING: all packages from all managers, plus your fpm.toml
config (immutable pins). No other Python package manager offers this.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-snap && mkdir -p /tmp/fpm-demo-snap && cd /tmp/fpm-demo-snap
fpm init .
```

> No activation needed — just `cd` into the project directory.

---

## Step 1: Install a Baseline Set of Packages

```bash
fpm install requests flask
```

**Expected:** Both packages + dependencies installed into `.venv`.

---

## Step 2: Configure Immutable Pins

```bash
cat > fpm.toml << 'EOF'
[project]
name = "snap-demo"
requires-python = ">=3.10"
dependencies = ["requests", "flask"]
[immutable]
packages = [{ name = "requests", version = "2.34.2" }]
EOF
```

This locks requests at exactly 2.34.2 — no one can change it.

---

## Step 3: Create a Snapshot (Baseline)

```bash
fpm snapshot create "baseline: requests + flask + immutable config"
```

**Expected Output:**

```
Snapshot created: 20260610-143000-001
  Packages: 11 total (11 fpm, 0 pip, 0 other)
  Message: baseline: requests + flask + immutable config
```

**What gets captured:**

- All packages from all managers (fpm, pip, uv, conda)
- Exact versions
- `fpm.toml` content (including immutable pins)
- Python version info

---

## Step 4: Install More Packages (Experiment)

```bash
fpm install pandas numpy
```

**Expected:** pandas, numpy + their dependencies installed. Environment grows.

---

## Step 5: Create Another Snapshot

```bash
fpm snapshot create "experiment: added data science stack"
```

---

## Step 6: List All Snapshots

```bash
fpm snapshot list
```

**Expected Output:**

```
Environment snapshots:

* 20260610-143100-001  2026-06-10 14:31  [20 packages]  experiment: added data science stack
  20260610-143000-001  2026-06-10 14:30  [11 packages]  baseline: requests + flask + immutable config
```

---

## Step 7: Compare Snapshots (Diff)

```bash
fpm snapshot diff 20260610-143000-001
```

**Expected Output:**

```
Diff: 20260610-143000-001 → current

  + pandas 2.2.0 (fpm)
  + numpy 1.24.0 (fpm)
  + python-dateutil 2.9.0 (fpm)
  + six 1.17.0 (fpm)
  ...
```

---

## Step 8: Break Something — Change Immutable Config

```bash
# Modify fpm.toml to pin something different
cat > fpm.toml << 'EOF'
[project]
name = "snap-demo"
requires-python = ">=3.10"
dependencies = ["requests", "flask"]
[immutable]
packages = [{ name = "numpy", version = "2.0.0" }]
EOF
```

Now your immutable policy has changed from protecting requests to protecting
numpy.

---

## Step 9: Restore to Baseline (Full Rollback)

```bash
fpm snapshot restore 20260610-143000-001
```

**Expected Output:**

```
Restoring snapshot 20260610-143000-001 (2026-06-10 14:30)...
  ✓ Restored 11 fpm-managed packages from cache

Environment restored to snapshot 20260610-143000-001.
```

**What restore does:**

- Removes fpm packages not in the snapshot (pandas, numpy gone)
- Re-links fpm packages from CAS (instant, no download needed)
- Reinstalls any missing pip/uv/conda packages
- Reverts `fpm.toml` to the snapshot state (requests immutable pin restored)

---

## Step 10: Verify Rollback

```bash
# Check packages
fpm list
# Expected: Only requests, flask, and their deps. No pandas/numpy.

# Check immutable config was restored
cat fpm.toml | grep -A2 immutable
# Expected: requests pinned at 2.34.2 (not numpy)

# Verify immutable enforcement works
fpm install requests==3.0.0
# Expected: error: pinned as immutable at 2.34.2
```

---

## Step 11: Restore to Experiment State

```bash
fpm snapshot restore 20260610-143100-001
```

**Expected:** pandas, numpy are back. Environment matches the experiment
snapshot exactly.

---

## Step 12: System-Level Snapshots

Snapshots also work at system level with `--system`:

```bash
cd /tmp  # Leave project directory

# Snapshot system Python packages
fpm snapshot create --system "system baseline"
fpm install -s chardet
fpm snapshot list --system
fpm snapshot restore --system <snap-id>
# chardet is removed — system back to baseline
```

---

## Step 13: Detecting Drift (Cross-Manager Changes)

If packages are changed outside fpm (e.g., by pip), snapshots detect and
restore:

```bash
cd /tmp/fpm-demo-snap
fpm snapshot create "before pip meddling"

# Someone uses pip directly
pip install black

# Diff detects the external change
fpm snapshot diff <latest-id>
```

**Expected:**

```
Diff: <id> → current

  + black 24.x.x (pip)     ← installed by external manager
```

On restore, black will be removed (it wasn't in the snapshot).

---

## Step 14: Delete Old Snapshots

```bash
fpm snapshot delete 20260610-143000-001
fpm snapshot list
```

**Expected:** Only remaining snapshots are shown.

---

## Cleanup

```bash
rm -rf /tmp/fpm-demo-snap
```

---

## Key Takeaway

> Snapshots are your full environment safety net. They capture packages from ALL
> managers plus your fpm.toml config. Before any risky operation, take a
> snapshot. If anything breaks — packages, versions, or even your immutable
> config — one command rolls everything back to the exact state you captured.
