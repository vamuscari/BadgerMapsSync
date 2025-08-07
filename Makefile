# BadgerSync Makefile
# Build and management commands for the BadgerSync service

# Variables
BINARY_NAME=badgersync
BUILD_DIR=build
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME)"

# Default target
.PHONY: all
all: clean build

# Build the application
.PHONY: build
build:
	@echo "Building BadgerSync..."
	@mkdir -p $(BUILD_DIR)
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) main.go
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for current platform
.PHONY: build-local
build-local:
	@echo "Building BadgerSync for local platform..."
	go build -o $(BINARY_NAME) main.go
	@echo "Build complete: $(BINARY_NAME)"

# Build for multiple platforms
.PHONY: build-all
build-all: clean
	@echo "Building BadgerSync for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 main.go
	
	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe main.go
	GOOS=windows GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe main.go
	
	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 main.go
	
	@echo "Multi-platform build complete in $(BUILD_DIR)/"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@rm -f $(BINARY_NAME)
	@echo "Clean complete"

# Install dependencies
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies installed"

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...
	@echo "Tests complete"

# Run with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -race -v ./...
	@echo "Race detection tests complete"

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Code formatting complete"

# Lint code
.PHONY: lint
lint:
	@echo "Linting code..."
	golangci-lint run
	@echo "Linting complete"

# Run the application
.PHONY: run
run:
	@echo "Running BadgerSync..."
	go run main.go

# Run with specific flags
.PHONY: run-accounts
run-accounts:
	@echo "Running BadgerSync (accounts only)..."
	go run main.go -accounts=true -checkins=false -profiles=false

# Build and run
.PHONY: build-run
build-run: build
	@echo "Running built binary..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Create release package
.PHONY: release
release: build-all
	@echo "Creating release package..."
	@mkdir -p release
	@cp $(BUILD_DIR)/* release/
	@cp README.md release/
	@cp config.example release/
	@echo "Release package created in release/"

# Show help
.PHONY: help
help:
	@echo "BadgerSync Build Commands:"
	@echo "  make build        - Build for current platform"
	@echo "  make build-all    - Build for multiple platforms"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make deps         - Install dependencies"
	@echo "  make test         - Run tests"
	@echo "  make test-race    - Run tests with race detection"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make run          - Run the application"
	@echo "  make run-accounts - Run with accounts sync only"
	@echo "  make build-run    - Build and run"
	@echo "  make release      - Create release package"
	@echo "  make help         - Show this help" 