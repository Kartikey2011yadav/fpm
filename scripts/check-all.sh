#!/bin/bash
# Run all checks (vet, build, test)
set -e

echo "=== go vet ==="
go vet ./...

echo "=== go build ==="
go build ./...

echo "=== go test ==="
go test ./... -count=1

echo ""
echo "All checks passed."
