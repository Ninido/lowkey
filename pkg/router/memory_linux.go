//go:build linux

package router

import (
	"os"
	"strconv"
	"strings"
)

// GetTotalMemoryBytes returns the total system memory in bytes
func GetTotalMemoryBytes() (int64, error) {
	return getMemoryBytes("/proc/meminfo", "MemTotal")
}

// GetAvailableMemoryBytes returns the available system memory in bytes
func GetAvailableMemoryBytes() (int64, error) {
	return getMemoryBytes("/proc/meminfo", "MemAvailable")
}

func getMemoryBytes(path, key string) (int64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	for _, line := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(line, key) {
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}
			// Parse kB to bytes
			kbStr := strings.TrimSpace(parts[1])
			// First word is the number
			kbStr = strings.Fields(kbStr)[0]
			kb, err := strconv.ParseInt(kbStr, 10, 64)
			if err != nil {
				return 0, err
			}
			return kb * 1024, nil
		}
	}
	return 0, nil
}
