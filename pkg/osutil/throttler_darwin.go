//go:build darwin

package osutil

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type DarwinThrottler struct{}

func NewOSThrottler() OSThrottler {
	return &DarwinThrottler{}
}

func (d *DarwinThrottler) ConfigureCommand(cmd *exec.Cmd) {
	// If taskpolicy or nice are desired, wrap binary execution or apply via OnProcessStarted
}

func (d *DarwinThrottler) OnProcessStarted(pid int) error {
	// 1. Lower priority via setpriority
	_ = syscall.Setpriority(syscall.PRIO_PROCESS, pid, 15)

	// 2. Apply macOS taskpolicy background clamping (efficiency cores + throttled disk I/O)
	tpCmd := exec.Command("taskpolicy", "-b", "-c", "background", "-d", "throttle", "-p", fmt.Sprintf("%d", pid))
	_ = tpCmd.Run()

	return nil
}

func (d *DarwinThrottler) PauseProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGSTOP)
}

func (d *DarwinThrottler) ResumeProcess(pid int) error {
	return syscall.Kill(pid, syscall.SIGCONT)
}

func (d *DarwinThrottler) IsOnBattery() bool {
	out, err := exec.Command("pmset", "-g", "batt").Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(out), "Battery Power")
}

func (d *DarwinThrottler) StartDutyCycleLoop(ctx context.Context, pid int, profile ThermalProfile) {
	if profile.WorkTime <= 0 || profile.PauseTime <= 0 {
		return
	}

	ticker := time.NewTicker(profile.WorkTime + profile.PauseTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = d.ResumeProcess(pid)
			return
		case <-ticker.C:
			// Pause
			_ = d.PauseProcess(pid)
			select {
			case <-ctx.Done():
				_ = d.ResumeProcess(pid)
				return
			case <-time.After(profile.PauseTime):
				_ = d.ResumeProcess(pid)
			}
		}
	}
}

func (d *DarwinThrottler) Cleanup() {
	// Reset Apple Silicon fan locks if mtplx max was previously active
	if path, err := exec.LookPath("mtplx"); err == nil && path != "" {
		_ = exec.Command(path, "max", "--off").Run()
	}
}
