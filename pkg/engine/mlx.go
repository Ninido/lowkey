package engine

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

type MLXEngine struct{}

func init() {
	Register(&MLXEngine{})
}

func (m *MLXEngine) ID() string {
	return "mlx"
}

func (m *MLXEngine) Name() string {
	return "MLX-LM (Apple mlx-community)"
}

func (m *MLXEngine) Description() string {
	return "Official Apple MLX framework server for Apple Silicon"
}

func (m *MLXEngine) IsInstalled() (bool, string) {
	for _, bin := range []string{"mlx_lm.server", "mlx-lm-server"} {
		if path, err := exec.LookPath(bin); err == nil && path != "" {
			return true, path
		}
	}
	// Check python module availability
	if python, err := exec.LookPath("python3"); err == nil {
		out, err := exec.Command(python, "-c", "import mlx_lm").CombinedOutput()
		if err == nil && len(out) == 0 {
			return true, python + " -m mlx_lm.server"
		}
	}
	return false, ""
}

func (m *MLXEngine) DefaultPort() int {
	return 8080
}

func (m *MLXEngine) DiscoverModels() ([]ModelInfo, error) {
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

func (m *MLXEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := m.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("mlx-lm server is not installed")
	}

	var args []string
	var execBin string

	if bin == "python3 -m mlx_lm.server" || len(bin) > 7 && bin[:7] == "python3" {
		execBin = "python3"
		args = []string{"-m", "mlx_lm.server", "--model", cfg.ModelPath}
	} else {
		execBin = bin
		args = []string{"--model", cfg.ModelPath}
	}

	if cfg.Port > 0 {
		args = append(args, "--port", strconv.Itoa(cfg.Port))
	}
	if cfg.Host != "" {
		args = append(args, "--host", cfg.Host)
	}

	return exec.CommandContext(ctx, execBin, args...), nil
}
