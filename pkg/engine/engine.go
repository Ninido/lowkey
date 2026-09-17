package engine

import (
	"context"
	"fmt"
	"os/exec"
)

// ModelInfo describes a discovered model
type ModelInfo struct {
	ID          string // e.g. path or identifier
	DisplayName string // user-friendly name
	Path        string // filesystem path or HF repo
	Size        string // optional size string
}

// LaunchConfig contains user-selected options to launch the engine
type LaunchConfig struct {
	EngineID         string            `json:"engine_id"`
	ModelID          string            `json:"model_id"`
	ModelPath        string            `json:"model_path"`
	Port             int               `json:"port"`
	Host             string            `json:"host"`
	ContextSize      int               `json:"context_size,omitempty"`
	ThinkingEffort   string            `json:"thinking_effort,omitempty"`
	SpeculationDepth int               `json:"speculation_depth,omitempty"`
	Preset           string            `json:"preset,omitempty"`
	ThermalProfile   string            `json:"thermal_profile"`
	ExtraFlags       map[string]string `json:"extra_flags,omitempty"`
}

// Engine defines the interface that all local inference engines must implement
type Engine interface {
	ID() string
	Name() string
	Description() string
	IsInstalled() (bool, string) // installed, path-to-binary
	DefaultPort() int
	DiscoverModels() ([]ModelInfo, error)
	BuildCommand(ctx context.Context, config *LaunchConfig) (*exec.Cmd, error)
}

// Registry manages all registered engines
var registry = make(map[string]Engine)

// Register adds an engine to the global registry
func Register(engine Engine) {
	registry[engine.ID()] = engine
}

// Get retrieves an engine by its ID
func Get(id string) (Engine, error) {
	eng, exists := registry[id]
	if !exists {
		return nil, fmt.Errorf("unknown inference engine: %s", id)
	}
	return eng, nil
}

// GetAll returns all registered engines
func GetAll() []Engine {
	var list []Engine
	for _, e := range registry {
		list = append(list, e)
	}
	return list
}

// DetectAvailable returns only the engines currently installed on the host machine
func DetectAvailable() []Engine {
	var available []Engine
	for _, e := range registry {
		if installed, _ := e.IsInstalled(); installed {
			available = append(available, e)
		}
	}
	return available
}
