package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type VLLMEngine struct{}

func init() {
	Register(&VLLMEngine{})
}

func (v *VLLMEngine) ID() string {
	return "vllm"
}

func (v *VLLMEngine) Name() string {
	return "vLLM"
}

func (v *VLLMEngine) Description() string {
	return "High-throughput and memory-efficient LLM serving engine (CUDA / ROCm / CPU)"
}

func (v *VLLMEngine) IsInstalled() (bool, string) {
	if path, err := exec.LookPath("vllm"); err == nil && path != "" {
		return true, path
	}
	return false, ""
}

func (v *VLLMEngine) DefaultPort() int {
	return 8000
}

func (v *VLLMEngine) DiscoverModels() ([]ModelInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	searchDirs := []string{
		filepath.Join(home, ".cache", "huggingface", "hub"),
	}
	var results []ModelInfo
	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() {
				full := filepath.Join(dir, e.Name())
				results = append(results, ModelInfo{
					ID:          full,
					DisplayName: e.Name(),
					Path:        full,
				})
			}
		}
	}
	return results, nil
}

func (v *VLLMEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := v.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("vllm is not installed")
	}

	args := []string{"serve", cfg.ModelPath}
	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}

	for k, val := range cfg.ExtraFlags {
		if val == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k, val)
		}
	}

	return exec.CommandContext(ctx, bin, args...), nil
}
