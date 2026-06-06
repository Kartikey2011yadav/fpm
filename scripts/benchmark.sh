#!/bin/bash
# Benchmark fpm operations
set -e

BINARY="${1:-./bin/fpm}"
TMPDIR=$(mktemp -d)
trap "rm -rf $TMPDIR" EXIT

echo "Benchmarking fpm..."
echo "Binary: $BINARY"
echo "Temp dir: $TMPDIR"
echo ""

# Benchmark: init
echo "=== fpm init ==="
time $BINARY init "$TMPDIR/bench-project" 2>/dev/null
echo ""

cd "$TMPDIR/bench-project"

# Benchmark: install (cold)
echo "=== fpm install requests (cold) ==="
time $BINARY install requests 2>/dev/null
echo ""

# Benchmark: install (cached)
rm -rf .venv
$BINARY venv 2>/dev/null
echo "=== fpm install requests (cached) ==="
time $BINARY install requests 2>/dev/null
echo ""

# Benchmark: pip list
echo "=== fpm pip list ==="
time $BINARY pip list 2>/dev/null
echo ""

# Benchmark: snapshot
echo "=== fpm snapshot create ==="
time $BINARY snapshot create "benchmark" 2>/dev/null
echo ""

echo "Done."
