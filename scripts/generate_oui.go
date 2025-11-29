//go:build ignore

// This script generates the embedded OUI database from external sources.
// Run with: go run scripts/generate_oui.go
//
// Sources:
// - IEEE OUI database: https://standards-oui.ieee.org/oui/oui.csv
// - Wireshark manuf: https://www.wireshark.org/download/automated/data/manuf

package main

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"
)

type OUIEntry struct {
	Prefix       string
	Manufacturer string
	Category     string
}

// Category mappings based on manufacturer names
var categoryKeywords = map[string]string{
	// Raspberry Pi
	"raspberry":      "Raspberry Pi",
	"raspberry pi":   "Raspberry Pi",
	// Network Equipment
	"cisco":          "Network Equipment",
	"juniper":        "Network Equipment",
	"arista":         "Network Equipment",
	"ubiquiti":       "Network Equipment",
	"mikrotik":       "Network Equipment",
	"netgear":        "Network Equipment",
	"tp-link":        "Network Equipment",
	"d-link":         "Network Equipment",
	"linksys":        "Network Equipment",
	"belkin":         "Network Equipment",
	"zyxel":          "Network Equipment",
	"aruba":          "Network Equipment",
	"fortinet":       "Network Equipment",
	"palo alto":      "Network Equipment",
	"sonicwall":      "Network Equipment",
	"meraki":         "Network Equipment",
	"ruckus":         "Network Equipment",
	"extreme":        "Network Equipment",
	"brocade":        "Network Equipment",
	"alcatel":        "Network Equipment",
	// Computers
	"apple":          "Computer/Phone",
	"dell":           "Computer",
	"hp":             "Computer",
	"hewlett":        "Computer",
	"lenovo":         "Computer",
	"intel":          "Computer",
	"microsoft":      "Computer",
	"acer":           "Computer",
	"asus":           "Computer/Network",
	// Phones/Mobile
	"samsung":        "Phone/TV",
	"xiaomi":         "Phone/IoT",
	"huawei":         "Phone/Network",
	"oneplus":        "Phone",
	"oppo":           "Phone",
	"vivo":           "Phone",
	"motorola":       "Phone",
	"nokia":          "Phone",
	"sony":           "Phone/TV",
	"google":         "Phone/IoT",
	// IoT/Smart Home
	"amazon":         "IoT/Smart Home",
	"ring":           "IoT/Smart Home",
	"nest":           "IoT/Smart Home",
	"philips":        "IoT/Smart Home",
	"sonos":          "IoT/Audio",
	"ecobee":         "IoT/Smart Home",
	"wyze":           "IoT/Smart Home",
	"tuya":           "IoT/Smart Home",
	"shelly":         "IoT/Smart Home",
	"espressif":      "IoT/Embedded",
	"arduino":        "IoT/Embedded",
	// TV/Entertainment
	"lg":             "TV/Display",
	"vizio":          "TV",
	"tcl":            "TV",
	"roku":           "TV/Streaming",
	// Gaming
	"nintendo":       "Gaming",
	"playstation":    "Gaming",
	"xbox":           "Gaming",
	"valve":          "Gaming",
	// Printers
	"canon":          "Printer/Camera",
	"epson":          "Printer",
	"brother":        "Printer",
	"xerox":          "Printer",
	"lexmark":        "Printer",
	// Security/Cameras
	"hikvision":      "Security Camera",
	"dahua":          "Security Camera",
	"axis":           "Security Camera",
	"lorex":          "Security Camera",
	"arlo":           "Security Camera",
	// Storage
	"synology":       "Storage/NAS",
	"qnap":           "Storage/NAS",
	"western digital":"Storage",
	"seagate":        "Storage",
	// Virtual/Cloud
	"vmware":         "Virtual",
	"xensource":      "Virtual",
	"parallels":      "Virtual",
}

func categorizeManufacturer(name string) string {
	nameLower := strings.ToLower(name)
	for keyword, category := range categoryKeywords {
		if strings.Contains(nameLower, keyword) {
			return category
		}
	}
	return "Unknown"
}

func normalizeOUI(oui string) string {
	// Remove common separators and convert to lowercase
	oui = strings.ToLower(oui)
	oui = strings.ReplaceAll(oui, "-", "")
	oui = strings.ReplaceAll(oui, ":", "")
	oui = strings.ReplaceAll(oui, ".", "")
	
	// Take only first 6 hex chars (3 bytes)
	if len(oui) >= 6 {
		oui = oui[:6]
	}
	
	// Format as xx:xx:xx
	if len(oui) == 6 {
		return fmt.Sprintf("%s:%s:%s", oui[0:2], oui[2:4], oui[4:6])
	}
	return oui
}

func downloadIEEEOUI() ([]OUIEntry, error) {
	fmt.Println("Downloading IEEE OUI database...")
	
	resp, err := http.Get("https://standards-oui.ieee.org/oui/oui.csv")
	if err != nil {
		return nil, fmt.Errorf("failed to download IEEE OUI: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	reader := csv.NewReader(resp.Body)
	var entries []OUIEntry
	
	// Skip header
	_, err = reader.Read()
	if err != nil {
		return nil, err
	}
	
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue // Skip malformed lines
		}
		
		if len(record) < 3 {
			continue
		}
		
		// IEEE CSV format: Registry, Assignment, Organization Name, ...
		oui := normalizeOUI(record[1])
		manufacturer := strings.TrimSpace(record[2])
		
		if oui == "" || manufacturer == "" {
			continue
		}
		
		entry := OUIEntry{
			Prefix:       oui,
			Manufacturer: manufacturer,
			Category:     categorizeManufacturer(manufacturer),
		}
		entries = append(entries, entry)
	}
	
	fmt.Printf("Downloaded %d entries from IEEE\n", len(entries))
	return entries, nil
}

func downloadWiresharkManuf() ([]OUIEntry, error) {
	fmt.Println("Downloading Wireshark manuf database...")
	
	resp, err := http.Get("https://www.wireshark.org/download/automated/data/manuf")
	if err != nil {
		return nil, fmt.Errorf("failed to download Wireshark manuf: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}
	
	var entries []OUIEntry
	scanner := bufio.NewScanner(resp.Body)
	
	// Pattern: OUI<tab>Short Name<tab>Long Name
	ouiPattern := regexp.MustCompile(`^([0-9A-Fa-f]{2}:[0-9A-Fa-f]{2}:[0-9A-Fa-f]{2})\s+(\S+)\s*(.*)$`)
	
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		
		matches := ouiPattern.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		
		oui := strings.ToLower(matches[1])
		shortName := matches[2]
		longName := strings.TrimSpace(matches[3])
		
		manufacturer := longName
		if manufacturer == "" {
			manufacturer = shortName
		}
		
		entry := OUIEntry{
			Prefix:       oui,
			Manufacturer: manufacturer,
			Category:     categorizeManufacturer(manufacturer),
		}
		entries = append(entries, entry)
	}
	
	fmt.Printf("Downloaded %d entries from Wireshark\n", len(entries))
	return entries, nil
}

func generateGoFile(entries []OUIEntry, outputPath string) error {
	// Deduplicate by OUI prefix
	seen := make(map[string]bool)
	var unique []OUIEntry
	for _, e := range entries {
		if !seen[e.Prefix] {
			seen[e.Prefix] = true
			unique = append(unique, e)
		}
	}
	
	// Sort by prefix
	sort.Slice(unique, func(i, j int) bool {
		return unique[i].Prefix < unique[j].Prefix
	})
	
	file, err := os.Create(outputPath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	w := bufio.NewWriter(file)
	
	// Write header
	fmt.Fprintf(w, `// Code generated by scripts/generate_oui.go; DO NOT EDIT.
// Generated: %s
// Total entries: %d

package data

// ManufacturerInfo contains information about a device manufacturer
type ManufacturerInfo struct {
	Name     string
	Category string
}

// OUIDatabase maps MAC address prefixes (OUI) to manufacturer information
// Format: "xx:xx:xx" -> ManufacturerInfo
var OUIDatabase = map[string]ManufacturerInfo{
`, time.Now().Format(time.RFC3339), len(unique))
	
	for _, entry := range unique {
		// Escape quotes in manufacturer names
		name := strings.ReplaceAll(entry.Manufacturer, `"`, `\"`)
		name = strings.ReplaceAll(name, "\n", " ")
		name = strings.ReplaceAll(name, "\r", "")
		
		fmt.Fprintf(w, "\t%q: {Name: %q, Category: %q},\n", 
			entry.Prefix, name, entry.Category)
	}
	
	fmt.Fprintln(w, "}")
	
	// Add Raspberry Pi specific entries to ensure they're included
	fmt.Fprintln(w, `
// RaspberryPiOUIs contains all known Raspberry Pi MAC prefixes
var RaspberryPiOUIs = []string{
	"b8:27:eb", // Original Raspberry Pi Foundation
	"dc:a6:32", // Raspberry Pi Trading Ltd
	"e4:5f:01", // Raspberry Pi 4 and newer
	"28:cd:c1", // Raspberry Pi 400 and some Pi 4
	"d8:3a:dd", // Some Raspberry Pi models
	"2c:cf:67", // Raspberry Pi 5 and newer models
}

// IsRaspberryPiOUI checks if a MAC prefix belongs to a Raspberry Pi
func IsRaspberryPiOUI(prefix string) bool {
	for _, oui := range RaspberryPiOUIs {
		if prefix == oui {
			return true
		}
	}
	return false
}`)
	
	return w.Flush()
}

func main() {
	fmt.Println("OUI Database Generator")
	fmt.Println("======================")
	
	var allEntries []OUIEntry
	
	// Try IEEE first
	ieeeEntries, err := downloadIEEEOUI()
	if err != nil {
		fmt.Printf("Warning: Could not download IEEE OUI: %v\n", err)
	} else {
		allEntries = append(allEntries, ieeeEntries...)
	}
	
	// Then Wireshark for additional entries
	wsEntries, err := downloadWiresharkManuf()
	if err != nil {
		fmt.Printf("Warning: Could not download Wireshark manuf: %v\n", err)
	} else {
		allEntries = append(allEntries, wsEntries...)
	}
	
	if len(allEntries) == 0 {
		fmt.Println("Error: No OUI entries downloaded. Check network connection.")
		os.Exit(1)
	}
	
	// Add manual Raspberry Pi entries to ensure they're always present
	piEntries := []OUIEntry{
		{Prefix: "b8:27:eb", Manufacturer: "Raspberry Pi Foundation", Category: "Raspberry Pi"},
		{Prefix: "dc:a6:32", Manufacturer: "Raspberry Pi Trading Ltd", Category: "Raspberry Pi"},
		{Prefix: "e4:5f:01", Manufacturer: "Raspberry Pi Trading Ltd", Category: "Raspberry Pi"},
		{Prefix: "28:cd:c1", Manufacturer: "Raspberry Pi Trading Ltd", Category: "Raspberry Pi"},
		{Prefix: "d8:3a:dd", Manufacturer: "Raspberry Pi Trading Ltd", Category: "Raspberry Pi"},
		{Prefix: "2c:cf:67", Manufacturer: "Raspberry Pi Trading Ltd", Category: "Raspberry Pi"},
	}
	allEntries = append(piEntries, allEntries...) // Pi entries first for priority
	
	fmt.Printf("\nTotal entries collected: %d\n", len(allEntries))
	
	outputPath := "data/oui.go"
	if err := generateGoFile(allEntries, outputPath); err != nil {
		fmt.Printf("Error generating Go file: %v\n", err)
		os.Exit(1)
	}
	
	fmt.Printf("Generated: %s\n", outputPath)
	fmt.Println("Done!")
}

