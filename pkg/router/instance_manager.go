package router

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"lowkey/pkg/engine"
)

// Instance represents a running inference engine process
type Instance struct {
	ModelName      string
	Port           int
	Process        *exec.Cmd
	Engine         engine.Engine
	LastUsed       time.Time
	Healthy        bool
	ActiveSessions int
}

// InstanceManager manages the lifecycle of inference engine instances
type InstanceManager struct {
	mu sync.RWMutex
	// ponytail: serialize model loads; per-model locks if parallel startup matters.
	spawnMu   sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
	instances map[string]*Instance
	cfg       *RouterConfig
}

// NewInstanceManager creates a new instance manager
func NewInstanceManager(cfg *RouterConfig) *InstanceManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &InstanceManager{
		ctx:       ctx,
		cancel:    cancel,
		instances: make(map[string]*Instance),
		cfg:       cfg,
	}
}

// GetInstance returns the instance for a model, spawning it if necessary
func (im *InstanceManager) GetInstance(ctx context.Context, modelName string) (*Instance, error) {
	// Try primary model
	inst, err := im.tryGetInstance(ctx, modelName)
	if err == nil {
		return inst, nil
	}

	// Try fallbacks
	modelCfg, exists := im.cfg.Models[modelName]
	if !exists || len(modelCfg.Fallback) == 0 {
		return nil, err
	}

	for _, fallbackName := range modelCfg.Fallback {
		fmt.Printf("Using fallback model %s instead of %s\n", fallbackName, modelName)
		inst, err = im.tryGetInstance(ctx, fallbackName)
		if err == nil {
			return inst, nil
		}
	}

	return nil, err
}

func (im *InstanceManager) tryGetInstance(ctx context.Context, modelName string) (*Instance, error) {
	im.spawnMu.Lock()
	defer im.spawnMu.Unlock()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	im.mu.Lock()
	instance, exists := im.instances[modelName]
	if exists && instance.Healthy {
		instance.LastUsed = time.Now()
		instance.ActiveSessions++
		im.mu.Unlock()
		return instance, nil
	}
	im.mu.Unlock()

	// Check if we have enough memory to load this model
	modelCfg, exists := im.cfg.Models[modelName]
	if exists {
		if !im.hasEnoughMemory(modelCfg.Path) {
			// Try to free up memory by killing idle instances (no active sessions)
			im.evictIdleInstances()
			if !im.hasEnoughMemory(modelCfg.Path) {
				return nil, fmt.Errorf("not enough memory to load model %s", modelName)
			}
		}
	}

	// Spawn new instance
	inst, err := im.spawnInstance(ctx, modelName)
	if err != nil {
		return nil, err
	}

	// Mark as having an active session
	im.mu.Lock()
	inst.ActiveSessions++
	im.mu.Unlock()

	return inst, nil
}

// EndSession marks a session as ended for a model
func (im *InstanceManager) EndSession(modelName string) {
	im.mu.Lock()
	defer im.mu.Unlock()

	instance, exists := im.instances[modelName]
	if exists && instance.ActiveSessions > 0 {
		instance.ActiveSessions--
	}
}

// hasEnoughMemory checks if there's enough memory to load a model
func (im *InstanceManager) hasEnoughMemory(modelPath string) bool {
	needed, err := EstimateModelMemory(modelPath)
	if err != nil {
		return true // If we can't estimate, proceed anyway
	}

	available, err := GetAvailableMemoryBytes()
	if err != nil {
		return true // If we can't check, proceed anyway
	}

	// Keep reserve_pct free for the OS
	reserve := int64(im.cfg.MemoryReservePct) * available / 100
	return needed <= available-reserve
}

// evictIdleInstances kills the least recently used idle instances to free memory
// Only evicts instances with no active sessions
func (im *InstanceManager) evictIdleInstances() {
	im.mu.Lock()
	defer im.mu.Unlock()

	// Find idle instances (no active sessions) and sort by last used (oldest first)
	type idleInstance struct {
		name string
		time time.Time
	}
	var idle []idleInstance

	for name, inst := range im.instances {
		if inst.ActiveSessions == 0 {
			idle = append(idle, idleInstance{name: name, time: inst.LastUsed})
		}
	}

	// Kill the oldest idle instance
	if len(idle) > 0 {
		var oldestIndex int
		for i, inst := range idle {
			if inst.time.Before(idle[oldestIndex].time) {
				oldestIndex = i
			}
		}
		oldestName := idle[oldestIndex].name
		inst := im.instances[oldestName]
		if inst.Process != nil && inst.Process.Process != nil {
			_ = inst.Process.Process.Kill()
		}
		delete(im.instances, oldestName)
	}
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
		EngineID:       eng.ID(),
		ModelID:        modelName,
		ModelPath:      modelCfg.Path,
		Port:           port,
		Host:           "127.0.0.1",
		ContextSize:    modelCfg.ContextSize,
		ThermalProfile: modelCfg.ThermalProfile,
	}

	// A shared server must outlive the request that first starts it.
	cmd, err := eng.BuildCommand(im.ctx, launchCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to build command for model %s: %w", modelName, err)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start process for model %s: %w", modelName, err)
	}

	exited := make(chan error, 1)
	go func() { exited <- cmd.Wait() }()
	readyCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	url := fmt.Sprintf("http://127.0.0.1:%d/v1/models", port)
	if err := waitForReady(readyCtx, url, exited); err != nil {
		_ = cmd.Process.Kill()
		return nil, fmt.Errorf("model %s failed to become ready: %w (see backend logs)", modelName, err)
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
	if err := im.ctx.Err(); err != nil {
		im.mu.Unlock()
		_ = cmd.Process.Kill()
		return nil, err
	}
	im.instances[modelName] = instance
	im.mu.Unlock()

	go func() {
		<-exited
		im.mu.Lock()
		defer im.mu.Unlock()
		instance.Healthy = false
		if im.instances[modelName] == instance {
			delete(im.instances, modelName)
		}
	}()
	return instance, nil
}

// Wait for a successful API response, not just an open port during model loading.
func waitForReady(ctx context.Context, url string, exited <-chan error) error {
	client := &http.Client{Timeout: time.Second}
	ticker := time.NewTicker(200 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case err := <-exited:
			return fmt.Errorf("backend exited: %v", err)
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		resp, err := client.Do(req)
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return nil
			}
		}
		select {
		case err := <-exited:
			return fmt.Errorf("backend exited: %v", err)
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
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
		if instance.ActiveSessions == 0 && now.Sub(instance.LastUsed) > timeout {
			if instance.Process != nil && instance.Process.Process != nil {
				_ = instance.Process.Process.Kill()
			}
			delete(im.instances, modelName)
		}
	}
}

// Shutdown kills all instances
func (im *InstanceManager) Shutdown() {
	im.cancel()
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
