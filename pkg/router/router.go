package router

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"
)

// RouterConfig holds the router configuration
type RouterConfig struct {
	Host             string           `json:"host"`
	Port             int              `json:"port"`
	MemoryReservePct int              `json:"memory_reserve_pct"`
	IdleTimeoutMin   int              `json:"idle_timeout_min"`
	Models           map[string]*ModelConfig `json:"models"`
}

// ModelConfig holds configuration for a specific model
type ModelConfig struct {
	Engine         string `json:"engine"`
	Path           string `json:"path"`
	ContextSize    int    `json:"context_size"`
	ThermalProfile string `json:"thermal_profile"`
}

// Router is the main router server
type Router struct {
	cfg          *RouterConfig
	instanceMgr  *InstanceManager
	clients      map[string]*http.Client
	clientsMu    sync.RWMutex
	idleTicker   *time.Ticker
}

// NewRouter creates a new router
func NewRouter(cfg *RouterConfig) (*Router, error) {
	if cfg.Port == 0 {
		cfg.Port = 8000
	}
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1" // localhost-only by default for security
	}
	if cfg.IdleTimeoutMin == 0 {
		cfg.IdleTimeoutMin = 10
	}

	return &Router{
		cfg:         cfg,
		instanceMgr: NewInstanceManager(cfg),
		clients:     make(map[string]*http.Client),
		idleTicker:  time.NewTicker(time.Minute),
	}, nil
}

// Run starts the router server
func (r *Router) Run() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/chat/completions", r.handleChatCompletions)
	mux.HandleFunc("/v1/chat/completions", r.handleChatCompletions)
	mux.HandleFunc("/completions", r.handleCompletions)
	mux.HandleFunc("/v1/completions", r.handleCompletions)
	mux.HandleFunc("/models", r.handleModels)
	mux.HandleFunc("/v1/models", r.handleModels)
	mux.HandleFunc("/health", r.handleHealth)

	// Start idle instance reaper
	go r.reapIdleInstances()

	addr := fmt.Sprintf("%s:%d", r.cfg.Host, r.cfg.Port)
	log.Printf("Router listening on %s", addr)
	if r.cfg.Host != "127.0.0.1" && r.cfg.Host != "localhost" {
		log.Printf("WARNING: Router is accessible from other machines (host=%s). Set host to 127.0.0.1 for localhost-only.", r.cfg.Host)
	}
	log.Printf("Available models: %v", r.modelNames())

	return http.ListenAndServe(addr, mux)
}

func (r *Router) handleChatCompletions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	modelName, err := extractModelName(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to extract model name: %v", err), http.StatusBadRequest)
		return
	}

	instance, err := r.instanceMgr.GetInstance(req.Context(), modelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get instance for model %s: %v", modelName, err), http.StatusServiceUnavailable)
		return
	}
	defer r.instanceMgr.EndSession(modelName)

	r.proxyRequest(w, req, instance, body)
}

func (r *Router) handleCompletions(w http.ResponseWriter, req *http.Request) {
	if req.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	modelName, err := extractModelName(body)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to extract model name: %v", err), http.StatusBadRequest)
		return
	}

	instance, err := r.instanceMgr.GetInstance(req.Context(), modelName)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to get instance for model %s: %v", modelName, err), http.StatusServiceUnavailable)
		return
	}
	defer r.instanceMgr.EndSession(modelName)

	r.proxyRequest(w, req, instance, body)
}

func (r *Router) handleModels(w http.ResponseWriter, req *http.Request) {
	models := make([]map[string]interface{}, 0, len(r.cfg.Models))
	for name := range r.cfg.Models {
		models = append(models, map[string]interface{}{
			"id":      name,
			"object":  "model",
			"created": time.Now().Unix(),
			"owned_by": "lowkey",
		})
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"object": "list",
		"data":   models,
	})
}

func (r *Router) handleHealth(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (r *Router) proxyRequest(w http.ResponseWriter, req *http.Request, instance *Instance, body []byte) {
	url := fmt.Sprintf("http://127.0.0.1:%d%s", instance.Port, req.URL.Path)

	proxyReq, err := http.NewRequest(req.Method, url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to create proxy request: %v", err), http.StatusInternalServerError)
		return
	}

	// Copy headers
	for key, values := range req.Header {
		for _, value := range values {
			proxyReq.Header.Add(key, value)
		}
	}

	// Re-wrap body
	proxyReq.Body = io.NopCloser(bytes.NewReader(body))
	proxyReq.ContentLength = int64(len(body))

	client := r.getClient(instance.ModelName)
	resp, err := client.Do(proxyReq)
	if err != nil {
		http.Error(w, fmt.Sprintf("failed to proxy request: %v", err), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func (r *Router) getClient(modelName string) *http.Client {
	r.clientsMu.RLock()
	client, exists := r.clients[modelName]
	r.clientsMu.RUnlock()

	if exists {
		return client
	}

	r.clientsMu.Lock()
	defer r.clientsMu.Unlock()

	// Double-check
	if client, exists = r.clients[modelName]; exists {
		return client
	}

	client = &http.Client{Timeout: 60 * time.Second}
	r.clients[modelName] = client
	return client
}

func (r *Router) reapIdleInstances() {
	for range r.idleTicker.C {
		timeout := time.Duration(r.cfg.IdleTimeoutMin) * time.Minute
		r.instanceMgr.KillIdleInstances(timeout)
	}
}

func (r *Router) modelNames() []string {
	names := make([]string, 0, len(r.cfg.Models))
	for name := range r.cfg.Models {
		names = append(names, name)
	}
	return names
}

// Shutdown stops the router and kills all instances
func (r *Router) Shutdown() {
	r.idleTicker.Stop()
	r.instanceMgr.Shutdown()
	log.Println("Router shut down")
}

// extractModelName extracts the model name from the request body
func extractModelName(body []byte) (string, error) {
	var req map[string]interface{}
	if err := json.Unmarshal(body, &req); err != nil {
		return "", err
	}

	model, exists := req["model"]
	if !exists {
		return "", fmt.Errorf("model field not found in request")
	}

	modelStr, ok := model.(string)
	if !ok {
		return "", fmt.Errorf("model field is not a string")
	}

	return modelStr, nil
}

// LoadRouterConfig loads router configuration from a JSON file
func LoadRouterConfig(path string) (*RouterConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg RouterConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
