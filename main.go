package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// Version information
const (
	Version = "2.0"
	Banner  = `
██████╗ ██████╗ ██╗██╗   ██╗███████╗██████╗
██╔══██╗██╔══██╗██║██║   ██║██╔════╝╚════██╗
██████╔╝██║  ██║██║██║   ██║█████╗   █████╔╝
██╔═══╝ ██║  ██║██║╚██╗ ██╔╝██╔══╝  ██╔═══╝
██║     ██████╔╝██║ ╚████╔╝ ███████╗███████╗
╚═╝     ╚═════╝ ╚═╝  ╚═══╝  ╚══════╝╚══════╝
`
)

// PortInfo represents information about a scanned port
type PortInfo struct {
	Port    int    `json:"port"`
	State   string `json:"state"`
	Service string `json:"service"`
}

// HostInfo represents information about a discovered host
type HostInfo struct {
	Host   string     `json:"host"`
	Status string     `json:"status"`
	Ports  []PortInfo `json:"ports"`
}

// ScanInfo represents metadata about the scan
type ScanInfo struct {
	Targets       []string  `json:"targets"`
	StartTime     time.Time `json:"start_time"`
	Scanner       string    `json:"scanner"`
	DiscoveryMode string    `json:"discovery_mode"`
}

// ScanResults represents the complete scan results
type ScanResults struct {
	ScanInfo           ScanInfo   `json:"scan_info"`
	Hosts              []HostInfo `json:"hosts"`
	UnresponsiveHosts  int        `json:"unresponsive_hosts"`
	mutex              sync.RWMutex
}

// PDive2 represents the main scanner configuration
type PDive2 struct {
	Targets       []string
	OutputDir     string
	Threads       int
	DiscoveryMode string
	Results       *ScanResults
	EnableNmap    bool
}

// NewPDive2 creates a new PDive2 instance
func NewPDive2(targets []string, outputDir string, threads int, discoveryMode string) *PDive2 {
	return &PDive2{
		Targets:       targets,
		OutputDir:     outputDir,
		Threads:       threads,
		DiscoveryMode: discoveryMode,
		Results: &ScanResults{
			ScanInfo: ScanInfo{
				Targets:       targets,
				StartTime:     time.Now(),
				Scanner:       fmt.Sprintf("PDive2 v%s", Version),
				DiscoveryMode: discoveryMode,
			},
			Hosts: make([]HostInfo, 0),
		},
	}
}

// Colors for output
var (
	cyan   = color.New(color.FgCyan)
	yellow = color.New(color.FgYellow)
	green  = color.New(color.FgGreen)
	red    = color.New(color.FgRed)
)

// PrintBanner prints the application banner
func (p *PDive2) PrintBanner() {
	targetsDisplay := strings.Join(p.Targets[:min(3, len(p.Targets))], ", ")
	if len(p.Targets) > 3 {
		targetsDisplay += fmt.Sprintf(" ... (+%d more)", len(p.Targets)-3)
	}

	cyan.Print(Banner)
	yellow.Println("Dive deep into the network")
	red.Println("For authorized security testing only!")
	fmt.Println()

	fmt.Printf("Targets (%d): %s\n", len(p.Targets), green.Sprintf(targetsDisplay))
	fmt.Printf("Output Directory: %s\n", green.Sprint(p.OutputDir))
	fmt.Printf("Threads: %s\n", green.Sprint(p.Threads))
	fmt.Printf("Discovery Mode: %s\n", green.Sprint(strings.ToUpper(p.DiscoveryMode)))
	fmt.Println()
}

// ValidateTargets validates if all targets are valid IP addresses, network ranges, or hostnames
func (p *PDive2) ValidateTargets() bool {
	var validTargets []string
	var invalidTargets []string

	for _, target := range p.Targets {
		if isValidTarget(target) {
			validTargets = append(validTargets, target)
		} else {
			invalidTargets = append(invalidTargets, target)
		}
	}

	if len(invalidTargets) > 0 {
		red.Printf("[-] Invalid targets: %s\n", strings.Join(invalidTargets, ", "))
	}

	p.Targets = validTargets
	return len(validTargets) > 0
}

// isValidTarget checks if a target is a valid IP, CIDR, or hostname
func isValidTarget(target string) bool {
	// Try parsing as IP/CIDR
	if _, _, err := net.ParseCIDR(target); err == nil {
		return true
	}
	if net.ParseIP(target) != nil {
		return true
	}

	// Try resolving as hostname
	if _, err := net.LookupHost(target); err == nil {
		return true
	}

	return false
}

// expandTargets expands CIDR ranges to individual IPs
func (p *PDive2) expandTargets() []string {
	var allHosts []string

	for _, target := range p.Targets {
		if strings.Contains(target, "/") {
			// CIDR range
			if ip, ipnet, err := net.ParseCIDR(target); err == nil {
				for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); inc(ip) {
					allHosts = append(allHosts, ip.String())
				}
			}
		} else {
			allHosts = append(allHosts, target)
		}
	}

	return removeDuplicates(allHosts)
}

// inc increments an IP address
func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

// removeDuplicates removes duplicate strings from a slice
func removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// HostDiscovery performs host discovery using ping and port-based detection
func (p *PDive2) HostDiscovery() []string {
	yellow.Println("\n[+] Starting Host Discovery...")

	allHosts := p.expandTargets()
	liveHosts := make(map[string]bool)
	var mu sync.Mutex

	// Common ports for host discovery fallback
	discoveryPorts := []int{80, 443, 22, 21, 25, 53, 135, 139, 445}

	// Phase 1: Ping discovery
	cyan.Println("[*] Phase 1: Ping discovery...")
	var wg sync.WaitGroup
	hostChan := make(chan string, len(allHosts))

	for _, host := range allHosts {
		hostChan <- host
	}
	close(hostChan)

	// Start ping workers
	for i := 0; i < p.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hostChan {
				if p.pingHost(host) {
					mu.Lock()
					liveHosts[host] = true
					mu.Unlock()
					green.Printf("[+] Host discovered (ping): %s\n", host)
				}
			}
		}()
	}

	wg.Wait()

	// Phase 2: Port-based discovery for non-ping responsive hosts
	var nonPingHosts []string
	for _, host := range allHosts {
		if !liveHosts[host] {
			nonPingHosts = append(nonPingHosts, host)
		}
	}

	if len(nonPingHosts) > 0 {
		cyan.Printf("[*] Phase 2: Port-based discovery for %d non-ping responsive hosts...\n", len(nonPingHosts))

		hostChan = make(chan string, len(nonPingHosts))
		for _, host := range nonPingHosts {
			hostChan <- host
		}
		close(hostChan)

		// Start port discovery workers
		for i := 0; i < min(p.Threads, 20); i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for host := range hostChan {
					if p.portDiscovery(host, discoveryPorts) {
						mu.Lock()
						liveHosts[host] = true
						mu.Unlock()
						green.Printf("[+] Host discovered (port): %s\n", host)
					}
				}
			}()
		}

		wg.Wait()
	}

	// Convert map to slice
	var liveHostsList []string
	for host := range liveHosts {
		liveHostsList = append(liveHostsList, host)
	}

	// Update results
	p.Results.mutex.Lock()
	for _, host := range liveHostsList {
		p.Results.Hosts = append(p.Results.Hosts, HostInfo{
			Host:   host,
			Status: "up",
			Ports:  make([]PortInfo, 0),
		})
	}
	p.Results.UnresponsiveHosts = len(allHosts) - len(liveHostsList)
	p.Results.mutex.Unlock()

	cyan.Printf("\n[*] Host discovery completed. Found %d live hosts from %d total hosts.\n",
		len(liveHostsList), len(allHosts))
	cyan.Printf("[*] Ping responsive: %d, Port responsive: %d\n",
		len(liveHosts)-len(nonPingHosts), len(liveHostsList)-(len(liveHosts)-len(nonPingHosts)))

	return liveHostsList
}

// pingHost performs a ping test on a host
func (p *PDive2) pingHost(host string) bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", "1", "-W", "2", host)
	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run() == nil
}

// portDiscovery tries to connect to common ports to detect live hosts
func (p *PDive2) portDiscovery(host string, ports []int) bool {
	for _, port := range ports {
		conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
		if err == nil {
			conn.Close()
			return true
		}
	}
	return false
}

// PortScan performs port scanning on discovered hosts
func (p *PDive2) PortScan(hosts []string) {
	yellow.Println("\n[+] Starting Port Scanning...")

	commonPorts := []int{21, 22, 23, 25, 53, 80, 110, 111, 135, 139, 143, 443, 993, 995, 1723, 3306, 3389, 5432, 5900, 8080, 8443}

	var wg sync.WaitGroup
	hostChan := make(chan string, len(hosts))

	for _, host := range hosts {
		hostChan <- host
	}
	close(hostChan)

	// Start port scanning workers
	for i := 0; i < p.Threads; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range hostChan {
				p.scanHostPorts(host, commonPorts)
			}
		}()
	}

	wg.Wait()
}

// scanHostPorts scans ports for a specific host
func (p *PDive2) scanHostPorts(host string, ports []int) {
	cyan.Printf("\n[*] Scanning %s...\n", host)
	var openPorts []PortInfo

	var wg sync.WaitGroup
	var mu sync.Mutex
	portChan := make(chan int, len(ports))

	for _, port := range ports {
		portChan <- port
	}
	close(portChan)

	// Start port workers
	for i := 0; i < min(p.Threads, 50); i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for port := range portChan {
				if p.scanPort(host, port) {
					mu.Lock()
					openPorts = append(openPorts, PortInfo{
						Port:    port,
						State:   "open",
						Service: "",
					})
					mu.Unlock()
					green.Printf("[+] Open port found: %s:%d\n", host, port)
				}
			}
		}()
	}

	wg.Wait()

	// Update results
	p.Results.mutex.Lock()
	for i := range p.Results.Hosts {
		if p.Results.Hosts[i].Host == host {
			p.Results.Hosts[i].Ports = openPorts
			break
		}
	}
	p.Results.mutex.Unlock()
}

// scanPort scans a specific port on a host
func (p *PDive2) scanPort(host string, port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("%s:%d", host, port), 3*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// ServiceEnumeration performs service enumeration on open ports
func (p *PDive2) ServiceEnumeration(hosts []string) {
	yellow.Println("\n[+] Starting Service Enumeration...")

	serviceMap := map[int]string{
		21: "ftp", 22: "ssh", 23: "telnet", 25: "smtp", 53: "dns",
		80: "http", 110: "pop3", 135: "rpc", 139: "netbios", 143: "imap",
		443: "https", 993: "imaps", 995: "pop3s", 1723: "pptp",
		3306: "mysql", 3389: "rdp", 5432: "postgresql", 5900: "vnc",
		8080: "http-alt", 8443: "https-alt",
	}

	for _, host := range hosts {
		p.Results.mutex.Lock()
		var hostIndex int = -1
		for i, h := range p.Results.Hosts {
			if h.Host == host {
				hostIndex = i
				break
			}
		}

		if hostIndex != -1 {
			for j, port := range p.Results.Hosts[hostIndex].Ports {
				service := p.enumerateService(host, port.Port, serviceMap)
				p.Results.Hosts[hostIndex].Ports[j].Service = service
				green.Printf("[+] Service identified: %s:%d -> %s\n", host, port.Port, service)
			}
		}
		p.Results.mutex.Unlock()
	}
}

// enumerateService performs basic service enumeration
func (p *PDive2) enumerateService(host string, port int, serviceMap map[int]string) string {
	service, exists := serviceMap[port]
	if !exists {
		return "unknown"
	}

	// Enhanced HTTP service detection
	if service == "http" || service == "https" || service == "http-alt" || service == "https-alt" {
		protocol := "http"
		if service == "https" || service == "https-alt" {
			protocol = "https"
		}

		portStr := ""
		if port != 80 && port != 443 {
			portStr = fmt.Sprintf(":%d", port)
		}

		url := fmt.Sprintf("%s://%s%s", protocol, host, portStr)

		client := &http.Client{
			Timeout: 5 * time.Second,
		}

		if resp, err := client.Get(url); err == nil {
			defer resp.Body.Close()
			if server := resp.Header.Get("Server"); server != "" {
				return fmt.Sprintf("%s (%s)", service, server)
			}
		}
	}

	return service
}

// PassiveDiscovery performs passive discovery using amass only
func (p *PDive2) PassiveDiscovery() []string {
	yellow.Println("\n[+] Starting Passive Discovery (amass only)...")

	var discoveredHosts []string

	for _, target := range p.Targets {
		domain := p.extractDomain(target)
		if domain == "" {
			continue
		}

		cyan.Printf("[*] Performing passive discovery on domain: %s\n", domain)
		hosts := p.amassDiscovery(domain)
		discoveredHosts = append(discoveredHosts, hosts...)
	}

	discoveredHosts = removeDuplicates(discoveredHosts)

	// Add discovered hosts to results
	p.Results.mutex.Lock()
	for _, host := range discoveredHosts {
		p.Results.Hosts = append(p.Results.Hosts, HostInfo{
			Host:   host,
			Status: "discovered",
			Ports:  make([]PortInfo, 0),
		})
	}
	p.Results.mutex.Unlock()

	cyan.Printf("\n[*] Passive discovery completed. Found %d hosts.\n", len(discoveredHosts))

	return discoveredHosts
}

// extractDomain extracts domain name from target
func (p *PDive2) extractDomain(target string) string {
	// If it's an IP or CIDR, skip
	if net.ParseIP(target) != nil {
		return ""
	}
	if _, _, err := net.ParseCIDR(target); err == nil {
		return ""
	}

	return strings.ToLower(strings.TrimSpace(target))
}

// amassDiscovery uses amass for passive subdomain enumeration
func (p *PDive2) amassDiscovery(domain string) []string {
	var discoveredHosts []string

	cyan.Printf("[*] Running amass on %s...\n", domain)

	// Check if amass is available
	if _, err := exec.LookPath("amass"); err != nil {
		red.Println("[-] Amass not found in PATH, skipping amass discovery")
		yellow.Println("[*] Install amass from: https://github.com/OWASP/Amass")
		return discoveredHosts
	}

	// Run amass with specified options (passive mode only)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "amass", "enum", "-d", domain, "-passive")
	output, err := cmd.Output()

	if err != nil {
		red.Printf("[-] Amass failed: %v\n", err)
		return discoveredHosts
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			discoveredHosts = append(discoveredHosts, line)
			green.Printf("[+] Amass discovered: %s\n", line)
		}
	}

	if len(discoveredHosts) == 0 {
		yellow.Printf("[*] Amass completed but found no subdomains for %s\n", domain)
	}

	return discoveredHosts
}

// MasscanScan performs fast port scanning using masscan
func (p *PDive2) MasscanScan(hosts []string) map[string][]PortInfo {
	yellow.Println("\n[+] Starting Fast Port Scan (masscan)...")

	if len(hosts) == 0 {
		red.Println("[-] No hosts provided for masscan")
		return make(map[string][]PortInfo)
	}

	// Check if masscan is available
	if _, err := exec.LookPath("masscan"); err != nil {
		red.Println("[-] Masscan not found in PATH, falling back to basic port scan")
		yellow.Println("[*] Install masscan from: https://github.com/robertdavidgraham/masscan")
		p.PortScan(hosts)

		// Convert results format
		results := make(map[string][]PortInfo)
		p.Results.mutex.RLock()
		for _, host := range p.Results.Hosts {
			if len(host.Ports) > 0 {
				results[host.Host] = host.Ports
			}
		}
		p.Results.mutex.RUnlock()
		return results
	}

	masscanResults := make(map[string][]PortInfo)

	// Create temporary target file for masscan
	tmpfile, err := os.CreateTemp("", "masscan_targets_*.txt")
	if err != nil {
		red.Printf("[-] Failed to create temp file: %v\n", err)
		return masscanResults
	}
	defer os.Remove(tmpfile.Name())

	for _, host := range hosts {
		fmt.Fprintln(tmpfile, host)
	}
	tmpfile.Close()

	cyan.Printf("[*] Running masscan on %d hosts...\n", len(hosts))

	// Run masscan with output in list format
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "masscan", "-iL", tmpfile.Name(), "-p1-65535", "--rate", "1000", "--output-format", "list")
	output, err := cmd.Output()

	if err != nil {
		red.Printf("[-] Masscan failed: %v\n", err)
		yellow.Println("[*] Falling back to basic port scan...")
		p.PortScan(hosts)

		// Convert results format
		results := make(map[string][]PortInfo)
		p.Results.mutex.RLock()
		for _, host := range p.Results.Hosts {
			if len(host.Ports) > 0 {
				results[host.Host] = host.Ports
			}
		}
		p.Results.mutex.RUnlock()
		return results
	}

	// Parse masscan output
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			// Masscan list format: "open tcp 80 1.2.3.4 1234567890"
			parts := strings.Fields(line)
			if len(parts) >= 4 && parts[0] == "open" && parts[1] == "tcp" {
				portStr := parts[2]
				host := parts[3]

				if port, err := strconv.Atoi(portStr); err == nil {
					if _, exists := masscanResults[host]; !exists {
						masscanResults[host] = make([]PortInfo, 0)
					}
					masscanResults[host] = append(masscanResults[host], PortInfo{
						Port:    port,
						State:   "open",
						Service: "",
					})

					green.Printf("[+] Masscan found: %s:%s\n", host, portStr)
				}
			}
		}
	}

	cyan.Printf("\n[*] Masscan completed. Found ports on %d hosts.\n", len(masscanResults))

	// Update results with masscan findings
	p.Results.mutex.Lock()
	for _, host := range hosts {
		var hostIndex int = -1
		for i, h := range p.Results.Hosts {
			if h.Host == host {
				hostIndex = i
				break
			}
		}

		if hostIndex == -1 {
			p.Results.Hosts = append(p.Results.Hosts, HostInfo{
				Host:   host,
				Status: "up",
				Ports:  make([]PortInfo, 0),
			})
			hostIndex = len(p.Results.Hosts) - 1
		}

		if ports, exists := masscanResults[host]; exists {
			p.Results.Hosts[hostIndex].Ports = append(p.Results.Hosts[hostIndex].Ports, ports...)
		}
	}
	p.Results.mutex.Unlock()

	return masscanResults
}

// GenerateReport generates comprehensive scan reports in text and CSV format
func (p *PDive2) GenerateReport() {
	yellow.Println("\n[+] Generating Reports...")

	// Create output directory
	if err := os.MkdirAll(p.OutputDir, 0755); err != nil {
		red.Printf("[-] Failed to create output directory: %v\n", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	endTime := time.Now()

	p.Results.mutex.RLock()
	totalHosts := len(p.Results.Hosts)
	totalPorts := 0
	for _, host := range p.Results.Hosts {
		totalPorts += len(host.Ports)
	}
	p.Results.mutex.RUnlock()

	// Generate detailed text report
	txtFile := filepath.Join(p.OutputDir, fmt.Sprintf("recon_report_%s.txt", timestamp))
	if f, err := os.Create(txtFile); err == nil {
		defer f.Close()

		fmt.Fprintln(f, "PDIVE2 DETAILED SCAN REPORT")
		fmt.Fprintln(f, strings.Repeat("=", 60))
		fmt.Fprintln(f)

		// Summary section
		fmt.Fprintln(f, "SCAN SUMMARY")
		fmt.Fprintln(f, strings.Repeat("-", 20))
		fmt.Fprintln(f, "Targets:")
		for _, target := range p.Targets {
			fmt.Fprintf(f, "  %s\n", target)
		}
		fmt.Fprintf(f, "\nScan Start Time: %s\n", p.Results.ScanInfo.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "Scan End Time: %s\n", endTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "Scanner Version: %s\n", p.Results.ScanInfo.Scanner)
		fmt.Fprintf(f, "Total Live Hosts: %d\n", totalHosts)
		fmt.Fprintf(f, "Total Open Ports: %d\n", totalPorts)
		fmt.Fprintf(f, "Unresponsive Hosts: %d\n\n", p.Results.UnresponsiveHosts)

		// Detailed results section
		fmt.Fprintln(f, "DETAILED RESULTS")
		fmt.Fprintln(f, strings.Repeat("-", 20))

		p.Results.mutex.RLock()
		if len(p.Results.Hosts) > 0 {
			for _, host := range p.Results.Hosts {
				fmt.Fprintf(f, "\nHost: %s\n", host.Host)
				fmt.Fprintln(f, strings.Repeat("=", len(host.Host)+6))
				if len(host.Ports) > 0 {
					fmt.Fprintln(f, "Open Ports:")
					for _, port := range host.Ports {
						service := port.Service
						if service == "" {
							service = "unknown"
						}
						fmt.Fprintf(f, "  %5d/tcp  %s\n", port.Port, service)
					}
				} else {
					fmt.Fprintln(f, "  No open ports detected")
				}
			}
		} else {
			fmt.Fprintln(f, "No live hosts discovered")
		}
		p.Results.mutex.RUnlock()
	}

	// Generate CSV report
	csvFile := filepath.Join(p.OutputDir, fmt.Sprintf("recon_results_%s.csv", timestamp))
	if f, err := os.Create(csvFile); err == nil {
		defer f.Close()

		writer := csv.NewWriter(f)
		defer writer.Flush()

		// CSV Headers
		writer.Write([]string{"Host", "Port", "Protocol", "State", "Service", "Scan_Time"})

		// CSV Data
		scanTime := p.Results.ScanInfo.StartTime.Format("2006-01-02 15:04:05")

		p.Results.mutex.RLock()
		if len(p.Results.Hosts) > 0 {
			for _, host := range p.Results.Hosts {
				if len(host.Ports) > 0 {
					for _, port := range host.Ports {
						service := port.Service
						if service == "" {
							service = "unknown"
						}
						writer.Write([]string{
							host.Host,
							strconv.Itoa(port.Port),
							"tcp",
							port.State,
							service,
							scanTime,
						})
					}
				} else {
					// Host is up but no ports detected
					writer.Write([]string{host.Host, "", "", "host_up", "no_open_ports", scanTime})
				}
			}
		}
		p.Results.mutex.RUnlock()
	}

	green.Println("[+] Reports saved to:")
	fmt.Printf("  - Detailed Report: %s\n", txtFile)
	fmt.Printf("  - CSV Data: %s\n", csvFile)
}

// GeneratePassiveReport generates simple report for passive discovery mode
func (p *PDive2) GeneratePassiveReport() {
	yellow.Println("\n[+] Generating Passive Discovery Report...")

	// Create output directory
	if err := os.MkdirAll(p.OutputDir, 0755); err != nil {
		red.Printf("[-] Failed to create output directory: %v\n", err)
		return
	}

	timestamp := time.Now().Format("20060102_150405")
	endTime := time.Now()

	p.Results.mutex.RLock()
	totalHosts := len(p.Results.Hosts)
	p.Results.mutex.RUnlock()

	// Generate simple text report for passive mode
	txtFile := filepath.Join(p.OutputDir, fmt.Sprintf("passive_discovery_%s.txt", timestamp))
	if f, err := os.Create(txtFile); err == nil {
		defer f.Close()

		fmt.Fprintln(f, "PDIVE2 PASSIVE DISCOVERY REPORT")
		fmt.Fprintln(f, strings.Repeat("=", 60))
		fmt.Fprintln(f)

		// Summary section
		fmt.Fprintln(f, "DISCOVERY SUMMARY")
		fmt.Fprintln(f, strings.Repeat("-", 20))
		fmt.Fprintln(f, "Targets:")
		for _, target := range p.Targets {
			fmt.Fprintf(f, "  %s\n", target)
		}
		fmt.Fprintf(f, "\nScan Start Time: %s\n", p.Results.ScanInfo.StartTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "Scan End Time: %s\n", endTime.Format("2006-01-02 15:04:05"))
		fmt.Fprintf(f, "Scanner Version: %s\n", p.Results.ScanInfo.Scanner)
		fmt.Fprintf(f, "Discovery Mode: %s\n", strings.ToUpper(p.Results.ScanInfo.DiscoveryMode))
		fmt.Fprintf(f, "Total Discovered Hosts: %d\n\n", totalHosts)

		// Host list section
		fmt.Fprintln(f, "DISCOVERED HOSTS")
		fmt.Fprintln(f, strings.Repeat("-", 20))

		p.Results.mutex.RLock()
		if len(p.Results.Hosts) > 0 {
			var hosts []string
			for _, host := range p.Results.Hosts {
				hosts = append(hosts, host.Host)
			}
			sort.Strings(hosts)
			for _, host := range hosts {
				fmt.Fprintln(f, host)
			}
		} else {
			fmt.Fprintln(f, "No hosts discovered")
		}
		p.Results.mutex.RUnlock()
	}

	// Generate simple CSV with just hostnames
	csvFile := filepath.Join(p.OutputDir, fmt.Sprintf("passive_hosts_%s.csv", timestamp))
	if f, err := os.Create(csvFile); err == nil {
		defer f.Close()

		writer := csv.NewWriter(f)
		defer writer.Flush()

		// CSV Headers
		writer.Write([]string{"Host", "Discovery_Method", "Scan_Time"})

		// CSV Data
		scanTime := p.Results.ScanInfo.StartTime.Format("2006-01-02 15:04:05")

		p.Results.mutex.RLock()
		if len(p.Results.Hosts) > 0 {
			for _, host := range p.Results.Hosts {
				writer.Write([]string{host.Host, "passive", scanTime})
			}
		}
		p.Results.mutex.RUnlock()
	}

	green.Println("[+] Passive discovery reports saved to:")
	fmt.Printf("  - Host List Report: %s\n", txtFile)
	fmt.Printf("  - CSV Host List: %s\n", csvFile)
}

// RunScan executes complete reconnaissance scan
func (p *PDive2) RunScan() {
	if !p.ValidateTargets() {
		red.Println("[-] No valid targets found")
		return
	}

	p.PrintBanner()

	if p.DiscoveryMode == "passive" {
		// Passive discovery mode - use passive techniques only
		discoveredHosts := p.PassiveDiscovery()
		if len(discoveredHosts) == 0 {
			red.Println("[-] No hosts discovered through passive methods.")
			return
		}

		// In passive mode, only return the list of discovered hosts
		yellow.Println("\n[+] PASSIVE DISCOVERY RESULTS")
		yellow.Println(strings.Repeat("=", 50))
		cyan.Printf("Total hosts discovered: %d\n\n", len(discoveredHosts))

		green.Println("Discovered hosts:")
		sort.Strings(discoveredHosts)
		for _, host := range discoveredHosts {
			fmt.Println(host)
		}

		// Generate simple report for passive mode
		p.GeneratePassiveReport()

	} else {
		// Active discovery mode - amass -> host discovery -> masscan -> nmap
		yellow.Println("\n[+] Starting Active Discovery Mode")
		cyan.Println("[*] Phase 1: Passive subdomain discovery with amass")

		// First, run amass to discover subdomains
		amassHosts := p.PassiveDiscovery()

		// Then do traditional host discovery
		cyan.Println("\n[*] Phase 2: Host discovery and connectivity check")
		liveHosts := p.HostDiscovery()

		// Combine amass results with live host discovery
		allDiscoveredHosts := removeDuplicates(append(amassHosts, liveHosts...))

		if len(allDiscoveredHosts) == 0 {
			red.Println("[-] No live hosts discovered.")
			return
		}

		// Ensure all discovered hosts are initialized in results before proceeding
		p.Results.mutex.Lock()
		hostMap := make(map[string]bool)
		for _, host := range p.Results.Hosts {
			hostMap[host.Host] = true
		}
		for _, host := range allDiscoveredHosts {
			if !hostMap[host] {
				p.Results.Hosts = append(p.Results.Hosts, HostInfo{
					Host:   host,
					Status: "up",
					Ports:  make([]PortInfo, 0),
				})
			}
		}
		p.Results.mutex.Unlock()

		cyan.Println("\n[*] Phase 3: Fast port scanning with masscan")
		// Use masscan for fast port discovery
		masscanResults := p.MasscanScan(allDiscoveredHosts)

		if p.EnableNmap && len(masscanResults) > 0 {
			cyan.Println("\n[*] Phase 4: Detailed service enumeration with nmap")
			// Note: Nmap integration would be implemented here
			yellow.Println("[*] Nmap integration not yet implemented in Go version")
		}

		if len(masscanResults) > 0 {
			// Do basic service enumeration on masscan results
			cyan.Println("\n[*] Phase 4: Basic service identification")
			p.ServiceEnumeration(allDiscoveredHosts)
		}

		// Generate full report for active mode
		p.GenerateReport()
	}

	green.Println("\n[+] Reconnaissance scan completed!")
}

// LoadTargetsFromFile loads targets from a text file, one per line
func LoadTargetsFromFile(filePath string) ([]string, error) {
	var targets []string

	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("target file not found: %s", filePath)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		target := strings.TrimSpace(scanner.Text())
		if target != "" && !strings.HasPrefix(target, "#") {
			targets = append(targets, target)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading target file: %v", err)
	}

	return targets, nil
}

// CLI command configuration
var (
	targetFlag     string
	targetFileFlag string
	outputFlag     string
	threadsFlag    int
	modeFlag       string
	nmapFlag       bool
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "pdive2",
		Short:   "PDIve2 - Automated Penetration Testing Discovery Tool (Go Edition)",
		Long: `PDIve2 - Automated Penetration Testing Discovery Tool (Go Edition)
Dive deep into the network - A defensive security tool for authorized network reconnaissance and vulnerability assessment.

Examples:
  pdive2 -t 192.168.1.0/24
  pdive2 -t 10.0.0.1 --nmap
  pdive2 -f targets.txt -o /tmp/scan_results -T 100
  pdive2 -t "192.168.1.1,example.com,10.0.0.0/24"
  pdive2 -t example.com -m passive
  pdive2 -t testphp.vulnweb.com -m active --nmap`,
		Version: Version,
		Run: func(cmd *cobra.Command, args []string) {
			// Validate mode and nmap combination
			if modeFlag == "passive" && nmapFlag {
				red.Println("[-] Error: --nmap flag is not compatible with passive mode")
				os.Exit(1)
			}

			var targets []string
			var err error

			if targetFileFlag != "" {
				targets, err = LoadTargetsFromFile(targetFileFlag)
				if err != nil {
					red.Printf("[-] %v\n", err)
					os.Exit(1)
				}
				if len(targets) == 0 {
					red.Println("[-] No valid targets found in file")
					os.Exit(1)
				}
			} else if targetFlag != "" {
				if strings.Contains(targetFlag, ",") {
					for _, t := range strings.Split(targetFlag, ",") {
						t = strings.TrimSpace(t)
						if t != "" {
							targets = append(targets, t)
						}
					}
				} else {
					targets = []string{targetFlag}
				}
			} else {
				red.Println("[-] Either -t or -f flag is required")
				os.Exit(1)
			}

			red.Println("WARNING: This tool is for authorized security testing only!")
			red.Println("Ensure you have proper permission before scanning any network.\n")

			targetsDisplay := strings.Join(targets[:min(3, len(targets))], ", ")
			if len(targets) > 3 {
				targetsDisplay += fmt.Sprintf(" ... (+%d more)", len(targets)-3)
			}

			fmt.Printf("Targets to scan: %s\n", targetsDisplay)
			fmt.Print("Do you have authorization to scan these targets? (y/N): ")

			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))

			if response != "y" {
				fmt.Println("Scan aborted.")
				os.Exit(1)
			}

			pdive := NewPDive2(targets, outputFlag, threadsFlag, modeFlag)
			pdive.EnableNmap = nmapFlag
			pdive.RunScan()
		},
	}

	rootCmd.Flags().StringVarP(&targetFlag, "target", "t", "", "Target IP address, hostname, CIDR range, or comma-separated list")
	rootCmd.Flags().StringVarP(&targetFileFlag, "file", "f", "", "File containing targets (one per line)")
	rootCmd.Flags().StringVarP(&outputFlag, "output", "o", "recon_output", "Output directory (default: recon_output)")
	rootCmd.Flags().IntVarP(&threadsFlag, "threads", "T", 50, "Number of threads (default: 50)")
	rootCmd.Flags().StringVarP(&modeFlag, "mode", "m", "active", "Discovery mode: active (default) or passive")
	rootCmd.Flags().BoolVar(&nmapFlag, "nmap", false, "Enable detailed Nmap scanning (Active mode only)")

	rootCmd.MarkFlagsMutuallyExclusive("target", "file")

	if err := rootCmd.Execute(); err != nil {
		red.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}