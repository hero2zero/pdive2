# PDive2 Makefile

.PHONY: build clean test run install deps cross-compile help

# Variables
BINARY_NAME=pdive2
BINARY_DIR=bin
GO_FILES=$(shell find . -name "*.go")
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags="-s -w -X main.Version=$(VERSION)"

# Default target
all: build

# Build the binary
build: deps
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME) .
	@echo "Build complete: $(BINARY_DIR)/$(BINARY_NAME)"

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Run the application (example usage)
run: build
	@echo "Running $(BINARY_NAME) with help flag..."
	./$(BINARY_DIR)/$(BINARY_NAME) --help

# Install to GOPATH/bin
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	go install $(LDFLAGS) .

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BINARY_DIR)
	go clean -cache -modcache
	@echo "Clean complete"

# Cross-compile for multiple platforms
cross-compile: deps
	@echo "Cross-compiling for multiple platforms..."
	@mkdir -p $(BINARY_DIR)

	# Linux
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-linux-arm64 .

	# Windows
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe .

	# macOS
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/$(BINARY_NAME)-darwin-arm64 .

	@echo "Cross-compilation complete. Binaries in $(BINARY_DIR)/"
	@ls -la $(BINARY_DIR)/

# Create release archives
release: cross-compile
	@echo "Creating release archives..."
	@mkdir -p releases

	# Linux x64
	tar -czf releases/$(BINARY_NAME)-linux-amd64.tar.gz -C $(BINARY_DIR) $(BINARY_NAME)-linux-amd64

	# Linux ARM64
	tar -czf releases/$(BINARY_NAME)-linux-arm64.tar.gz -C $(BINARY_DIR) $(BINARY_NAME)-linux-arm64

	# Windows x64
	zip releases/$(BINARY_NAME)-windows-amd64.zip $(BINARY_DIR)/$(BINARY_NAME)-windows-amd64.exe

	# macOS x64
	tar -czf releases/$(BINARY_NAME)-darwin-amd64.tar.gz -C $(BINARY_DIR) $(BINARY_NAME)-darwin-amd64

	# macOS ARM64
	tar -czf releases/$(BINARY_NAME)-darwin-arm64.tar.gz -C $(BINARY_DIR) $(BINARY_NAME)-darwin-arm64

	@echo "Release archives created in releases/"
	@ls -la releases/

# Development build with debug info
dev-build:
	@echo "Building development version with debug info..."
	@mkdir -p $(BINARY_DIR)
	go build -gcflags="all=-N -l" -o $(BINARY_DIR)/$(BINARY_NAME)-dev .
	@echo "Development build complete: $(BINARY_DIR)/$(BINARY_NAME)-dev"

# Format Go code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...

# Lint the code (requires golangci-lint)
lint:
	@echo "Linting code..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest" && exit 1)
	golangci-lint run

# Show help
help:
	@echo "PDive2 Build System"
	@echo ""
	@echo "Available targets:"
	@echo "  build          Build the binary (default)"
	@echo "  clean          Clean build artifacts"
	@echo "  test           Run tests"
	@echo "  run            Build and run with --help"
	@echo "  install        Install to GOPATH/bin"
	@echo "  deps           Install/update dependencies"
	@echo "  cross-compile  Build for multiple platforms"
	@echo "  release        Create release archives"
	@echo "  dev-build      Build with debug info"
	@echo "  fmt            Format Go code"
	@echo "  lint           Lint the code"
	@echo "  help           Show this help"
	@echo ""
	@echo "Example usage:"
	@echo "  make build"
	@echo "  make cross-compile"
	@echo "  make release"