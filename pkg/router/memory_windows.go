//go:build windows

package router

import (
	"os/exec"
	"strconv"
	"strings"
)

// GetTotalMemoryBytes returns the total system memory in bytes
func GetTotalMemoryBytes() (int64, error) {
	// Use WMI via PowerShell
	cmd := exec.Command("powershell", "-Command",
		"Get-CimInstance Win32_OperatingSystem | Select-Object -ExpandProperty TotalVisibleMemorySize")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	// Result is in KB
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}
	return kb * 1024, nil
}

// GetAvailableMemoryBytes returns the available system memory in bytes
func GetAvailableMemoryBytes() (int64, error) {
	// Use WMI via PowerShell
	cmd := exec.Command("powershell", "-Command",
		"Get-CimInstance Win32_OperatingSystem | Select-Object -ExpandProperty FreePhysicalMemory")
	out, err := cmd.Output()
	if err != nil {
		return 0, err
	}
	// Result is in KB
	kb, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0, err
	}
	return kb * 1024, nil
}
