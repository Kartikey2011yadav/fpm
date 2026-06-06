#!/bin/bash
# Development environment setup
set -e

echo "Setting up fpm development environment..."

# Check Go
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Get it from https://go.dev/dl/"
    exit 1
fi

echo "Go: $(go version)"

# Install dependencies
echo "Downloading dependencies..."
go mod download

# Build
echo "Building fpm..."
make build

# Run tests
echo "Running tests..."
go test ./... -short

echo ""
echo "Done! Binary at ./bin/fpm"
echo "Run: ./bin/fpm version"
