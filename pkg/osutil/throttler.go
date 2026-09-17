package osutil

import (
	"context"
	"os/exec"
	"time"
)

// ThermalProfile defines duty cycle timings and background priority
type ThermalProfile struct {
	Name        string
	WorkTime    time.Duration
	PauseTime   time.Duration
	Description string
}

var (
	ProfileQuiet = ThermalProfile{
		Name:        "quiet",
		WorkTime:    1200 * time.Millisecond,
		PauseTime:   300 * time.Millisecond,
		Description: "Quiet (75-80% duty cycle, fans stay low)",
	}
	ProfileBalanced = ThermalProfile{
		Name:        "balanced",
		WorkTime:    1800 * time.Millisecond,
		PauseTime:   200 * time.Millisecond,
		Description: "Balanced (85-90% duty cycle, moderate heat)",
	}
	ProfileEco = ThermalProfile{
		Name:        "eco",
		WorkTime:    800 * time.Millisecond,
		PauseTime:   400 * time.Millisecond,
		Description: "Eco / Battery (65% duty cycle, maximum battery life)",
	}
	ProfilePriorityOnly = ThermalProfile{
		Name:        "priority-only",
		WorkTime:    0,
		PauseTime:   0,
		Description: "Priority-only (100% duty, no pauses, background QoS only)",
	}
)

// GetAllThermalProfiles returns all built-in thermal profiles
func GetAllThermalProfiles() []ThermalProfile {
	return []ThermalProfile{ProfileQuiet, ProfileBalanced, ProfileEco, ProfilePriorityOnly}
}

// OSThrottler provides cross-platform process throttling, power detection, and priority control.
type OSThrottler interface {
	// ConfigureCommand configures the exec.Cmd before execution (e.g. setting priority/QoS)
	ConfigureCommand(cmd *exec.Cmd)
	// OnProcessStarted is called right after the process starts
	OnProcessStarted(pid int) error
	// PauseProcess suspends execution of the target process
	PauseProcess(pid int) error
	// ResumeProcess resumes execution of the target process
	ResumeProcess(pid int) error
	// IsOnBattery detects if the system is running on battery power
	IsOnBattery() bool
	// StartDutyCycleLoop runs the duty cycle throttle loop until context is canceled
	StartDutyCycleLoop(ctx context.Context, pid int, profile ThermalProfile)
	// Cleanup restores any system changes (e.g. fan speeds)
	Cleanup()
}
