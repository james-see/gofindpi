#!/bin/bash
set -e

echo "Building gofindpi..."

# Build for current platform
go build -o gofindpi -ldflags="-s -w" .

echo "Build complete: ./gofindpi"

# Optional: build for multiple platforms
if [ "$1" = "all" ]; then
    echo "Building for multiple platforms..."
    
    GOOS=darwin GOARCH=amd64 go build -o gofindpi-darwin-amd64 -ldflags="-s -w" .
    GOOS=darwin GOARCH=arm64 go build -o gofindpi-darwin-arm64 -ldflags="-s -w" .
    GOOS=linux GOARCH=amd64 go build -o gofindpi-linux-amd64 -ldflags="-s -w" .
    GOOS=linux GOARCH=arm64 go build -o gofindpi-linux-arm64 -ldflags="-s -w" .
    
    echo "Multi-platform builds complete"
fi