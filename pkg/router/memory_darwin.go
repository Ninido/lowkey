//go:build darwin

package router

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetTotalMemoryBytes returns the total system memory in bytes
func GetTotalMemoryBytes() (int64, error) {
	// Use sysctl to get hw.memsize
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
}

// GetAvailableMemoryBytes returns the available system memory in bytes
func GetAvailableMemoryBytes() (int64, error) {
	// Use vm_stat to get available memory
	out, err := exec.Command("vm_stat").Output()
	if err != nil {
		return 0, err
	}

	pageSize := int64(4096)
	free := int64(0)

	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "Pages free:") {
			free = parseVmStatNumber(line)
		}
	}

	// Fallback: free pages only
	return free * pageSize, nil
}

func parseVmStatNumber(line string) int64 {
	// Format: "Pages free:          123456."
	for _, part := range strings.Split(line, ":") {
		for _, word := range strings.Fields(part) {
			// Remove trailing period
			word = strings.TrimRight(word, ".")
			if num, err := strconv.ParseInt(word, 10, 64); err == nil {
				return num
			}
		}
	}
	return 0
}
