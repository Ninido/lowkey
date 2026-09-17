package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

type LlamaCppEngine struct{}

func init() {
	Register(&LlamaCppEngine{})
}

func (l *LlamaCppEngine) ID() string {
	return "llamacpp"
}

func (l *LlamaCppEngine) Name() string {
	return "llama.cpp (llama-server)"
}

func (l *LlamaCppEngine) Description() string {
	return "Fast and lightweight GGUF inference engine across CPU and GPU"
}

func (l *LlamaCppEngine) IsInstalled() (bool, string) {
	for _, binName := range []string{"llama-server", "llama.cpp-server"} {
		if path, err := exec.LookPath(binName); err == nil && path != "" {
			return true, path
		}
	}
	home, _ := os.UserHomeDir()
	checkPaths := []string{
		"/opt/homebrew/bin/llama-server",
		"/usr/local/bin/llama-server",
		filepath.Join(home, "llama.cpp", "build", "bin", "llama-server"),
	}
	for _, p := range checkPaths {
		if _, err := os.Stat(p); err == nil {
			return true, p
		}
	}
	return false, ""
}

func (l *LlamaCppEngine) DefaultPort() int {
	return 8080
}

func (l *LlamaCppEngine) DiscoverModels() ([]ModelInfo, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	searchDirs := []string{
		filepath.Join(home, ".cache", "lm-studio", "models"),
		filepath.Join(home, ".cache", "huggingface", "hub"),
		filepath.Join(home, "models"),
		filepath.Join(home, ".mtplx", "models"),
	}

	var results []ModelInfo
	seen := make(map[string]bool)

	for _, dir := range searchDirs {
		_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return nil
			}
			if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".gguf") {
				if seen[path] {
					return nil
				}
				seen[path] = true
				results = append(results, ModelInfo{
					ID:          path,
					DisplayName: fmt.Sprintf("[%s] %s (%s)", filepath.Base(filepath.Dir(path)), info.Name(), formatSize(info.Size())),
					Path:        path,
				})
			}
			return nil
		})
	}

	return results, nil
}

func (l *LlamaCppEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := l.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("llama-server is not installed")
	}

	args := []string{"-m", cfg.ModelPath}

	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}
	if cfg.ContextSize > 0 {
		args = append(args, "-c", strconv.Itoa(cfg.ContextSize))
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

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
