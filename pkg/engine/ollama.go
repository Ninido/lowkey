package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

type OllamaEngine struct{}

func init() {
	Register(&OllamaEngine{})
}

func (o *OllamaEngine) ID() string {
	return "ollama"
}

func (o *OllamaEngine) Name() string {
	return "Ollama"
}

func (o *OllamaEngine) Description() string {
	return "One-command local LLM runner and model manager"
}

func (o *OllamaEngine) IsInstalled() (bool, string) {
	if path, err := exec.LookPath("ollama"); err == nil && path != "" {
		return true, path
	}
	return false, ""
}

func (o *OllamaEngine) DefaultPort() int {
	return 11434
}

type ollamaListOutput struct {
	Models []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"models"`
}

func (o *OllamaEngine) DiscoverModels() ([]ModelInfo, error) {
	_, bin := o.IsInstalled()
	if bin == "" {
		return nil, fmt.Errorf("ollama not found")
	}

	out, err := exec.Command(bin, "list").Output()
	if err != nil {
		return nil, err
	}

	var results []ModelInfo
	// Try parsing json or lines
	var data ollamaListOutput
	if jsonErr := json.Unmarshal(out, &data); jsonErr == nil && len(data.Models) > 0 {
		for _, m := range data.Models {
			results = append(results, ModelInfo{
				ID:          m.Name,
				DisplayName: fmt.Sprintf("%s (%s)", m.Name, formatSize(m.Size)),
				Path:        m.Name,
			})
		}
		return results, nil
	}

	// Line-based fallback
	lines := splitLines(string(out))
	for i, line := range lines {
		if i == 0 || line == "" { // skip header "NAME ID SIZE MODIFIED"
			continue
		}
		parts := splitFields(line)
		if len(parts) > 0 {
			results = append(results, ModelInfo{
				ID:          parts[0],
				DisplayName: parts[0],
				Path:        parts[0],
			})
		}
	}

	return results, nil
}

func (o *OllamaEngine) BuildCommand(ctx context.Context, cfg *LaunchConfig) (*exec.Cmd, error) {
	installed, bin := o.IsInstalled()
	if !installed {
		return nil, fmt.Errorf("ollama is not installed")
	}

	// ollama run <model>
	args := []string{"run", cfg.ModelID}
	cmd := exec.CommandContext(ctx, bin, args...)

	if cfg.Port > 0 {
		cmd.Env = append(cmd.Environ(), "OLLAMA_HOST=0.0.0.0:"+strconv.Itoa(cfg.Port))
	}

	return cmd, nil
}

func splitLines(s string) []string {
	var lines []string
	curr := ""
	for _, r := range s {
		if r == '\n' {
			lines = append(lines, curr)
			curr = ""
		} else if r != '\r' {
			curr += string(r)
		}
	}
	if curr != "" {
		lines = append(lines, curr)
	}
	return lines
}

func splitFields(s string) []string {
	var fields []string
	curr := ""
	for _, r := range s {
		if r == ' ' || r == '\t' {
			if curr != "" {
				fields = append(fields, curr)
				curr = ""
			}
		} else {
			curr += string(r)
		}
	}
	if curr != "" {
		fields = append(fields, curr)
	}
	return fields
}
