.PHONY: build clean test lint run install docker-build docker-run help

# Build configuration
BINARY_NAME=gofindpi
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

## help: Display this help message
help:
	@echo "Available targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) .
	@echo "Build complete: ./$(BINARY_NAME)"

## build-all: Build for all platforms
build-all:
	@echo "Building for multiple platforms..."
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	@GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .
	@GOOS=linux GOARCH=arm go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm .
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .
	@echo "Multi-platform builds complete"

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	@rm -f $(BINARY_NAME) $(BINARY_NAME)-*
	@rm -f ~/devicesfound.txt ~/pilist.txt
	@echo "Clean complete"

## test: Run tests
test:
	@echo "Running tests..."
	@go test -v -race -coverprofile=coverage.out ./...

## lint: Run linters
lint:
	@echo "Running linters..."
	@go fmt ./...
	@go vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed, skipping..."; \
	fi

## run: Build and run the scanner
run: build
	@./$(BINARY_NAME)

## install: Install the binary to GOPATH/bin
install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) .
	@echo "Installed to $(shell go env GOPATH)/bin/$(BINARY_NAME)"

## deps: Download and tidy dependencies
deps:
	@echo "Updating dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

## docker-build: Build Docker image
docker-build:
	@echo "Building Docker image..."
	@docker-compose build
	@echo "Docker image built"

## docker-run: Run in Docker container
docker-run: docker-build
	@docker-compose run --rm gofindpi

## mod-upgrade: Upgrade all dependencies
mod-upgrade:
	@echo "Upgrading dependencies..."
	@go get -u ./...
	@go mod tidy
	@echo "Dependencies upgraded"

## update-oui: Update the OUI database from external sources
update-oui:
	@echo "Updating OUI database..."
	@go run scripts/generate_oui.go
	@echo "OUI database updated"

## oui-stats: Show OUI database statistics
oui-stats:
	@echo "OUI Database Statistics:"
	@wc -l data/oui.go | awk '{print "  Lines:", $$1}'
	@grep -c "^	\"" data/oui.go | awk '{print "  Entries:", $$1}'
