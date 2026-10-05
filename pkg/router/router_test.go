package router

import (
	"strings"
	"testing"
)

func TestStartupSummary(t *testing.T) {
	r := &Router{cfg: &RouterConfig{
		IdleTimeoutMin: 20,
		Models: map[string]*ModelConfig{
			"z-model": {},
			"a-model": {},
		},
	}}
	got := r.startupSummary("127.0.0.1:8088")
	for _, want := range []string{
		"Status       Listening", "API          http://127.0.0.1:8088/v1",
		"Idle timeout 20 minutes", "Models (2)\n    • a-model\n    • z-model\n",
		"Press Ctrl+C to stop.",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing %q in summary:\n%s", want, got)
		}
	}
}
