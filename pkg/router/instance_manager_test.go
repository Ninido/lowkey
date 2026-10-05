package router

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"

	"lowkey/pkg/engine"
)

// Use this test binary as a tiny backend; no models or installed engines needed.
func TestRouterBackendProcess(t *testing.T) {
	port := os.Getenv("LOWKEY_TEST_BACKEND_PORT")
	if port == "" {
		return
	}
	if os.Getenv("LOWKEY_TEST_BACKEND_MODE") == "exit" {
		os.Exit(2)
	}
	time.Sleep(150 * time.Millisecond)
	http.ListenAndServe("127.0.0.1:"+port, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Write([]byte(`{"data":[]}`))
	}))
	os.Exit(1)
}

type testBackend struct{ engine.MtplxEngine }

func (testBackend) ID() string { return "router-test-backend" }
func (testBackend) BuildCommand(ctx context.Context, cfg *engine.LaunchConfig) (*exec.Cmd, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cmd := exec.CommandContext(ctx, exe, "-test.run=^TestRouterBackendProcess$")
	cmd.Env = append(os.Environ(), "LOWKEY_TEST_BACKEND_PORT="+strconv.Itoa(cfg.Port), "LOWKEY_TEST_BACKEND_MODE="+cfg.ModelPath)
	return cmd, nil
}

func TestInstanceLifecycle(t *testing.T) {
	engine.Register(&testBackend{})
	im := NewInstanceManager(&RouterConfig{Models: map[string]*ModelConfig{
		"test": {Engine: "router-test-backend", Path: "serve"},
		"fail": {Engine: "router-test-backend", Path: "exit"},
	}})
	t.Cleanup(im.Shutdown)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	inst, err := im.GetInstance(ctx, "test")
	if err != nil {
		t.Fatal(err)
	}
	// Completing/canceling the initiating request must not kill the backend.
	cancel()
	im.EndSession(inst.ModelName)
	url := "http://127.0.0.1:" + strconv.Itoa(inst.Port) + "/v1/models"
	client := &http.Client{Timeout: time.Second}
	time.Sleep(50 * time.Millisecond)
	resp, err := client.Get(url)
	if err != nil {
		t.Fatalf("backend died with initiating request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("backend not ready: %d", resp.StatusCode)
	}
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	reused, err := im.GetInstance(ctx, "test")
	if err != nil || reused != inst {
		t.Fatalf("backend was not reused: %v", err)
	}
	im.KillIdleInstances(-time.Second)
	im.mu.RLock()
	_, exists := im.instances["test"]
	im.mu.RUnlock()
	if !exists {
		t.Fatal("reaper killed an active session")
	}
	im.EndSession(inst.ModelName)
	if err := inst.Process.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		im.mu.RLock()
		_, exists = im.instances["test"]
		im.mu.RUnlock()
		if !exists {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("dead backend remained cached")
		}
		time.Sleep(10 * time.Millisecond)
	}
	restarted, err := im.GetInstance(ctx, "test")
	if err != nil || restarted == inst {
		t.Fatalf("dead backend was not restarted: %v", err)
	}
	if _, err := im.GetInstance(ctx, "fail"); err == nil || !strings.Contains(err.Error(), "backend exited") {
		t.Fatalf("missing startup failure: %v", err)
	}
}

func TestReadinessTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if err := waitForReady(ctx, srv.URL, make(chan error)); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("accepted a loading backend or missed timeout: %v", err)
	}
}
