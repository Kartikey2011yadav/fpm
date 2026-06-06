#!/bin/bash
# Create a new release
set -e

if [ -z "$1" ]; then
    echo "Usage: ./scripts/release.sh <version>"
    echo "Example: ./scripts/release.sh 0.2.0"
    exit 1
fi

VERSION="$1"
TAG="v${VERSION}"

echo "Releasing fpm ${TAG}..."

# Verify clean working tree
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: working directory is not clean"
    exit 1
fi

# Run tests
echo "Running tests..."
go test ./...

# Build all platforms
echo "Building binaries..."
make build-all

# Tag
echo "Creating tag ${TAG}..."
git tag -a "${TAG}" -m "Release ${TAG}"

echo ""
echo "Release ${TAG} ready."
echo "Push with: git push origin ${TAG}"
echo "Then GoReleaser will handle the GitHub release."
