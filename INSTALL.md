# PDive2 Installation Guide

This guide covers different installation methods for PDive2, from simple binary downloads to building from source.

## Quick Install (Recommended)

### Option 1: Download Pre-built Binary

**Note**: Pre-compiled binaries will be available once the project is published to GitHub releases.

```bash
# This will be available after first release:
# curl -L -o pdive2 https://github.com/your-org/pdive2/releases/latest/download/pdive2-linux-amd64
# chmod +x pdive2
# sudo mv pdive2 /usr/local/bin/
# pdive2 --version

# For now, use Option 2 (build from source)
```

### Option 2: Build from Source

```bash
# Prerequisites: Go 1.21 or higher
go version

# Clone and build
git clone https://github.com/your-org/pdive2.git
cd pdive2
make build

# Binary will be in bin/pdive2
./bin/pdive2 --version
```

### Option 3: Go Install

**Note**: Go install will be available once the project is published to a Git repository.

```bash
# This will be available after publishing:
# go install github.com/your-org/pdive2@latest
# export PATH=$PATH:$(go env GOPATH)/bin
# pdive2 --version

# For now, use Option 2 (build from source)
```

## System Requirements

### Operating Systems
- **Linux**: x64, ARM64 (Primary target)
- **macOS**: x64, ARM64 (Apple Silicon)
- **Windows**: x64

### Go Version (for building)
- **Go 1.21** or higher required
- Earlier versions are not supported

## Installing Go

### Ubuntu/Debian

#### Option 1: Using apt (Simple but may not be latest)
```bash
# Install Go from Ubuntu/Debian repositories
sudo apt update
sudo apt install golang-go

# Verify installation
go version

# Note: This may install an older version of Go
# Check if version is 1.21+ for PDive2 compatibility
```

#### Option 2: Manual Installation (Recommended for latest version)
```bash
# Download and install latest Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
```

### Other Operating Systems
- **macOS**: Use Homebrew (`brew install go`) or download from https://go.dev/dl/
- **Windows**: Download installer from https://go.dev/dl/

## External Tool Dependencies

PDive2 requires external security tools for full functionality:

### Required Tools

#### 1. Amass (Required for both modes)
```bash
# Ubuntu/Debian
sudo apt update && sudo apt install amass

# macOS with Homebrew
brew install amass

# Manual installation
go install -v github.com/OWASP/Amass/v3/cmd/amass@master

# Verify installation
amass version
```

#### 2. Masscan (Required for active mode)
```bash
# Ubuntu/Debian
sudo apt install masscan

# Build from source (if package not available)
git clone https://github.com/robertdavidgraham/masscan
cd masscan
make
sudo make install

# Verify installation
masscan --version
```

#### 3. Nmap (Optional for enhanced service detection)
```bash
# Ubuntu/Debian
sudo apt install nmap

# macOS with Homebrew
brew install nmap

# Verify installation
nmap --version
```

### Complete System Setup (Ubuntu/Debian)

```bash
# Install all dependencies at once
sudo apt update
sudo apt install -y amass masscan nmap curl git

# Install Go (if not already installed)
# Option 1: Using apt (Ubuntu/Debian - may not be latest version)
sudo apt install golang-go

# Option 2: Install latest Go manually (recommended)
wget https://go.dev/dl/go1.21.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go1.21.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify Go installation
go version

# Install PDive2
git clone https://github.com/your-org/pdive2.git
cd pdive2
make build
sudo cp bin/pdive2 /usr/local/bin/

# Test complete installation
pdive2 --help
```

## Building from Source

### Development Build

```bash
# Clone repository
git clone https://github.com/your-org/pdive2.git
cd pdive2

# Install dependencies
make deps

# Build binary
make build

# Run tests
make test

# Binary location
./bin/pdive2 --version
```

### Production Build

```bash
# Optimized build with smaller binary size
make build

# Cross-compile for multiple platforms
make cross-compile

# Create release archives
make release
```

### Available Make Targets

```bash
make help              # Show all available targets
make build             # Build the binary
make clean             # Clean build artifacts
make test              # Run tests
make install           # Install to GOPATH/bin
make cross-compile     # Build for multiple platforms
make release           # Create release archives
make dev-build         # Build with debug info
make fmt               # Format Go code
make lint              # Lint the code (requires golangci-lint)
```

## Platform-Specific Instructions

### Linux

```bash
# Install dependencies
sudo apt update
sudo apt install amass masscan nmap

# Build PDive2
git clone https://github.com/your-org/pdive2.git
cd pdive2
make build

# Install system-wide (optional)
sudo cp bin/pdive2 /usr/local/bin/
```

### macOS

```bash
# Install dependencies with Homebrew
brew install amass nmap
brew install masscan  # May need to build from source if not available

# Build PDive2
git clone https://github.com/your-org/pdive2.git
cd pdive2
make build

# Install system-wide (optional)
sudo cp bin/pdive2 /usr/local/bin/
```

### Windows

```bash
# Install dependencies (use WSL for easier setup)
# Or download Windows binaries manually

# Build PDive2 (in WSL or with Go for Windows)
git clone https://github.com/your-org/pdive2.git
cd pdive2
make build

# Use the .exe binary on Windows
./bin/pdive2.exe --help
```

## Docker Installation (Alternative)

### Using Docker

```bash
# Create Dockerfile
cat > Dockerfile << 'EOF'
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make

# Set working directory
WORKDIR /app

# Copy source code
COPY . .

# Build the application
RUN make build

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates

# Install external tools
RUN apk add --no-cache nmap

# Note: Amass and Masscan would need to be built separately for Alpine
# This is a simplified example

# Copy binary from builder
COPY --from=builder /app/bin/pdive2 /usr/local/bin/

# Create output directory
RUN mkdir -p /app/output

# Set working directory
WORKDIR /app

# Entry point
ENTRYPOINT ["pdive2"]
EOF

# Build Docker image
docker build -t pdive2 .

# Run with Docker
docker run --rm -v $(pwd)/output:/app/output pdive2 -t example.com -o /app/output
```

## Verification

### Test Installation

```bash
# Check version
pdive2 --version

# Check help
pdive2 --help

# Test external dependencies
amass version
masscan --version
nmap --version

# Run a simple test (requires authorization)
echo "example.com" > test_targets.txt
pdive2 -f test_targets.txt -m passive -o test_output
```

### Expected Output

```
PDive2 - Automated Penetration Testing Discovery Tool (Go Edition)
Version: 2.0

External Tools:
✓ Amass: Available
✓ Masscan: Available
✓ Nmap: Available
```

## Troubleshooting

### Common Issues

#### 1. "Go not found"
```bash
# Install Go - Option 1: Using apt (Ubuntu/Debian)
sudo apt update && sudo apt install golang-go

# Install Go - Option 2: Manual installation (latest version)
curl -L https://go.dev/dl/go1.21.0.linux-amd64.tar.gz -o go.tar.gz
sudo tar -C /usr/local -xzf go.tar.gz
export PATH=$PATH:/usr/local/go/bin

# Verify installation
go version
```

#### 2. "amass: command not found"
```bash
# Install amass
sudo apt install amass
# OR
go install -v github.com/OWASP/Amass/v3/cmd/amass@master
```

#### 3. "masscan: command not found"
```bash
# Install masscan
sudo apt install masscan
# OR build from source
```

#### 4. Build errors
```bash
# Clean and rebuild
make clean
make deps
make build
```

#### 5. Permission denied
```bash
# Make binary executable
chmod +x bin/pdive2
# OR for system install
sudo chmod +x /usr/local/bin/pdive2
```

### Performance Optimization

#### Memory Usage
```bash
# For systems with limited RAM, reduce thread count
pdive2 -t target -T 25  # Default is 50
```

#### Network Timeouts
```bash
# For slow networks, reduce concurrency
pdive2 -t target -T 10
```

## Upgrading

### From Previous Version

```bash
# Stop running instances
pkill pdive2

# Backup old binary (if needed)
cp /usr/local/bin/pdive2 /usr/local/bin/pdive2.old

# Download new version
curl -L -o pdive2 https://github.com/your-org/pdive2/releases/latest/download/pdive2-linux-amd64
chmod +x pdive2
sudo mv pdive2 /usr/local/bin/

# Verify upgrade
pdive2 --version
```

### From Source

```bash
cd pdive2
git pull origin main
make clean
make build
sudo cp bin/pdive2 /usr/local/bin/
```

## Uninstallation

```bash
# Remove binary
sudo rm -f /usr/local/bin/pdive2

# Remove source directory (if applicable)
rm -rf ~/pdive2

# External tools (optional - only if not used by other applications)
sudo apt remove amass masscan nmap  # Ubuntu/Debian
```

This completes the installation guide for PDive2. For usage instructions, refer to the main README.md file.