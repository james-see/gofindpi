package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-ping/ping"
	"github.com/jaypipes/ghw"
)

// Raspberry Pi MAC address prefixes (OUI - Organizationally Unique Identifier)
// Comprehensive list including all known Pi models
var raspberryPiMACs = []string{
	"b8:27:eb", // Original Raspberry Pi Foundation
	"dc:a6:32", // Raspberry Pi Trading Ltd
	"e4:5f:01", // Raspberry Pi 4 and newer
	"28:cd:c1", // Raspberry Pi 400 and some Pi 4
	"d8:3a:dd", // Some Raspberry Pi models
	"2c:cf:67", // Raspberry Pi 5 and newer models
}

type device struct {
	IP  string
	MAC string
}

// Configuration for scanning
type scanConfig struct {
	timeout       time.Duration
	maxGoroutines int
	pingCount     int
}

// Checks if a MAC address prefix matches any Raspberry Pi MAC
func isRaspberryPi(mac string) bool {
	if len(mac) < 8 {
		return false
	}
	macPrefix := strings.ToLower(mac[:8])
	for _, piMAC := range raspberryPiMACs {
		if macPrefix == strings.ToLower(piMAC) {
			return true
		}
	}
	return false
}

// Checks if running as root user
func isRoot() bool {
	currentUser, err := user.Current()
	if err != nil {
		log.Printf("[Warning] Unable to get current user: %v", err)
		return false
	}
	return currentUser.Username == "root" || currentUser.Uid == "0"
}

// Writes device list to a file in user's home directory
func writeToFile(devices []device, fileName string) error {
	dirname, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	filePath := fmt.Sprintf("%s/%s", dirname, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed creating file %s: %w", filePath, err)
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	defer writer.Flush()

	for _, dev := range devices {
		_, err := writer.WriteString(fmt.Sprintf("ip:%s mac:%s\n", dev.IP, dev.MAC))
		if err != nil {
			return fmt.Errorf("failed writing to file: %w", err)
		}
	}

	return nil
}

// Pings an IP address with proper context and timeout
func pingIP(ctx context.Context, ipAddress string, config scanConfig) bool {
	pinger, err := ping.NewPinger(ipAddress)
	if err != nil {
		return false
	}
	defer pinger.Stop()

	pinger.Count = config.pingCount
	pinger.Timeout = config.timeout
	pinger.SetPrivileged(false) // Use unprivileged ICMP

	received := false
	pinger.OnRecv = func(pkt *ping.Packet) {
		received = true
	}

	// Run with context
	done := make(chan bool, 1)
	go func() {
		err = pinger.Run()
		done <- true
	}()

	select {
	case <-ctx.Done():
		pinger.Stop()
		return false
	case <-done:
		return received && err == nil
	}
}

// Scans a range of IPs concurrently with proper goroutine management
func scanIPRange(ctx context.Context, ips []string, config scanConfig) []string {
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		foundIPs  []string
		semaphore = make(chan struct{}, config.maxGoroutines)
		completed = 0
		total     = len(ips)
		lastPrint int
	)

	fmt.Printf("Scanning %d IP addresses...\n", total)

	for _, ip := range ips {
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore

		go func(ipAddr string) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore

			if pingIP(ctx, ipAddr, config) {
				mu.Lock()
				foundIPs = append(foundIPs, ipAddr)
				mu.Unlock()
			}

			mu.Lock()
			completed++
			progress := (completed * 100) / total
			if progress > lastPrint && progress%10 == 0 {
				fmt.Printf("Progress: %d%% (%d/%d)\n", progress, completed, total)
				lastPrint = progress
			}
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	close(semaphore)
	return foundIPs
}

// Parses ARP table to get MAC addresses
func parseARPTable(foundIPs []string) ([]device, []device) {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		log.Printf("Error running arp command: %v", err)
		return nil, nil
	}

	var allDevices []device
	var piDevices []device
	foundIPMap := make(map[string]bool)
	for _, ip := range foundIPs {
		foundIPMap[ip] = true
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if !strings.Contains(line, "(") || strings.Contains(line, "incomplete") {
			continue
		}

		// Parse: ? (192.168.1.1) at aa:bb:cc:dd:ee:ff on en0 ifscope [ethernet]
		parts := strings.Split(line, "(")
		if len(parts) < 2 {
			continue
		}

		ipPart := strings.Split(parts[1], ")")
		if len(ipPart) < 2 {
			continue
		}
		ip := strings.TrimSpace(ipPart[0])

		macParts := strings.Split(ipPart[1], "at")
		if len(macParts) < 2 {
			continue
		}

		macPart := strings.Split(macParts[1], "on")
		if len(macPart) < 1 {
			continue
		}
		mac := strings.TrimSpace(macPart[0])

		// Only include devices we actually pinged successfully
		if !foundIPMap[ip] {
			continue
		}

		dev := device{IP: ip, MAC: mac}
		allDevices = append(allDevices, dev)

		if isRaspberryPi(mac) {
			piDevices = append(piDevices, dev)
		}
	}

	return allDevices, piDevices
}

// Sets system resource limits for better performance
func setResourceLimits() {
	var rLimit syscall.Rlimit
	if err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Printf("Warning: Could not get resource limit: %v", err)
		return
	}

	// Set to a high but reasonable limit
	rLimit.Cur = 8192
	if rLimit.Max < rLimit.Cur {
		rLimit.Cur = rLimit.Max
	}

	if err := syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit); err != nil {
		log.Printf("Warning: Could not set resource limit: %v", err)
	}
}

// Gets the number of CPU cores
func getCPUCores() int {
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("sysctl", "-n", "hw.ncpu").Output()
		if err == nil {
			if cores, err := strconv.Atoi(strings.TrimSpace(string(out))); err == nil {
				return cores
			}
		}
	}

	cpuData, err := ghw.CPU()
	if err == nil && cpuData != nil {
		return int(cpuData.TotalCores)
	}

	return runtime.NumCPU()
}

// Generates all IPs in a /24 subnet
func generateIPRange(baseIP string) []string {
	parts := strings.Split(baseIP, ".")
	if len(parts) != 4 {
		return nil
	}

	basePrefix := strings.Join(parts[:3], ".")
	ips := make([]string, 0, 254)

	for i := 1; i <= 254; i++ {
		ips = append(ips, fmt.Sprintf("%s.%d", basePrefix, i))
	}

	return ips
}

// Gets all local IP addresses
func getLocalIPs() []string {
	var localIPs []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Printf("Error getting network interfaces: %v", err)
		return nil
	}

	for _, address := range addrs {
		if ipnet, ok := address.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				localIPs = append(localIPs, ipnet.IP.String())
			}
		}
	}

	return localIPs
}

// Converts IP to CIDR notation
func ipToCIDR(ip string) string {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return ip
	}
	return fmt.Sprintf("%s.%s.%s.0/24", parts[0], parts[1], parts[2])
}

func main() {
	fmt.Println("=== Raspberry Pi Network Scanner ===")
	fmt.Println()

	// Set resource limits for better performance
	setResourceLimits()

	// Get local IP addresses
	localIPs := getLocalIPs()
	if len(localIPs) == 0 {
		log.Fatal("No network interfaces found")
	}

	// Display available networks
	fmt.Println("Available networks:")
	for i, ip := range localIPs {
		fmt.Printf("  [%d] %s\n", i, ipToCIDR(ip))
	}
	fmt.Println()

	// Get CPU info
	cores := getCPUCores()
	fmt.Printf("System: %d CPU cores available\n", cores)
	fmt.Println()

	// Get user selection
	fmt.Print("Select network to scan [0]: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := strings.TrimSpace(scanner.Text())

	selection := 0
	if input != "" {
		var err error
		selection, err = strconv.Atoi(input)
		if err != nil || selection < 0 || selection >= len(localIPs) {
			log.Fatal("Invalid selection")
		}
	}

	selectedIP := localIPs[selection]
	fmt.Printf("Scanning network: %s\n\n", ipToCIDR(selectedIP))

	// Configure scan parameters
	config := scanConfig{
		timeout:       time.Millisecond * 500,
		maxGoroutines: cores * 32, // Balanced for network I/O
		pingCount:     1,
	}

	// Generate IP range
	ips := generateIPRange(selectedIP)
	if len(ips) == 0 {
		log.Fatal("Failed to generate IP range")
	}

	// Start scanning
	startTime := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	foundIPs := scanIPRange(ctx, ips, config)
	fmt.Printf("\nFound %d active devices\n", len(foundIPs))

	// Parse ARP table
	fmt.Println("Parsing ARP table...")
	allDevices, piDevices := parseARPTable(foundIPs)

	// Save results
	if len(allDevices) > 0 {
		if err := writeToFile(allDevices, "devicesfound.txt"); err != nil {
			log.Printf("Warning: Failed to save all devices: %v", err)
		} else {
			fmt.Printf("Saved %d devices to ~/devicesfound.txt\n", len(allDevices))
		}
	}

	if len(piDevices) > 0 {
		if err := writeToFile(piDevices, "pilist.txt"); err != nil {
			log.Printf("Warning: Failed to save Pi list: %v", err)
		} else {
			fmt.Printf("Saved %d Raspberry Pi devices to ~/pilist.txt\n", len(piDevices))
		}

		fmt.Println("\nRaspberry Pi devices found:")
		for _, pi := range piDevices {
			fmt.Printf("  • %s [%s]\n", pi.IP, pi.MAC)
		}
	} else {
		fmt.Println("\nNo Raspberry Pi devices found on this network")
	}

	duration := time.Since(startTime)
	fmt.Printf("\nScan completed in %.2f seconds\n", duration.Seconds())
}
