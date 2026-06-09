# Scenario 2: Dependency Graph & Intelligent Cleanup

## What This Proves

fpm tracks which packages you explicitly installed ("requested") vs which came
along as dependencies ("transitive"). This enables intelligent cleanup —
removing a package also removes its orphaned dependencies, just like apt/yum do
on Linux.

pip and uv leave orphaned dependencies forever. fpm doesn't.

---

## Setup

```bash
rm -rf /tmp/fpm-demo-deps && mkdir -p /tmp/fpm-demo-deps && cd /tmp/fpm-demo-deps
fpm venv && source .venv/bin/activate
```

---

## Step 1: Install Flask (Pulls Multiple Dependencies)

```bash
fpm install flask
```

**Expected Output:**

```
Resolving dependencies...
  + flask 3.x.x
  + werkzeug 3.x.x
  + jinja2 3.x.x
  + click 8.x.x
  + markupsafe 2.x.x
  + itsdangerous 2.x.x
  + blinker 1.x.x
```

---

## Step 2: View the Dependency Tree

```bash
fpm tree
```

**Expected Output:**

```
flask 3.x.x
├── werkzeug 3.x.x
├── jinja2 3.x.x
│   └── markupsafe 2.x.x
├── click 8.x.x
├── itsdangerous 2.x.x
└── blinker 1.x.x
```

---

## Step 3: Check Package Status (Requested vs Dependency)

```bash
fpm mark --show flask werkzeug jinja2
```

**Expected Output:**

```
flask        requested   (you installed this)
werkzeug     dependency  (pulled by flask)
jinja2       dependency  (pulled by flask)
```

---

## Step 4: Install Another Package That Shares Dependencies

```bash
fpm install jinja2
```

Now jinja2 is **also** a requested package (not just a dependency of flask).

```bash
fpm mark --show jinja2
```

**Expected:** `jinja2  requested` — it's now explicitly requested.

---

## Step 5: Remove Flask (Simple Remove)

```bash
fpm remove flask
```

**Expected:** Only flask is removed. Dependencies remain because they might
still be needed.

```bash
fpm list
```

**Expected:** werkzeug, jinja2, click, markupsafe, itsdangerous, blinker still
present.

---

## Step 6: Autoremove Orphaned Dependencies

```bash
fpm autoremove
```

**Expected Output:**

```
Removing orphaned packages:
  - werkzeug 3.x.x
  - click 8.x.x
  - itsdangerous 2.x.x
  - blinker 1.x.x
Kept (still needed):
  - jinja2 3.x.x (requested)
  - markupsafe 2.x.x (dependency of jinja2)
```

**Key insight:** jinja2 stays because we explicitly installed it in Step 4.
markupsafe stays because jinja2 still needs it.

---

## Step 7: Purge (Remove + Autoremove in One Step)

```bash
# First, reinstall flask
fpm install flask

# Now purge it — removes flask AND its orphaned deps in one step
fpm remove --purge flask
```

**Expected:** flask + all dependencies that nothing else needs are removed.
jinja2 and markupsafe stay (jinja2 is still explicitly requested).

---

## Step 8: Reverse Dependency Tree

```bash
# Reinstall flask for this demo
fpm install flask

fpm tree --invert
```

**Expected Output:**

```
markupsafe 2.x.x
└── (required by) jinja2 3.x.x
    └── (required by) flask 3.x.x

click 8.x.x
└── (required by) flask 3.x.x
...
```

This shows "who depends on me" — essential for understanding impact of removing
a package.

---

## Step 9: Manually Reclassify a Package

```bash
# Mark click as explicitly requested (protect it from autoremove)
fpm mark --requested click

# Now remove flask and autoremove
fpm remove flask
fpm autoremove
```

**Expected:** click stays (it's now "requested"), while werkzeug, itsdangerous,
blinker are removed.

---

## Cleanup

```bash
deactivate
rm -rf /tmp/fpm-demo-deps
```

---

## Key Takeaway

> fpm maintains a full dependency graph (like apt/yum) that tracks why each
> package exists. This means `autoremove` is safe, `purge` is smart, and you
> never accumulate package cruft. pip/uv have no equivalent — once you install
> something, its dependencies live forever.
