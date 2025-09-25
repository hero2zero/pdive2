# PDive2 (Go Edition)

**Dive deep into the network** - An automated penetration testing discovery tool for authorized security assessments.

PDive2 is a high-performance Go rewrite of the original PDIve tool, featuring both passive and active discovery modes. Built for security professionals conducting authorized network assessments, vulnerability testing, and OSINT gathering.

## Key Features

- 🚀 **High Performance**: Written in Go for maximum speed and concurrency
- 🔍 **Dual Discovery Modes**: Passive (stealth OSINT) and Active (comprehensive scanning)
- ⚡ **Concurrent Operations**: Efficient goroutine-based scanning with configurable thread pools
- 🎯 **Multi-Target Support**: IP addresses, CIDR ranges, hostnames, and domain lists
- 📊 **Professional Reporting**: Detailed text and CSV reports for documentation
- 🛡️ **Security-Focused**: Built for authorized testing with clear legal disclaimers
- 🔧 **Single Binary**: No Python dependencies - compile once, run anywhere

## Installation

### Option 1: Build from Source

```bash
# Clone the repository
git clone https://github.com/hero2zero/pdive2.git
cd pdive2

# Build the binary
go build -o pdive2 .

# Run the tool
./pdive2 --help
```

### Option 2: Download Pre-compiled Binary

**Note**: Pre-compiled binaries will be available once the project is published to GitHub releases.

```bash
# This will be available after first release:
# wget https://github.com/hero2zero/pdive2/releases/latest/download/pdive2-linux-amd64
# chmod +x pdive2-linux-amd64
# mv pdive2-linux-amd64 pdive2

# For now, use Option 1 (build from source) or Option 3 (go install)
```

### Option 3: Install with Go

**Note**: Go install will be available once the project is published to a Git repository.

```bash
# This will be available after publishing:
# go install github.com/hero2zero/pdive2@latest

# For now, use Option 1 (build from source)
```

## Prerequisites

### Required System Tools
- **Go 1.21+**: For building from source
- **Amass**: OWASP Amass for passive subdomain enumeration (required for both modes)
- **Masscan**: Fast port scanner (required for active mode)
- **Nmap**: Detailed service enumeration (optional for active mode)

### Installation of External Tools

**Ubuntu/Debian:**
```bash
sudo apt update
sudo apt install amass masscan nmap
```

**Manual Installation:**
```bash
# Amass - https://github.com/OWASP/Amass
# Masscan - https://github.com/robertdavidgraham/masscan
# Nmap - https://nmap.org/download.html
```

## Usage

### Passive Discovery Mode

Perfect for stealth reconnaissance and OSINT gathering:

```bash
# Basic passive discovery
./pdive2 -t example.com -m passive

# Passive discovery from file
./pdive2 -f domains.txt -m passive

# Multiple domains
./pdive2 -t "example.com,testsite.com" -m passive

# Custom output directory
./pdive2 -t example.com -m passive -o /tmp/passive_recon
```

### Active Discovery Mode

Traditional network scanning and analysis:

```bash
# Basic active scan
./pdive2 -t 192.168.1.0/24

# Active scan with nmap integration (when implemented)
./pdive2 -t 10.0.0.1 --nmap

# Multiple targets active scan
./pdive2 -t "192.168.1.1,example.com,10.0.0.0/24"

# High-performance scan with more threads
./pdive2 -t 192.168.1.0/24 -T 200
```

### Advanced Examples

```bash
# Scan from file with custom settings
./pdive2 -f targets.txt -o /tmp/scan_results -T 100

# Domain passive discovery with custom output
./pdive2 -t "*.company.com" -m passive -o /tmp/passive_recon
```

### Command Line Options

```
Usage:
  pdive2 [flags]

Flags:
  -t, --target string    Target IP address, hostname, CIDR range, or comma-separated list
  -f, --file string      File containing targets (one per line)
  -m, --mode string      Discovery mode - active (default) or passive (default "active")
  -o, --output string    Output directory (default "recon_output")
  -T, --threads int      Number of threads (default 50)
      --nmap             Enable detailed Nmap scanning (Active mode only)
  -h, --help             help for pdive2
      --version          version for pdive2
```

**Notes**:
- Either `-t` or `-f` is required, but not both
- `--nmap` flag cannot be used with passive mode
- Passive mode works best with domain names, not IP addresses

### Target File Format

When using the `-f` option, create a text file with one target per line:

```
# Comments start with #
# For passive mode, use domains:
example.com
testsite.org
company.net

# For active mode, use IPs/networks:
192.168.1.0/24
10.0.0.1
server.local
```

## Discovery Methods

### Passive Discovery Techniques

1. **Amass Enumeration**: Uses OWASP Amass for passive subdomain discovery
   - Sources: Certificate transparency, DNS aggregation, web archives
   - Command: `amass enum -d domain.com -passive`
   - Pure passive mode - no active network traffic to targets

### Active Discovery Process

1. **Authorization Check**: Prompts user to confirm scanning authorization
2. **Phase 1 - Amass Discovery**: Passive subdomain enumeration using amass
3. **Phase 2 - Host Discovery**: Concurrent ping sweep and port-based host detection
4. **Phase 3 - Masscan**: Fast port scanning (1-65535) with high concurrency
5. **Phase 4 - Service Enumeration**: HTTP service detection and basic service mapping
6. **Report Generation**: Creates comprehensive scan reports

## Performance Advantages

### Go vs Python Performance

| Feature | PDive (Python) | PDive2 (Go) |
|---------|----------------|-------------|
| **Startup Time** | ~2-3 seconds | ~50ms |
| **Memory Usage** | ~50-100MB | ~10-20MB |
| **Concurrency** | Thread-based (GIL limited) | Goroutine-based (true parallelism) |
| **Port Scanning Speed** | ~500-1000 ports/sec | ~5000-10000 ports/sec |
| **Binary Size** | N/A (requires Python) | ~15MB (self-contained) |
| **Dependencies** | Python + pip packages | Single binary (no runtime deps) |

### Concurrency Model

- **Goroutines**: Lightweight threads for maximum concurrency
- **Channel-based Communication**: Safe data sharing between goroutines
- **Configurable Thread Pools**: Adjustable concurrency limits
- **Memory Efficient**: Low overhead per concurrent operation

## Output and Reports

### Passive Mode Reports

**Host List Report (`passive_discovery_TIMESTAMP.txt`)**:
```
PDIVE2 PASSIVE DISCOVERY REPORT
============================================================

DISCOVERY SUMMARY
--------------------
Targets: example.com
Discovery Mode: PASSIVE
Total Discovered Hosts: 45

DISCOVERED HOSTS
--------------------
accounts.example.com
api.example.com
mail.example.com
www.example.com
```

**CSV Host List (`passive_hosts_TIMESTAMP.csv`)**:
- Simple format: Host, Discovery_Method, Scan_Time
- Perfect for further analysis and integration

### Active Mode Reports

**Detailed Text Report (`recon_report_YYYYMMDD_HHMMSS.txt`)**:
- Complete scan summary with timestamps and statistics
- Detailed host information with port and service listings
- Professional format suitable for documentation

**CSV Report (`recon_results_YYYYMMDD_HHMMSS.csv`)**:
- Structured data: Host, Port, Protocol, State, Service, Scan_Time
- Compatible with Excel, databases, and analysis tools

## Tool Integration

### Masscan Integration (Active Mode - Phase 3)
- **Port Range**: Scans ports 1-65535 (complete coverage)
- **High Speed**: Concurrent execution with configurable rate limiting
- **Output Parsing**: Processes masscan list format output
- **Fallback**: Automatic fallback to built-in scanner if masscan unavailable

### Amass Integration (Both Modes)
- **Passive Enumeration**: Uses `amass enum -d domain.com -passive`
- **Timeout Handling**: 60-second timeout with proper error handling
- **Output Processing**: Parses and validates discovered subdomains
- **Availability Check**: Automatic detection and graceful degradation

## Dependencies

### Go Dependencies
```go
require (
    github.com/fatih/color v1.16.0    // Colored terminal output
    github.com/spf13/cobra v1.8.0     // CLI framework
)
```

### Required External Tools
- **Amass**: OWASP Amass for passive subdomain enumeration
  - Required for both passive and active modes
  - Install: https://github.com/OWASP/Amass
  - Ubuntu/Debian: `sudo apt install amass`

- **Masscan**: Fast port scanner
  - Required for active mode (fallback to basic scan if unavailable)
  - Install: https://github.com/robertdavidgraham/masscan
  - Ubuntu/Debian: `sudo apt install masscan`

- **Nmap**: Detailed service enumeration
  - Optional for active mode (enhanced service detection)
  - Install: https://nmap.org/download.html
  - Ubuntu/Debian: `sudo apt install nmap`

## Build Instructions

### Development Build
```bash
# Clone and build
git clone https://github.com/hero2zero/pdive2.git
cd pdive2
go mod tidy
go build -o pdive2 .
```

### Production Build
```bash
# Optimized build with reduced binary size
go build -ldflags="-s -w" -o pdive2 .

# Cross-compilation for different platforms
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o pdive2-linux-amd64 .
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w" -o pdive2-windows-amd64.exe .
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o pdive2-darwin-amd64 .
```

### Docker Build (Optional)
```bash
# Create a Dockerfile for containerized deployment
docker build -t pdive2 .
docker run --rm -v $(pwd)/output:/app/output pdive2 -t example.com -o /app/output
```

## Use Cases

### Passive Mode - Perfect For:
- 🕵️ **OSINT Collection**: Gathering public information without direct contact
- 🔒 **Stealth Reconnaissance**: Minimal network footprint operations
- 📊 **Domain Analysis**: Understanding an organization's digital footprint
- 🛡️ **Defensive Assessment**: Identifying your own exposed assets
- 📋 **Compliance Auditing**: Asset discovery for security compliance

### Active Mode - Ideal For:
- 🎯 **Penetration Testing**: Authorized security assessments
- 🔍 **Vulnerability Assessment**: Identifying open services and versions
- 🖥️ **Network Discovery**: Mapping internal network topology
- 🛠️ **Infrastructure Analysis**: Detailed service enumeration
- 📈 **Security Monitoring**: Regular network security checks

## Troubleshooting

### Build Issues

```bash
# Update Go modules
go mod tidy

# Clean build cache
go clean -cache -modcache

# Verify Go version
go version  # Should be 1.21 or higher
```

### Missing System Packages

On Debian/Ubuntu systems:
```bash
# Install all required packages
sudo apt update
sudo apt install amass masscan nmap

# Verify installation
which amass masscan nmap
```

### Common Runtime Issues

- **Passive mode with IPs**: Use domain names for passive discovery, not IP addresses
- **Amass timeout**: Large domains may take longer; tool has built-in timeout handling
- **Permission denied**: Ensure proper file permissions for output directory
- **Network timeouts**: Reduce thread count with `-T` option for slower networks
- **Binary not found**: Make sure pdive2 binary has execute permissions (`chmod +x pdive2`)

## Migration from Python Version

### Key Differences

| Aspect | PDive (Python) | PDive2 (Go) |
|--------|----------------|-------------|
| **Dependencies** | pip install -r requirements.txt | Single binary |
| **Execution** | python pdive.py | ./pdive2 |
| **Performance** | Thread-limited concurrency | True parallelism |
| **Memory Usage** | Higher overhead | Lower footprint |
| **Cross-platform** | Requires Python runtime | Self-contained binary |

### Command Compatibility

Most commands are directly compatible:

```bash
# Python version
python pdive.py -t example.com -m passive

# Go version
./pdive2 -t example.com -m passive
```

## Security Considerations

- **Authorization**: Always obtain explicit written permission before scanning
- **Scope**: Stay within authorized target scope and timeframes
- **Rate Limiting**: Use appropriate thread counts to avoid overwhelming targets
- **Data Handling**: Secure storage and disposal of reconnaissance data
- **Legal Compliance**: Follow local laws and organizational policies
- **Ethical Use**: Use for legitimate security testing and defensive purposes only

## Examples

### Comprehensive Passive Reconnaissance
```bash
# Discover all subdomains for multiple organizations
echo -e "example.com\ncompany.org\ntarget.net" > domains.txt
./pdive2 -f domains.txt -m passive -o passive_results

# Results show all discovered subdomains from amass
```

### High-Performance Active Network Assessment
```bash
# Full internal network scan with maximum concurrency
./pdive2 -t 192.168.0.0/16 -m active -o internal_scan -T 200

# Results include live hosts, open ports, and service identification
```

### Hybrid Approach
```bash
# 1. Start with passive discovery
./pdive2 -t company.com -m passive -o recon_phase1

# 2. Use discovered hosts for targeted active scanning
./pdive2 -f discovered_hosts.txt -m active -o recon_phase2
```

## Version History

- **v2.0**: Go rewrite with significant performance improvements, native concurrency, and single binary distribution
- **v1.2**: Original Python version with enhanced workflow and dual discovery modes
- **v1.1**: Added passive discovery mode with external tool integration
- **v1.0**: Initial Python release

## Contributing

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This tool is provided for educational and authorized security testing purposes only.

## Disclaimer

The authors are not responsible for any misuse of this tool. Users are solely responsible for ensuring they have proper authorization before using this tool on any network or system. Both passive and active reconnaissance should be conducted within the bounds of authorized testing scope and applicable laws.