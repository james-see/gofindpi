#!/bin/bash
set -e

echo "Running Go linters..."

# Format check
echo "Checking formatting..."
if [ -n "$(gofmt -l .)" ]; then
    echo "Code is not formatted. Run: gofmt -w ."
    gofmt -l .
    exit 1
fi

# Vet
echo "Running go vet..."
go vet ./...

# Static check (if installed)
if command -v staticcheck &> /dev/null; then
    echo "Running staticcheck..."
    staticcheck ./...
else
    echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"
fi

# golangci-lint (if installed)
if command -v golangci-lint &> /dev/null; then
    echo "Running golangci-lint..."
    golangci-lint run
else
    echo "golangci-lint not installed. Install from: https://golangci-lint.run/usage/install/"
fi

echo "Linting complete!"
