package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-ping/ping"
	"github.com/james-see/gofindpi/data"
	"github.com/jaypipes/ghw"
)

// Build-time variables (set by goreleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// ANSI color codes for TUI
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorWhite   = "\033[37m"
	colorBold    = "\033[1m"
	colorDim     = "\033[2m"

	// Bright colors
	colorBrightGreen  = "\033[92m"
	colorBrightYellow = "\033[93m"
	colorBrightCyan   = "\033[96m"
	colorBrightWhite  = "\033[97m"
)

// Box drawing characters
const (
	boxTopLeft     = "╔"
	boxTopRight    = "╗"
	boxBottomLeft  = "╚"
	boxBottomRight = "╝"
	boxHorizontal  = "═"
	boxVertical    = "║"
	boxTeeRight    = "╠"
	boxTeeLeft     = "╣"
	boxCross       = "╬"

	// Single line
	lineHorizontal  = "─"
	lineVertical    = "│"
	lineTeeDown     = "┬"
	lineTeeUp       = "┴"
	lineTeeRight    = "├"
	lineTeeLeft     = "┤"
	lineTopLeft     = "┌"
	lineTopRight    = "┐"
	lineBottomLeft  = "└"
	lineBottomRight = "┘"

	// Bullets and symbols
	bullet       = "●"
	bulletHollow = "○"
	checkMark    = "✓"
	crossMark    = "✗"
	arrowRight   = "→"
	piSymbol     = "🍓"
)

// Device represents a discovered network device with full identification
type Device struct {
	IP            string `json:"ip"`
	MAC           string `json:"mac"`
	Manufacturer  string `json:"manufacturer"`
	Category      string `json:"category"`
	IsRaspberryPi bool   `json:"is_raspberry_pi"`
	Hostname      string `json:"hostname,omitempty"`
}

// ScanResult contains the complete scan results with metadata
type ScanResult struct {
	Timestamp    string         `json:"timestamp"`
	Network      string         `json:"network"`
	Duration     float64        `json:"duration_seconds"`
	TotalDevices int            `json:"total_devices"`
	PiCount      int            `json:"raspberry_pi_count"`
	Devices      []Device       `json:"devices"`
	Statistics   map[string]int `json:"manufacturer_statistics"`
	Categories   map[string]int `json:"category_statistics"`
}

// Configuration for scanning
type scanConfig struct {
	timeout       time.Duration
	maxGoroutines int
	pingCount     int
}

// printHeader displays the application header
func printHeader() {
	width := 62
	title := "NETWORK DEVICE SCANNER"
	subtitle := fmt.Sprintf("v%s • Manufacturer Detection • OUI Database", version)

	fmt.Println()
	fmt.Printf("%s%s%s%s%s\n", colorCyan, boxTopLeft, strings.Repeat(boxHorizontal, width), boxTopRight, colorReset)
	fmt.Printf("%s%s%s %s%-*s%s %s%s\n", colorCyan, boxVertical, colorReset, colorBold+colorBrightCyan, width-2, centerText(title, width-2), colorReset, colorCyan, boxVertical+colorReset)
	fmt.Printf("%s%s%s %-*s %s%s\n", colorCyan, boxVertical, colorReset, width-2, centerText(subtitle, width-2), colorCyan, boxVertical+colorReset)
	fmt.Printf("%s%s%s%s%s\n", colorCyan, boxBottomLeft, strings.Repeat(boxHorizontal, width), boxBottomRight, colorReset)
	fmt.Println()
}

// centerText centers text within a given width
func centerText(text string, width int) string {
	if len(text) >= width {
		return text
	}
	padding := (width - len(text)) / 2
	return strings.Repeat(" ", padding) + text
}

// printSection prints a section header
func printSection(title string) {
	fmt.Printf("\n%s%s %s %s%s\n", colorBold+colorYellow, lineHorizontal+lineHorizontal, title, strings.Repeat(lineHorizontal, 50-len(title)), colorReset)
}

// printProgressBar displays a progress bar
func printProgressBar(current, total int, width int) {
	percent := float64(current) / float64(total)
	filled := int(percent * float64(width))
	empty := width - filled

	bar := strings.Repeat("█", filled) + strings.Repeat("░", empty)
	fmt.Printf("\r  %s[%s%s%s]%s %s%3d%%%s (%d/%d)",
		colorDim, colorBrightGreen, bar, colorDim, colorReset,
		colorBrightWhite, int(percent*100), colorReset,
		current, total)
}

// lookupManufacturer retrieves manufacturer info from the OUI database
func lookupManufacturer(mac string) (data.ManufacturerInfo, bool) {
	if len(mac) < 8 {
		return data.ManufacturerInfo{Name: "Unknown", Category: "Unknown"}, false
	}

	// Normalize MAC address prefix
	macPrefix := strings.ToLower(mac[:8])

	// Ensure format is xx:xx:xx
	if !strings.Contains(macPrefix, ":") {
		// Convert formats like xxxx.xxxx to xx:xx:xx
		macPrefix = strings.ReplaceAll(macPrefix, ".", "")
		macPrefix = strings.ReplaceAll(macPrefix, "-", "")
		if len(macPrefix) >= 6 {
			macPrefix = fmt.Sprintf("%s:%s:%s", macPrefix[0:2], macPrefix[2:4], macPrefix[4:6])
		}
	}

	// Lookup in database
	if info, ok := data.OUIDatabase[macPrefix]; ok {
		return info, true
	}

	return data.ManufacturerInfo{Name: "Unknown", Category: "Unknown"}, false
}

// isRaspberryPi checks if the device is a Raspberry Pi based on MAC prefix
func isRaspberryPi(mac string) bool {
	if len(mac) < 8 {
		return false
	}
	macPrefix := strings.ToLower(mac[:8])
	return data.IsRaspberryPiOUI(macPrefix)
}

// resolveHostname attempts to get the hostname for an IP address
func resolveHostname(ip string) string {
	names, err := net.LookupAddr(ip)
	if err != nil || len(names) == 0 {
		return ""
	}
	// Remove trailing dot if present
	hostname := strings.TrimSuffix(names[0], ".")
	return hostname
}

// Writes device list to a text file in user's home directory
func writeToFile(devices []Device, fileName string) error {
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
		line := fmt.Sprintf("ip:%s mac:%s manufacturer:%s category:%s",
			dev.IP, dev.MAC, dev.Manufacturer, dev.Category)
		if dev.Hostname != "" {
			line += fmt.Sprintf(" hostname:%s", dev.Hostname)
		}
		if dev.IsRaspberryPi {
			line += " [Raspberry Pi]"
		}
		_, err := writer.WriteString(line + "\n")
		if err != nil {
			return fmt.Errorf("failed writing to file: %w", err)
		}
	}

	return nil
}

// writeJSON writes the scan results as JSON
func writeJSON(result ScanResult, fileName string) error {
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

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")

	if err := encoder.Encode(result); err != nil {
		return fmt.Errorf("failed encoding JSON: %w", err)
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
	)

	fmt.Printf("  %sScanning %d addresses...%s\n\n", colorDim, total, colorReset)

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
			printProgressBar(completed, total, 40)
			mu.Unlock()
		}(ip)
	}

	wg.Wait()
	close(semaphore)
	fmt.Println() // New line after progress bar
	return foundIPs
}

// Parses ARP table to get MAC addresses and identifies devices
func parseARPTable(foundIPs []string, resolveHosts bool) []Device {
	out, err := exec.Command("arp", "-a").Output()
	if err != nil {
		log.Printf("Error running arp command: %v", err)
		return nil
	}

	var devices []Device
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

		// Lookup manufacturer info
		info, _ := lookupManufacturer(mac)

		// Check if Raspberry Pi
		isPi := isRaspberryPi(mac)
		if isPi {
			info.Category = "Raspberry Pi"
		}

		dev := Device{
			IP:            ip,
			MAC:           mac,
			Manufacturer:  info.Name,
			Category:      info.Category,
			IsRaspberryPi: isPi,
		}

		// Optionally resolve hostname
		if resolveHosts {
			dev.Hostname = resolveHostname(ip)
		}

		devices = append(devices, dev)
	}

	return devices
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

// calculateStatistics generates manufacturer and category statistics
func calculateStatistics(devices []Device) (map[string]int, map[string]int) {
	manufacturerStats := make(map[string]int)
	categoryStats := make(map[string]int)

	for _, dev := range devices {
		manufacturerStats[dev.Manufacturer]++
		categoryStats[dev.Category]++
	}

	return manufacturerStats, categoryStats
}

// printDeviceTable prints devices in a nice table format
func printDeviceTable(devices []Device) {
	if len(devices) == 0 {
		return
	}

	fmt.Printf("\n  %s%-16s %-18s %-30s %s%s\n",
		colorBold, "IP ADDRESS", "MAC ADDRESS", "MANUFACTURER", "CATEGORY", colorReset)
	fmt.Printf("  %s%s%s\n", colorDim, strings.Repeat(lineHorizontal, 80), colorReset)

	for _, dev := range devices {
		manufacturer := dev.Manufacturer
		if len(manufacturer) > 28 {
			manufacturer = manufacturer[:25] + "..."
		}

		categoryColor := colorWhite
		switch dev.Category {
		case "Raspberry Pi":
			categoryColor = colorBrightGreen
		case "Computer/Phone", "Computer":
			categoryColor = colorBrightCyan
		case "Network Equipment":
			categoryColor = colorYellow
		case "IoT/Smart Home", "IoT/Audio", "IoT/Embedded":
			categoryColor = colorMagenta
		case "Unknown":
			categoryColor = colorDim
		}

		piIndicator := ""
		if dev.IsRaspberryPi {
			piIndicator = " " + piSymbol
		}

		fmt.Printf("  %-16s %-18s %-30s %s%-15s%s%s\n",
			dev.IP, dev.MAC, manufacturer, categoryColor, dev.Category, colorReset, piIndicator)
	}
}

// printStatistics displays a summary of the scan results
func printStatistics(devices []Device, piCount int, manufacturerStats map[string]int, categoryStats map[string]int) {
	printSection("SCAN RESULTS")

	// Summary in a cleaner format
	fmt.Println()
	fmt.Printf("  %s┌─────────────────────┬─────────────────────┐%s\n", colorCyan, colorReset)
	fmt.Printf("  %s│%s  %sTOTAL DEVICES%s      %s│%s  %sRASPBERRY PI%s       %s│%s\n",
		colorCyan, colorReset, colorBold, colorReset, colorCyan, colorReset, colorBold, colorReset, colorCyan, colorReset)
	fmt.Printf("  %s│%s       %s%4d%s          %s│%s       %s%4d%s          %s│%s\n",
		colorCyan, colorReset, colorBrightWhite+colorBold, len(devices), colorReset,
		colorCyan, colorReset, colorBrightGreen+colorBold, piCount, colorReset, colorCyan, colorReset)
	fmt.Printf("  %s└─────────────────────┴─────────────────────┘%s\n", colorCyan, colorReset)

	// Manufacturers
	printSection("MANUFACTURERS")

	type kv struct {
		Key   string
		Value int
	}
	var sortedManufacturers []kv
	for k, v := range manufacturerStats {
		sortedManufacturers = append(sortedManufacturers, kv{k, v})
	}
	sort.Slice(sortedManufacturers, func(i, j int) bool {
		return sortedManufacturers[i].Value > sortedManufacturers[j].Value
	})

	maxCount := 0
	for _, item := range sortedManufacturers {
		if item.Value > maxCount {
			maxCount = item.Value
		}
	}

	for i, item := range sortedManufacturers {
		if i >= 10 { // Show top 10
			if len(sortedManufacturers) > 10 {
				fmt.Printf("  %s... and %d more%s\n", colorDim, len(sortedManufacturers)-10, colorReset)
			}
			break
		}
		name := item.Key
		if len(name) > 35 {
			name = name[:32] + "..."
		}

		barWidth := int(float64(item.Value) / float64(maxCount) * 20)
		bar := strings.Repeat("█", barWidth) + strings.Repeat("░", 20-barWidth)

		fmt.Printf("  %s%-35s%s %s%s%s %s%2d%s\n",
			colorWhite, name, colorReset,
			colorBrightGreen, bar, colorReset,
			colorBrightWhite, item.Value, colorReset)
	}

	// Categories
	printSection("DEVICE CATEGORIES")

	var sortedCategories []kv
	for k, v := range categoryStats {
		sortedCategories = append(sortedCategories, kv{k, v})
	}
	sort.Slice(sortedCategories, func(i, j int) bool {
		return sortedCategories[i].Value > sortedCategories[j].Value
	})

	for _, item := range sortedCategories {
		icon := bullet
		color := colorWhite
		switch item.Key {
		case "Raspberry Pi":
			icon = piSymbol
			color = colorBrightGreen
		case "Computer/Phone", "Computer":
			icon = "💻"
			color = colorBrightCyan
		case "Network Equipment":
			icon = "🌐"
			color = colorYellow
		case "IoT/Smart Home", "IoT/Audio":
			icon = "🏠"
			color = colorMagenta
		case "TV/Streaming", "TV/Display", "TV":
			icon = "📺"
			color = colorBlue
		case "Phone/TV", "Phone":
			icon = "📱"
			color = colorCyan
		case "Printer":
			icon = "🖨️"
			color = colorWhite
		case "Security Camera":
			icon = "📷"
			color = colorRed
		case "Gaming":
			icon = "🎮"
			color = colorGreen
		case "Unknown":
			icon = "❓"
			color = colorDim
		}

		fmt.Printf("  %s %s%-20s%s %s%2d%s\n",
			icon, color, item.Key, colorReset,
			colorBrightWhite, item.Value, colorReset)
	}
}

func main() {
	// Check for version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("gofindpi %s (commit: %s, built: %s)\n", version, commit, date)
		os.Exit(0)
	}

	printHeader()

	// Set resource limits for better performance
	setResourceLimits()

	// Get local IP addresses
	localIPs := getLocalIPs()
	if len(localIPs) == 0 {
		log.Fatal("No network interfaces found")
	}

	// Display available networks
	printSection("AVAILABLE NETWORKS")
	for i, ip := range localIPs {
		fmt.Printf("  %s[%d]%s %s%s%s\n", colorBrightCyan, i, colorReset, colorWhite, ipToCIDR(ip), colorReset)
	}

	// Get CPU info
	cores := getCPUCores()
	printSection("SYSTEM INFO")
	fmt.Printf("  %s%s%s CPU Cores: %s%d%s\n", colorDim, bullet, colorReset, colorBrightWhite, cores, colorReset)
	fmt.Printf("  %s%s%s OUI Database: %s%d%s entries\n", colorDim, bullet, colorReset, colorBrightWhite, len(data.OUIDatabase), colorReset)

	// Get user selection
	fmt.Printf("\n  %sSelect network to scan%s [%s0%s]: ", colorYellow, colorReset, colorBrightWhite, colorReset)
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
	networkCIDR := ipToCIDR(selectedIP)

	printSection("SCANNING: " + networkCIDR)

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
	fmt.Printf("\n  %s%s%s Found %s%d%s active devices\n", colorGreen, checkMark, colorReset, colorBrightWhite, len(foundIPs), colorReset)

	// Parse ARP table and identify devices
	fmt.Printf("  %s%s%s Identifying manufacturers...\n", colorDim, arrowRight, colorReset)
	devices := parseARPTable(foundIPs, true) // Enable hostname resolution

	duration := time.Since(startTime)

	// Calculate statistics
	manufacturerStats, categoryStats := calculateStatistics(devices)

	// Filter Raspberry Pi devices
	var piDevices []Device
	for _, dev := range devices {
		if dev.IsRaspberryPi {
			piDevices = append(piDevices, dev)
		}
	}

	// Create scan result
	result := ScanResult{
		Timestamp:    time.Now().Format(time.RFC3339),
		Network:      networkCIDR,
		Duration:     duration.Seconds(),
		TotalDevices: len(devices),
		PiCount:      len(piDevices),
		Devices:      devices,
		Statistics:   manufacturerStats,
		Categories:   categoryStats,
	}

	// Save results
	printSection("OUTPUT FILES")

	if len(devices) > 0 {
		if err := writeToFile(devices, "devicesfound.txt"); err != nil {
			fmt.Printf("  %s%s%s Failed to save devices: %v\n", colorRed, crossMark, colorReset, err)
		} else {
			fmt.Printf("  %s%s%s ~/devicesfound.txt %s(%d devices)%s\n", colorGreen, checkMark, colorReset, colorDim, len(devices), colorReset)
		}
	}

	if err := writeJSON(result, "devicesfound.json"); err != nil {
		fmt.Printf("  %s%s%s Failed to save JSON: %v\n", colorRed, crossMark, colorReset, err)
	} else {
		fmt.Printf("  %s%s%s ~/devicesfound.json %s(full scan data)%s\n", colorGreen, checkMark, colorReset, colorDim, colorReset)
	}

	if len(piDevices) > 0 {
		if err := writeToFile(piDevices, "pilist.txt"); err != nil {
			fmt.Printf("  %s%s%s Failed to save Pi list: %v\n", colorRed, crossMark, colorReset, err)
		} else {
			fmt.Printf("  %s%s%s ~/pilist.txt %s(%d Raspberry Pi)%s\n", colorGreen, checkMark, colorReset, colorDim, len(piDevices), colorReset)
		}
	}

	// Print device table (top 15)
	if len(devices) > 0 {
		printSection("DISCOVERED DEVICES")
		if len(devices) > 15 {
			printDeviceTable(devices[:15])
			fmt.Printf("\n  %s... and %d more devices (see devicesfound.txt)%s\n", colorDim, len(devices)-15, colorReset)
		} else {
			printDeviceTable(devices)
		}
	}

	// Show Raspberry Pi devices prominently if found
	if len(piDevices) > 0 {
		printSection(piSymbol + " RASPBERRY PI DEVICES")
		for _, pi := range piDevices {
			hostInfo := ""
			if pi.Hostname != "" {
				hostInfo = fmt.Sprintf(" %s(%s)%s", colorDim, pi.Hostname, colorReset)
			}
			fmt.Printf("  %s%s%s %s%s%s%s %s[%s]%s\n",
				colorBrightGreen, bullet, colorReset,
				colorBrightWhite, pi.IP, colorReset, hostInfo,
				colorDim, pi.MAC, colorReset)
		}
	}

	// Print statistics
	printStatistics(devices, len(piDevices), manufacturerStats, categoryStats)

	// Footer
	fmt.Printf("\n%s%s%s\n", colorDim, strings.Repeat(lineHorizontal, 64), colorReset)
	fmt.Printf("  %sScan completed in %s%.2f seconds%s\n", colorDim, colorBrightWhite, duration.Seconds(), colorReset)
	fmt.Println()
}
