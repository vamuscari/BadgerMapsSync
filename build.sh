#!/bin/bash

# BadgerSync Build Script for Unix/Linux/macOS
# Build and management commands for the BadgerSync service

BINARY_NAME="badgersync"
BUILD_DIR="build"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Function to show help
show_help() {
    echo "BadgerSync Build Commands:"
    echo "  ./build.sh         - Build for current platform"
    echo "  ./build.sh clean   - Clean build artifacts"
    echo "  ./build.sh deps    - Install dependencies"
    echo "  ./build.sh test    - Run tests"
    echo "  ./build.sh fmt     - Format code"
    echo "  ./build.sh lint    - Lint code"
    echo "  ./build.sh run     - Run the application"
    echo "  ./build.sh build-all - Build for multiple platforms"
    echo "  ./build.sh release - Create release package"
    echo "  ./build.sh help    - Show this help"
}

# Function to build for current platform
build_local() {
    echo "Building BadgerSync for current platform..."
    mkdir -p "$BUILD_DIR"
    go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/$BINARY_NAME" main.go
    echo "Build complete: $BUILD_DIR/$BINARY_NAME"
}

# Function to clean build artifacts
clean() {
    echo "Cleaning build artifacts..."
    rm -rf "$BUILD_DIR"
    rm -f "$BINARY_NAME"
    echo "Clean complete"
}

# Function to install dependencies
deps() {
    echo "Installing dependencies..."
    go mod download
    go mod tidy
    echo "Dependencies installed"
}

# Function to run tests
test() {
    echo "Running tests..."
    go test -v ./...
    echo "Tests complete"
}

# Function to format code
fmt() {
    echo "Formatting code..."
    go fmt ./...
    echo "Code formatting complete"
}

# Function to lint code
lint() {
    echo "Linting code..."
    if command -v golangci-lint &> /dev/null; then
        golangci-lint run
    else
        echo "golangci-lint not found. Install it with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"
    fi
    echo "Linting complete"
}

# Function to run the application
run() {
    echo "Running BadgerSync..."
    go run main.go
}

# Function to build for multiple platforms
build_all() {
    echo "Building BadgerSync for multiple platforms..."
    clean
    mkdir -p "$BUILD_DIR"
    
    # Linux
    GOOS=linux GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-linux-amd64" main.go
    GOOS=linux GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-linux-arm64" main.go
    
    # Windows
    GOOS=windows GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-windows-amd64.exe" main.go
    GOOS=windows GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-windows-arm64.exe" main.go
    
    # macOS
    GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-darwin-amd64" main.go
    GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME" -o "$BUILD_DIR/${BINARY_NAME}-darwin-arm64" main.go
    
    echo "Multi-platform build complete in $BUILD_DIR/"
}

# Function to create release package
release() {
    echo "Creating release package..."
    build_all
    mkdir -p release
    cp "$BUILD_DIR"/* release/
    cp README.md release/
    cp config.example release/
    echo "Release package created in release/"
}

# Main script logic
case "${1:-build}" in
    "clean")
        clean
        ;;
    "deps")
        deps
        ;;
    "test")
        test
        ;;
    "fmt")
        fmt
        ;;
    "lint")
        lint
        ;;
    "run")
        run
        ;;
    "build-all")
        build_all
        ;;
    "release")
        release
        ;;
    "help"|"-h"|"--help")
        show_help
        ;;
    "build"|"")
        build_local
        ;;
    *)
        echo "Unknown command: $1"
        show_help
        exit 1
        ;;
esac 