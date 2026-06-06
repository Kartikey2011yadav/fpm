#!/bin/bash
# Generate shell completions
set -e

BINARY="${1:-./bin/fpm}"
OUT="${2:-./completions}"

mkdir -p "$OUT"

echo "Generating shell completions..."

$BINARY completion bash > "$OUT/fpm.bash"
$BINARY completion zsh > "$OUT/_fpm"
$BINARY completion fish > "$OUT/fpm.fish"
$BINARY completion powershell > "$OUT/fpm.ps1"

echo "Generated:"
echo "  $OUT/fpm.bash"
echo "  $OUT/_fpm (zsh)"
echo "  $OUT/fpm.fish"
echo "  $OUT/fpm.ps1"
