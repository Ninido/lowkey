package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type MtplxEngine struct{}

func init() {
	Register(&MtplxEngine{})
}

func (m *MtplxEngine) ID() string {
	return "mtplx"
}

func (m *MtplxEngine) Name() string {
	return "MTPLX (Apple Silicon MTP Server)"
}

func (m *MtplxEngine) Description() string {
	return "Multi-Token Prediction high throughput server for Apple Silicon"
}

func (m *MtplxEngine) IsInstalled() (bool, string) {
	path, err := exec.LookPath("mtplx")
	if err == nil && path != "" {
		return true, path
	}
	home, _ := os.UserHomeDir()
	fallback := filepath.Join(home, ".cargo", "bin", "mtplx")
	if _, err := os.Stat(fallback); err == nil {
		return true, fallback
	}
	return false, ""
}

func (m *MtplxEngine) DefaultPort() int {
	return 8000
}

func (m *MtplxEngine) DiscoverModels() ([]ModelInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	searchDirs := []string{
		filepath.Join(home, ".mtplx", "models"),
		filepath.Join(home, ".cache", "lm-studio", "models"),
		filepath.Join(home, ".cache", "huggingface", "hub"),
	}

	var results []ModelInfo
	seen := make(map[string]bool)

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			fullPath := filepath.Join(dir, e.Name())
			if seen[fullPath] {
				continue
			}
			seen[fullPath] = true

			// If it's a directory or gguf/safetensors
			if e.IsDir() {
				results = append(results, ModelInfo{
					ID:          fullPath,
					DisplayName: fmt.Sprintf("[%s] %s", filepath.Base(dir), e.Name()),
					Path:        fullPath,
				})
			}
		}
	}

	return results, nil
}

func (m *MtplxEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := m.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("mtplx is not installed")
	}

	args := []string{"serve", "--model", cfg.ModelPath, "--no-auth"}

	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}

	// Preset mapping for mtplx:
	// Maps wizard presets to exact mtplx serve flags
	switch cfg.Preset {
	case "throughput-4":
		args = append(args,
			"--batching-preset", "throughput",
			"--max-active-requests", "4",
			"--decode-batch-max", "4",
			"--prefill-chunk-tokens", "2048",
			"--mtp-batch-numerics", "throughput",
			"--profile", "turbo",
			"--fan-mode", "default",
			"--ssd-session-cache", "off",
		)
	case "long-context":
		args = append(args,
			"--batching-preset", "agent",
			"--max-active-requests", "4",
			"--decode-batch-max", "4",
			"--prefill-chunk-tokens", "2048",
			"--profile", "turbo",
			"--fan-mode", "default",
			"--ssd-session-cache", "off",
		)
	case "throughput-8":
		args = append(args,
			"--batching-preset", "throughput",
			"--max-active-requests", "8",
			"--decode-batch-max", "8",
			"--prefill-chunk-tokens", "2048",
			"--mtp-batch-numerics", "throughput",
			"--profile", "turbo",
			"--fan-mode", "default",
			"--ssd-session-cache", "off",
		)
	default:
		// Default profile and fan safety
		args = append(args, "--profile", "turbo", "--fan-mode", "default")
	}

	if cfg.ThinkingEffort != "" && cfg.ThinkingEffort != "off" {
		args = append(args, "--reasoning-effort", cfg.ThinkingEffort)
	} else if cfg.ThinkingEffort == "off" {
		args = append(args, "--reasoning", "off")
	}

	// MTPLX CLI flag is --depth, not --speculation-depth
	if cfg.SpeculationDepth > 0 {
		args = append(args, "--depth", strconv.Itoa(cfg.SpeculationDepth))
	}

	for k, v := range cfg.ExtraFlags {
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k, v)
		}
	}

	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(),
		"MTPLX_STREAM_STALL_DEADLINE_S=180",
		"MTPLX_FORCE_SPECULATIVE=1",
	)

	return cmd, nil
}
