//go:build !windows

package main

import (
	"log"
	"syscall"
)

// setResourceLimits sets system resource limits for better performance (Unix only)
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

