package profile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"lowkey/pkg/engine"
)

// Profile represents a saved setup configuration
type Profile struct {
	Name        string              `json:"name"`
	Description string              `json:"description,omitempty"`
	CreatedAt   time.Time           `json:"created_at"`
	Config      engine.LaunchConfig `json:"config"`
}

// GetProfilesDir returns the directory where profiles are stored
func GetProfilesDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".lowkey", "profiles")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	return dir, nil
}

// ListProfiles returns all saved profiles found on disk
func ListProfiles() ([]Profile, error) {
	dir, err := GetProfilesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var profiles []Profile
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(strings.ToLower(e.Name()), ".json") {
			filePath := filepath.Join(dir, e.Name())
			data, err := os.ReadFile(filePath)
			if err != nil {
				continue
			}
			var p Profile
			if err := json.Unmarshal(data, &p); err == nil {
				if p.Name == "" {
					p.Name = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
				}
				profiles = append(profiles, p)
			}
		}
	}

	return profiles, nil
}

// SaveProfile saves a profile to JSON
func SaveProfile(p Profile) (string, error) {
	dir, err := GetProfilesDir()
	if err != nil {
		return "", err
	}

	sanitizedName := sanitizeFilename(p.Name)
	if sanitizedName == "" {
		sanitizedName = fmt.Sprintf("setup-%d", time.Now().Unix())
	}
	p.CreatedAt = time.Now()

	filePath := filepath.Join(dir, sanitizedName+".json")
	data, err := json.MarshalIndent(p, "", "  ")
	if err != nil {
		return "", err
	}

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", err
	}

	return filePath, nil
}

// LoadProfile loads a profile by name or file path
func LoadProfile(nameOrPath string) (*Profile, error) {
	var filePath string
	if strings.HasSuffix(nameOrPath, ".json") && fileExists(nameOrPath) {
		filePath = nameOrPath
	} else {
		dir, err := GetProfilesDir()
		if err != nil {
			return nil, err
		}
		filePath = filepath.Join(dir, sanitizeFilename(nameOrPath)+".json")
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("could not read profile at %s: %w", filePath, err)
	}

	var p Profile
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("invalid profile json: %w", err)
	}

	return &p, nil
}

func sanitizeFilename(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	replacer := strings.NewReplacer(
		" ", "-",
		"/", "-",
		"\\", "-",
		":", "-",
		"*", "",
		"?", "",
		"\"", "",
		"<", "",
		">", "",
		"|", "",
	)
	return replacer.Replace(s)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}
