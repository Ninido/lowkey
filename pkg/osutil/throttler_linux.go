//go:build linux

package osutil

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

type LinuxThrottler struct{}

func NewOSThrottler() OSThrottler {
	return &LinuxThrottler{}
}

func (l *LinuxThrottler) ConfigureCommand(cmd *exec.Cmd) {
	// Can be configured with nice or ionice if needed
}

func (l *LinuxThrottler) OnProcessStarted(pid int) error {
	// Set nice priority to 15 (background CPU schedule)
	_ = syscall.Setpriority(syscall.PRIO_PROCESS, pid, 15)

	// Best-effort ionice idle class
	if ionice, err := exec.LookPath("ionice"); err == nil {
		_ = exec.Command(ionice, "-c3", "-p", string(rune(pid))).Run()
	}

	return nil
}

func (l *LinuxThrottler) PauseProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGSTOP)
}

func (l *LinuxThrottler) ResumeProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGCONT)
}

func (l *LinuxThrottler) IsOnBattery() bool {
	// Check sysfs power supply status
	matches, err := filepath.Glob("/sys/class/power_supply/BAT*/status")
	if err == nil {
		for _, match := range matches {
			data, err := os.ReadFile(match)
			if err == nil && strings.TrimSpace(string(data)) == "Discharging" {
				return true
			}
		}
	}
	return false
}

func (l *LinuxThrottler) StartDutyCycleLoop(ctx context.Context, pid int, profile ThermalProfile) {
	if profile.WorkTime <= 0 || profile.PauseTime <= 0 {
		return
	}

	ticker := time.NewTicker(profile.WorkTime + profile.PauseTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = l.ResumeProcess(pid)
			return
		case <-ticker.C:
			_ = l.PauseProcess(pid)
			select {
			case <-ctx.Done():
				_ = l.ResumeProcess(pid)
				return
			case <-time.After(profile.PauseTime):
				_ = l.ResumeProcess(pid)
			}
		}
	}
}

func (l *LinuxThrottler) Cleanup() {
	// No fan override reset needed on general Linux
}
