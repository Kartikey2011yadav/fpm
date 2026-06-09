# Scenario 4: Snapshots & Rollback

## What This Proves

fpm provides git-like history for your package environment. You can snapshot the
current state, experiment freely, then instantly roll back if something breaks.
No other Python package manager offers this.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-snap && mkdir -p /tmp/fpm-demo-snap && cd /tmp/fpm-demo-snap
fpm venv && source .venv/bin/activate
```

---

## Step 1: Install a Baseline Set of Packages

```bash
fpm install requests flask
```

**Expected:** Both packages + dependencies installed.

---

## Step 2: Create a Snapshot (Baseline)

```bash
fpm snapshot create "baseline: requests + flask"
```

**Expected Output:**

```
Snapshot created: 20260609-143000
  Message: baseline: requests + flask
  Packages: 10 (from 1 manager)
```

Note the snapshot ID (timestamp-based). You'll use this later.

---

## Step 3: Install More Packages (Experiment)

```bash
fpm install pandas numpy scipy
```

**Expected:** pandas, numpy, scipy + their dependencies installed. Environment
is now much larger.

---

## Step 4: Create Another Snapshot

```bash
fpm snapshot create "experiment: added data science stack"
```

---

## Step 5: List All Snapshots

```bash
fpm snapshot list
```

**Expected Output:**

```
ID                  Packages  Message
20260609-143000     10        baseline: requests + flask
20260609-143100     25        experiment: added data science stack
```

---

## Step 6: Compare Snapshots (Diff)

```bash
# Diff between baseline and current state
fpm snapshot diff 20260609-143000
```

**Expected Output:**

```
Comparing 20260609-143000 → current:

+ pandas 2.x.x          (fpm)
+ numpy 1.x.x           (fpm)
+ scipy 1.x.x           (fpm)
+ python-dateutil 2.x.x (fpm)
+ pytz 2024.x           (fpm)
+ six 1.x.x             (fpm)
...
```

All additions since the baseline are clearly shown.

---

## Step 7: Diff Between Two Snapshots

```bash
fpm snapshot diff 20260609-143000 20260609-143100
```

**Expected:** Same as above — shows what changed between the two snapshots.

---

## Step 8: Break Something On Purpose

```bash
# Simulate a bad install that causes issues
fpm install "numpy==1.19.0"  # downgrade numpy (might break scipy)
```

Now the environment is in a potentially broken state.

---

## Step 9: Restore to Baseline (Rollback)

```bash
fpm snapshot restore 20260609-143000
```

**Expected Output:**

```
Restoring snapshot 20260609-143000...
  Removing: pandas, numpy, scipy, python-dateutil, pytz, six, ...
  Keeping: requests, flask, werkzeug, jinja2, ...
Restored to: "baseline: requests + flask"
```

---

## Step 10: Verify Rollback

```bash
fpm list
```

**Expected:** Only requests, flask, and their original dependencies. All data
science packages are gone. Environment is exactly as it was at Step 2.

```bash
python -c "import requests; print(requests.__version__)"  # works
python -c "import pandas"  # ImportError — it's gone
```

---

## Step 11: Restore to Experiment State

```bash
fpm snapshot restore 20260609-143100
```

**Expected:** pandas, numpy, scipy are back — environment matches the experiment
snapshot exactly.

---

## Step 12: Delete Old Snapshots

```bash
fpm snapshot delete 20260609-143000
fpm snapshot list
```

**Expected:** Only the experiment snapshot remains.

---

## Advanced: Detecting Drift

If packages are changed outside fpm (e.g., by pip), snapshots detect this:

```bash
fpm snapshot create "before pip meddling"

# Someone uses pip directly
pip install black

# Diff detects the external change
fpm snapshot diff <latest-id>
```

**Expected Output:**

```
Comparing <id> → current:

+ black 24.x.x (pip)     ← DRIFT: installed by external manager
```

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-snap
```

---

## Key Takeaway

> Snapshots are your safety net. Before any risky operation (upgrade,
> experiment, CI change), capture the state. If anything breaks, one command
> rolls you back. This is the "undo" button that Python packaging has always
> been missing.
