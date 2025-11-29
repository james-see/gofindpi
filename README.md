# gofindpi

Fast, concurrent network device scanner written in Go. Scans your local network to identify **all devices** with manufacturer detection using an embedded OUI database of 38,000+ entries.

## Features

- **Full Device Identification**: Identifies all network devices with manufacturer name and category
- **Embedded OUI Database**: 38,000+ manufacturer entries compiled directly into the binary
- **Raspberry Pi Detection**: Special detection for all Raspberry Pi models (Pi 1-5, Pi 400, Zero)
- **Fast Concurrent Scanning**: Parallel ping scanning with optimized goroutine management
- **Multiple Output Formats**: Text files and structured JSON output
- **Statistics & Analytics**: Manufacturer breakdown and device category statistics
- **Hostname Resolution**: Optionally resolves hostnames via reverse DNS
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Self-Contained**: No external dependencies at runtime - everything embedded in binary

## Device Categories

The scanner identifies devices into categories:
- **Raspberry Pi**: All Pi models (special detection)
- **Network Equipment**: Cisco, Ubiquiti, Netgear, TP-Link, etc.
- **Computer/Phone**: Apple, Dell, Lenovo, Samsung, etc.
- **IoT/Smart Home**: Amazon devices, Ring, Nest, Philips Hue, etc.
- **TV/Display**: LG, Samsung, Roku, Vizio, etc.
- **Security Camera**: Hikvision, Dahua, Arlo, etc.
- **Printer**: HP, Canon, Epson, Brother, etc.
- **Gaming**: Nintendo, PlayStation, Xbox
- **Storage/NAS**: Synology, QNAP
- And many more...

## Installation

### Homebrew (macOS/Linux)

```bash
brew tap james-see/tap
brew install gofindpi
```

### Go Install

```bash
go install github.com/james-see/gofindpi@latest
```

### Download Binary

Download pre-built binaries from the [Releases page](https://github.com/james-see/gofindpi/releases).

### Build Locally

```bash
git clone https://github.com/james-see/gofindpi.git
cd gofindpi
make build
```

### Multi-Platform Build

```bash
make build-all
```

Creates binaries for:
- macOS (Intel & Apple Silicon)
- Linux (amd64, arm64, arm)
- Windows (amd64)

## Usage

### Basic Usage

```bash
./gofindpi
```

The scanner will:
1. Display available network interfaces
2. Ask you to select which network to scan
3. Scan all IPs in the selected /24 subnet
4. Identify each device with manufacturer and category
5. Display statistics and save results

### Example Output

The scanner features a modern TUI with color-coded output, progress bars, and visual statistics:

```
╔══════════════════════════════════════════════════════════════╗
║                    NETWORK DEVICE SCANNER                    ║
║        v2.0.0 • Manufacturer Detection • OUI Database        ║
╚══════════════════════════════════════════════════════════════╝

── AVAILABLE NETWORKS ────────────────────────────────────────
  [0] 192.168.1.0/24

── SYSTEM INFO ───────────────────────────────────────────────
  ● CPU Cores: 8
  ● OUI Database: 38203 entries

  Select network to scan [0]: 0

── SCANNING: 192.168.1.0/24 ──────────────────────────────────
  Scanning 254 addresses...

  [████████████████████████████████████████] 100% (254/254)

  ✓ Found 15 active devices
  → Identifying manufacturers...

── OUTPUT FILES ──────────────────────────────────────────────
  ✓ ~/devicesfound.txt (15 devices)
  ✓ ~/devicesfound.json (full scan data)
  ✓ ~/pilist.txt (3 Raspberry Pi)

── 🍓 RASPBERRY PI DEVICES ───────────────────────────────────
  ● 192.168.1.100 (raspberrypi.local) [b8:27:eb:a1:b2:c3]
  ● 192.168.1.105 (pi4.local) [dc:a6:32:d4:e5:f6]
  ● 192.168.1.110 (pi5.local) [2c:cf:67:aa:bb:cc]

── SCAN RESULTS ──────────────────────────────────────────────

  ┌─────────────────────┬─────────────────────┐
  │  TOTAL DEVICES      │  RASPBERRY PI       │
  │         15          │          3          │
  └─────────────────────┴─────────────────────┘

── MANUFACTURERS ─────────────────────────────────────────────
  Apple, Inc.                         ████████████████████  4
  Raspberry Pi Trading Ltd            ███████████████░░░░░  3
  Amazon Technologies Inc.            ██████████░░░░░░░░░░  2
  ...

── DEVICE CATEGORIES ─────────────────────────────────────────
  💻 Computer/Phone       5
  🍓 Raspberry Pi         3
  🏠 IoT/Smart Home       2
  📺 TV/Streaming         2
  ...

────────────────────────────────────────────────────────────
  Scan completed in 2.34 seconds
```

### Output Files

Three files are created in your home directory:

**1. `~/devicesfound.txt`** - All devices (text format)
```
ip:192.168.1.100 mac:b8:27:eb:a1:b2:c3 manufacturer:Raspberry Pi Foundation category:Raspberry Pi hostname:raspberrypi.local [Raspberry Pi]
ip:192.168.1.105 mac:dc:a6:32:d4:e5:f6 manufacturer:Apple, Inc. category:Computer/Phone hostname:macbook.local
```

**2. `~/devicesfound.json`** - Complete results (JSON format)
```json
{
  "timestamp": "2025-10-05T14:30:00Z",
  "network": "192.168.1.0/24",
  "duration_seconds": 2.34,
  "total_devices": 15,
  "raspberry_pi_count": 3,
  "devices": [
    {
      "ip": "192.168.1.100",
      "mac": "b8:27:eb:a1:b2:c3",
      "manufacturer": "Raspberry Pi Foundation",
      "category": "Raspberry Pi",
      "is_raspberry_pi": true,
      "hostname": "raspberrypi.local"
    }
  ],
  "manufacturer_statistics": {
    "Raspberry Pi Foundation": 3,
    "Apple, Inc.": 4
  },
  "category_statistics": {
    "Raspberry Pi": 3,
    "Computer/Phone": 5
  }
}
```

**3. `~/pilist.txt`** - Raspberry Pi devices only (text format)

## OUI Database

The scanner includes an embedded OUI (Organizationally Unique Identifier) database containing 38,000+ manufacturer entries sourced from:
- Wireshark manufacturer database
- IEEE OUI registry

### Updating the OUI Database

To refresh the OUI database with the latest manufacturer entries:

```bash
make update-oui
```

This downloads the latest data from Wireshark and regenerates `data/oui.go`.

### View OUI Statistics

```bash
make oui-stats
```

## Supported Raspberry Pi Models

Detects all Raspberry Pi models by their MAC address OUI prefixes:
- `b8:27:eb` - Original Raspberry Pi Foundation
- `dc:a6:32` - Raspberry Pi Trading Ltd
- `e4:5f:01` - Raspberry Pi 4 and newer
- `28:cd:c1` - Raspberry Pi 400 and some Pi 4
- `d8:3a:dd` - Various Raspberry Pi models
- `2c:cf:67` - Raspberry Pi 5 and newest models

## Docker Usage

### Build Image

```bash
docker-compose build
```

### Run Scanner

```bash
docker-compose run --rm gofindpi
```

**Note**: The container uses host networking mode to access your local network.

## Requirements

- Go 1.23+ (for building from source)
- Network access
- ARP command availability (standard on Unix systems)

## Performance

Typical scan times for a /24 subnet (254 addresses):
- **gofindpi**: ~2-4 seconds (with device identification)
- nmap: ~5-6 seconds
- Traditional sequential scan: ~20-30 seconds

Performance scales with CPU cores and network latency.

## How It Works

1. **Network Discovery**: Identifies local network interfaces and their subnets
2. **Concurrent Ping**: Sends ICMP echo requests to all IPs in parallel using goroutines
3. **ARP Resolution**: Parses the system ARP table for MAC addresses
4. **OUI Lookup**: Cross-references MAC prefixes against 38k+ manufacturer database
5. **Categorization**: Assigns device categories based on manufacturer
6. **Hostname Resolution**: Optionally resolves hostnames via reverse DNS
7. **Results**: Outputs text files, JSON, and statistics

## Development

### Run Tests

```bash
go test ./...
```

### Lint Code

```bash
make lint
```

### Build

```bash
make build
```

### Update Dependencies

```bash
make mod-upgrade
```

## Make Targets

```
make help          # Show all available targets
make build         # Build the binary
make build-all     # Build for all platforms
make clean         # Remove build artifacts
make run           # Build and run
make lint          # Run linters
make deps          # Update dependencies
make update-oui    # Refresh OUI database
make oui-stats     # Show OUI database statistics
make docker-build  # Build Docker image
make docker-run    # Run in Docker
```

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

See [LICENSE](LICENSE) file for details.

## Acknowledgments

Built with:
- [go-ping/ping](https://github.com/go-ping/ping) - ICMP ping library
- [jaypipes/ghw](https://github.com/jaypipes/ghw) - Hardware information library
- [Wireshark](https://www.wireshark.org/) - OUI database source

## Changelog

### v2.0.0 (2025)
- **Major Enhancement**: Full device identification with manufacturer detection
- Added embedded OUI database with 38,000+ entries
- Added device category classification
- Added JSON output with scan metadata and statistics
- Enhanced text output with manufacturer and category info
- Added manufacturer and category statistics display
- Added hostname resolution via reverse DNS
- Added OUI database generator script
- Updated to Go 1.23+ with latest dependencies
- Complete codebase modernization

### v1.0.3 (2021)
- Initial release
- Basic Raspberry Pi detection
- Support for Pi 1-4 models