# gofindpi

Fast, concurrent Raspberry Pi network scanner written in Go. Scans your local network to discover all Raspberry Pi devices using MAC address detection.

## Features

- **Fast Concurrent Scanning**: Parallel ping scanning with optimized goroutine management
- **Comprehensive Pi Detection**: Detects all Raspberry Pi models including Pi 5, Pi 400, Pi 4, Pi Zero 2 W, and older models
- **Resilient**: Proper timeout handling, context management, and error recovery
- **Cross-Platform**: Works on Linux, macOS, and Windows
- **Low Resource Usage**: Efficient memory and CPU utilization
- **Progress Reporting**: Real-time scan progress updates

## Supported Raspberry Pi Models

Detects all Raspberry Pi models by their MAC address OUI prefixes:
- Raspberry Pi 5 and newer
- Raspberry Pi 400
- Raspberry Pi 4 (all variants)
- Raspberry Pi 3 (all variants)
- Raspberry Pi Zero 2 W
- Raspberry Pi 2
- Raspberry Pi 1 and Zero

## Installation

### From Source

```bash
go install github.com/james-see/gofindpi@latest
```

### Build Locally

```bash
git clone https://github.com/james-see/gofindpi.git
cd gofindpi
go build -o gofindpi
```

### Multi-Platform Build

```bash
./scripts/build.sh all
```

This creates binaries for:
- macOS (Intel & Apple Silicon)
- Linux (amd64 & arm64)

## Usage

### Basic Usage

Simply run the binary:

```bash
./gofindpi
```

The scanner will:
1. Display available network interfaces
2. Ask you to select which network to scan
3. Scan all IPs in the selected /24 subnet
4. Display found Raspberry Pi devices
5. Save results to your home directory

### Output Files

Two files are created in your home directory:
- `~/devicesfound.txt` - All devices found on the network
- `~/pilist.txt` - Only Raspberry Pi devices

### Example Output

```
=== Raspberry Pi Network Scanner ===

Available networks:
  [0] 192.168.1.0/24
  [1] 172.16.0.0/24

System: 8 CPU cores available

Select network to scan [0]: 0
Scanning network: 192.168.1.0/24

Scanning 254 IP addresses...
Progress: 50% (127/254)
Progress: 100% (254/254)

Found 12 active devices
Parsing ARP table...
Saved 12 devices to ~/devicesfound.txt
Saved 3 Raspberry Pi devices to ~/pilist.txt

Raspberry Pi devices found:
  • 192.168.1.100 [b8:27:eb:a1:b2:c3]
  • 192.168.1.105 [dc:a6:32:d4:e5:f6]
  • 192.168.1.110 [e4:5f:01:aa:bb:cc]

Scan completed in 2.34 seconds
```

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
- **gofindpi**: ~1-3 seconds (depending on network and CPU cores)
- nmap: ~5-6 seconds
- Traditional sequential scan: ~20-30 seconds

Performance scales with CPU cores and network latency.

## How It Works

1. **Network Discovery**: Identifies local network interfaces and their subnets
2. **Concurrent Ping**: Sends ICMP echo requests to all IPs in parallel using goroutines
3. **ARP Resolution**: Parses the system ARP table for MAC addresses
4. **MAC Matching**: Compares MAC address prefixes against known Raspberry Pi OUIs
5. **Results**: Saves and displays all Pi devices found

## Configuration

Scan parameters can be adjusted in the code:

```go
config := scanConfig{
    timeout:       time.Millisecond * 500,  // Ping timeout
    maxGoroutines: cores * 32,              // Concurrent goroutines
    pingCount:     1,                        // Pings per IP
}
```

## Troubleshooting

### "No devices found"
- Ensure devices are powered on and connected
- Try increasing the timeout value
- Check firewall settings aren't blocking ICMP

### "Permission denied" on Linux
Some systems require privileges for raw sockets. The scanner uses unprivileged ICMP by default, but if issues persist:
```bash
sudo setcap cap_net_raw+ep ./gofindpi
```

### Slow scanning
- Check network congestion
- Reduce `maxGoroutines` if experiencing system slowdown
- Increase `timeout` for slower networks

## Development

### Run Tests

```bash
go test ./...
```

### Lint Code

```bash
./scripts/lint.sh
```

Requires:
- `staticcheck`: `go install honnef.co/go/tools/cmd/staticcheck@latest`
- `golangci-lint`: https://golangci-lint.run/usage/install/

### Build

```bash
./scripts/build.sh
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

## Changelog

### v2.0.0 (2025)
- Updated to Go 1.23+
- Complete rewrite with proper concurrent scanning
- Added support for Raspberry Pi 5 and newer models
- Enhanced error handling and resilience
- Improved performance with optimized goroutine management
- Added progress reporting
- Better cross-platform compatibility
- Modern Docker support

### v1.0.3 (2021)
- Initial release
- Basic network scanning
- Support for Pi 1-4