package engine

import (
	"context"
	"strings"
	"testing"
)

func TestLlamaCppMTPFlags(t *testing.T) {
	l := &LlamaCppEngine{}
	if ok, _ := l.IsInstalled(); !ok {
		t.Skip("llama-server not installed")
	}
	build := func(cfg LaunchConfig) string {
		cmd, err := l.BuildCommand(context.Background(), &cfg)
		if err != nil {
			t.Fatal(err)
		}
		return strings.Join(cmd.Args, " ")
	}

	if got := build(LaunchConfig{ModelPath: "/m/Qwen-MTP-Q8_0.gguf", SpeculationDepth: 2}); !strings.Contains(got, "--spec-type draft-mtp --spec-draft-n-max 2") {
		t.Errorf("MTP model missing spec flags: %s", got)
	}
	if got := build(LaunchConfig{ModelPath: "/m/plain-Q4.gguf", SpeculationDepth: 3}); strings.Contains(got, "--spec-type") {
		t.Errorf("non-MTP model got spec flags: %s", got)
	}
	if got := build(LaunchConfig{ModelPath: "/m/Qwen-MTP.gguf", SpeculationDepth: 2, ExtraFlags: map[string]string{"spec-type": "draft-mtp"}}); strings.Count(got, "--spec-type") != 1 {
		t.Errorf("spec-type duplicated: %s", got)
	}
}
