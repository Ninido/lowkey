package router

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	"lowkey/pkg/engine"
)

// Instance represents a running inference engine process
type Instance struct {
	ModelName    string
	Port         int
	Process      *exec.Cmd
	Engine       engine.Engine
	LastUsed     time.Time
	Healthy      bool
}

// InstanceManager manages the lifecycle of inference engine instances
type InstanceManager struct {
	mu        sync.RWMutex
	instances map[string]*Instance
	cfg       *RouterConfig
}

// NewInstanceManager creates a new instance manager
func NewInstanceManager(cfg *RouterConfig) *InstanceManager {
	return &InstanceManager{
		instances: make(map[string]*Instance),
		cfg:       cfg,
	}
}

// GetInstance returns the instance for a model, spawning it if necessary
func (im *InstanceManager) GetInstance(ctx context.Context, modelName string) (*Instance, error) {
	im.mu.RLock()
	instance, exists := im.instances[modelName]
	im.mu.RUnlock()

	if exists && instance.Healthy {
		instance.LastUsed = time.Now()
		return instance, nil
	}

	// Spawn new instance
	return im.spawnInstance(ctx, modelName)
}

func (im *InstanceManager) spawnInstance(ctx context.Context, modelName string) (*Instance, error) {
	// Find model config
	modelCfg, exists := im.cfg.Models[modelName]
	if !exists {
		return nil, fmt.Errorf("unknown model: %s", modelName)
	}

	// Get engine
	eng, err := engine.Get(modelCfg.Engine)
	if err != nil {
		return nil, fmt.Errorf("failed to get engine for model %s: %w", modelName, err)
	}

	// Find available port
	port, err := findAvailablePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find available port for model %s: %w", modelName, err)
	}

	// Build command
	launchCfg := &engine.LaunchConfig{
		EngineID:     eng.ID(),
		ModelID:      modelName,
		ModelPath:    modelCfg.Path,
		Port:         port,
		Host:         "127.0.0.1",
		ContextSize:  modelCfg.ContextSize,
		ThermalProfile: modelCfg.ThermalProfile,
	}

	cmd, err := eng.BuildCommand(ctx, launchCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for model %s: %w", modelName, err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process for model %s: %w", modelName, err)
	}

	instance := &Instance{
		ModelName: modelName,
		Port:      port,
		Process:   cmd,
		Engine:    eng,
		LastUsed:  time.Now(),
		Healthy:   true,
	}

	im.mu.Lock()
	im.instances[modelName] = instance
	im.mu.Unlock()

	return instance, nil
}

func (im *InstanceManager) killInstance(modelName string) {
	im.mu.Lock()
	instance, exists := im.instances[modelName]
	if exists {
		if instance.Process != nil && instance.Process.Process != nil {
			_ = instance.Process.Process.Kill()
		}
		delete(im.instances, modelName)
	}
	im.mu.Unlock()
}

// KillIdleInstances kills instances that haven't been used recently
func (im *InstanceManager) KillIdleInstances(timeout time.Duration) {
	now := time.Now()
	im.mu.Lock()
	defer im.mu.Unlock()

	for modelName, instance := range im.instances {
		if now.Sub(instance.LastUsed) > timeout {
			if instance.Process != nil && instance.Process.Process != nil {
				_ = instance.Process.Process.Kill()
			}
			delete(im.instances, modelName)
		}
	}
}

// Shutdown kills all instances
func (im *InstanceManager) Shutdown() {
	im.mu.Lock()
	defer im.mu.Unlock()

	for modelName, instance := range im.instances {
		if instance.Process != nil && instance.Process.Process != nil {
			_ = instance.Process.Process.Kill()
		}
		delete(im.instances, modelName)
	}
}

func findAvailablePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}
