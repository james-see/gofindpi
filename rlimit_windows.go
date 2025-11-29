//go:build windows

package main

// setResourceLimits is a no-op on Windows (resource limits handled differently)
func setResourceLimits() {
	// Windows doesn't use RLIMIT_NOFILE - file handle limits are managed differently
}

