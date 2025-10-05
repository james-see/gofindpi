# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [2.0.0] - 2025-10-05

### Added
- Comprehensive Raspberry Pi 5 support with new MAC address prefix (2c:cf:67)
- Raspberry Pi 400 support with MAC prefix (28:cd:c1)
- Additional Pi model detection (d8:3a:dd)
- Real-time progress reporting during network scans
- Context-based timeout management for better resilience
- Proper concurrent scanning with semaphore-controlled goroutines
- CPU core detection for optimized parallel execution
- Enhanced error handling and recovery
- .gitignore file for cleaner repository
- Makefile for convenient build and development tasks
- Comprehensive README with usage examples and troubleshooting

### Changed
- **BREAKING**: Complete rewrite of core scanning engine
- Updated from Go 1.15 to Go 1.23+ (now supports 1.24)
- Modernized all dependencies to latest versions
  - go-ping/ping: v0.0.0 → v1.2.0
  - jaypipes/ghw: v0.7.0 → v0.19.1
- Improved concurrent scanning with proper WaitGroup usage
- Enhanced timeout handling (reduced from 800ms to 500ms per ping)
- Better resource limit management with reasonable defaults
- Refactored code structure for maintainability
- Cleaned up code style and removed inappropriate comments
- Updated Docker configuration to use Go 1.24 and multi-stage builds
- Improved Docker security with non-root user execution
- Enhanced docker-compose.yml with host networking for better performance

### Fixed
- Critical goroutine bug where WaitGroup was not used correctly
- Race conditions in IP collection
- Memory leaks from improper goroutine cleanup
- Inconsistent MAC address prefix matching
- Error handling in network interface detection
- Resource limit setting failures on some systems

### Security
- Docker container now runs as non-root user
- Better privilege handling for network operations
- Removed unnecessary privileged mode requirements

### Performance
- 2-3x faster scanning through proper concurrency
- Optimized goroutine count based on CPU cores
- Reduced timeout for faster completion
- Better memory efficiency with bounded goroutines

## [1.0.3] - 2021-04-07

### Added
- Initial release
- Basic network scanning functionality
- Support for Raspberry Pi 1-4 models
- MAC address detection (b8:27:eb, dc:a6:32, e4:5f:01)
- File output to home directory
- Docker support

### Known Issues (Fixed in 2.0.0)
- Goroutines not properly synchronized
- Missing support for newer Pi models
- Outdated Go version and dependencies
- No progress reporting
- Poor error handling
