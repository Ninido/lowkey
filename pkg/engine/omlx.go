package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type OmlxEngine struct{}

func init() {
	Register(&OmlxEngine{})
}

func (o *OmlxEngine) ID() string {
	return "omlx"
}

func (o *OmlxEngine) Name() string {
	return "oMLX (Production MLX Server)"
}

func (o *OmlxEngine) Description() string {
	return "Production-ready Apple Silicon multi-model LLM server"
}

func (o *OmlxEngine) IsInstalled() (bool, string) {
	if path, err := exec.LookPath("omlx"); err == nil && path != "" {
		return true, path
	}
	checkPaths := []string{
		"/opt/homebrew/bin/omlx",
		"/usr/local/bin/omlx",
	}
	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}
	return false, ""
}

func (o *OmlxEngine) DefaultPort() int {
	return 8000
}

func (o *OmlxEngine) DiscoverModels() ([]ModelInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	searchDirs := []string{
		filepath.Join(home, ".cache", "huggingface", "hub"),
		filepath.Join(home, ".cache", "lm-studio", "models"),
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

func (o *OmlxEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := o.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("omlx is not installed")
	}

	args := []string{"serve", cfg.ModelPath}

	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}

	for k, v := range cfg.ExtraFlags {
		if v == "" {
			args = append(args, "--"+k)
		} else {
			args = append(args, "--"+k, v)
		}
	}

	return exec.CommandContext(ctx, bin, args...), nil
}
