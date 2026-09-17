//go:build windows

package osutil

import (
	"context"
	"fmt"
	"os/exec"
	"syscall"
	"time"
	"unsafe"
)

var (
	modkernel32        = syscall.NewLazyDLL("kernel32.dll")
	modntdll           = syscall.NewLazyDLL("ntdll.dll")
	procOpenProcess    = modkernel32.NewProc("OpenProcess")
	procCloseHandle    = modkernel32.NewProc("CloseHandle")
	procSetPriority    = modkernel32.NewProc("SetPriorityClass")
	procGetPowerStatus = modkernel32.NewProc("GetSystemPowerStatus")

	procNtSuspendProcess = modntdll.NewProc("NtSuspendProcess")
	procNtResumeProcess  = modntdll.NewProc("NtResumeProcess")
)

const (
	PROCESS_SUSPEND_RESUME  = 0x0800
	PROCESS_SET_INFORMATION = 0x0200
	IDLE_PRIORITY_CLASS     = 0x00000040
	BELOW_NORMAL_PRIORITY   = 0x00004000
)

type SYSTEM_POWER_STATUS struct {
	ACLineStatus        byte
	BatteryFlag         byte
	BatteryLifePercent  byte
	SystemStatusFlag    byte
	BatteryLifeTime     uint32
	BatteryFullLifeTime uint32
}

type WindowsThrottler struct{}

func NewOSThrottler() OSThrottler {
	return &WindowsThrottler{}
}

func (w *WindowsThrottler) ConfigureCommand(cmd *exec.Cmd) {
	// Cmd can be configured with specific creation flags if needed (e.g. CREATE_SUSPENDED or BELOW_NORMAL_PRIORITY_CLASS)
}

func (w *WindowsThrottler) OnProcessStarted(pid int) error {
	// Set Below Normal or Idle Priority Class
	handle, _, _ := procOpenProcess.Call(
		uintptr(PROCESS_SET_INFORMATION),
		uintptr(0),
		uintptr(pid),
	)
	if handle != 0 {
		defer procCloseHandle.Call(handle)
		_, _, _ = procSetPriority.Call(handle, uintptr(BELOW_NORMAL_PRIORITY))
	}
	return nil
}

func (w *WindowsThrottler) PauseProcess(pid int) error {
	handle, _, err := procOpenProcess.Call(
		uintptr(PROCESS_SUSPEND_RESUME),
		uintptr(0),
		uintptr(pid),
	)
	if handle == 0 {
		return fmt.Errorf("open process failed: %v", err)
	}
	defer procCloseHandle.Call(handle)

	r, _, err := procNtSuspendProcess.Call(handle)
	if r != 0 {
		return fmt.Errorf("NtSuspendProcess failed: %v", err)
	}
	return nil
}

func (w *WindowsThrottler) ResumeProcess(pid int) error {
	handle, _, err := procOpenProcess.Call(
		uintptr(PROCESS_SUSPEND_RESUME),
		uintptr(0),
		uintptr(pid),
	)
	if handle == 0 {
		return fmt.Errorf("open process failed: %v", err)
	}
	defer procCloseHandle.Call(handle)

	r, _, err := procNtResumeProcess.Call(handle)
	if r != 0 {
		return fmt.Errorf("NtResumeProcess failed: %v", err)
	}
	return nil
}

func (w *WindowsThrottler) IsOnBattery() bool {
	var sps SYSTEM_POWER_STATUS
	ret, _, _ := procGetPowerStatus.Call(uintptr(unsafe.Pointer(&sps)))
	if ret != 0 {
		// ACLineStatus: 0 = Offline (on battery), 1 = Online, 255 = Unknown
		return sps.ACLineStatus == 0
	}
	return false
}

func (w *WindowsThrottler) StartDutyCycleLoop(ctx context.Context, pid int, profile ThermalProfile) {
	if profile.WorkTime <= 0 || profile.PauseTime <= 0 {
		return
	}

	ticker := time.NewTicker(profile.WorkTime + profile.PauseTime)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			_ = w.ResumeProcess(pid)
			return
		case <-ticker.C:
			_ = w.PauseProcess(pid)
			select {
			case <-ctx.Done():
				_ = w.ResumeProcess(pid)
				return
			case <-time.After(profile.PauseTime):
				_ = w.ResumeProcess(pid)
			}
		}
	}
}

func (w *WindowsThrottler) Cleanup() {
	// No fan hooks on standard Windows
}
