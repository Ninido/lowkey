package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type LMStudioEngine struct{}

func init() {
	Register(&LMStudioEngine{})
}

func (l *LMStudioEngine) ID() string {
	return "lmstudio"
}

func (l *LMStudioEngine) Name() string {
	return "LM Studio (lms CLI)"
}

func (l *LMStudioEngine) Description() string {
	return "LM Studio local developer CLI and headless server"
}

func (l *LMStudioEngine) IsInstalled() (bool, string) {
	if path, err := exec.LookPath("lms"); err == nil && path != "" {
		return true, path
	}
	home, _ := os.UserHomeDir()
	checkPaths := []string{
		filepath.Join(home, ".lmstudio", "bin", "lms"),
		filepath.Join(home, ".cache", "lm-studio", "bin", "lms"),
	}
	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}
	return false, ""
}

func (l *LMStudioEngine) DefaultPort() int {
	return 1234
}

type lmsModelEntry struct {
	ModelKey string `json:"modelKey"`
	Path     string `json:"path"`
	Size     string `json:"sizeBytes"`
	Arch     string `json:"architecture"`
	Params   string `json:"paramsString"`
}

func (l *LMStudioEngine) DiscoverModels() ([]ModelInfo, error) {
	_, bin := l.IsInstalled()
	if bin != "" {
		out, err := exec.Command(bin, "ls", "--json").Output()
		if err == nil {
			var entries []lmsModelEntry
			if jsonErr := json.Unmarshal(out, &entries); jsonErr == nil && len(entries) > 0 {
				var list []ModelInfo
				for _, m := range entries {
					name := m.ModelKey
					if m.Params != "" {
						name = fmt.Sprintf("%s (%s)", m.ModelKey, m.Params)
					}
					list = append(list, ModelInfo{
						ID:          m.ModelKey,
						DisplayName: name,
						Path:        m.Path,
					})
				}
				return list, nil
			}
		}
	}

	// Fallback to searching ~/.cache/lm-studio/models
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(home, ".cache", "lm-studio", "models")
	var results []ModelInfo
	_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if !info.IsDir() && filepath.Ext(path) == ".gguf" {
			rel, _ := filepath.Rel(dir, path)
			results = append(results, ModelInfo{
				ID:          path,
				DisplayName: rel,
				Path:        path,
			})
		}
		return nil
	})
	return results, nil
}

func (l *LMStudioEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := l.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("lms is not installed")
	}

	// lms load <model> --cors --port <port>
	// or lms server start
	args := []string{"load", cfg.ModelID, "--cors"}

	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.ContextSize > 0 {
		args = append(args, "--context-length", strconv.Itoa(cfg.ContextSize))
	}

	return exec.CommandContext(ctx, bin, args...), nil
}
