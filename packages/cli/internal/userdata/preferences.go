package userdata

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Preferences represents user-wide defaults stored in preferences.yaml.
type Preferences struct {
	OutputFormat   string `yaml:"output_format,omitempty"`
	Color          bool   `yaml:"color,omitempty"`
	Verbose        bool   `yaml:"verbose,omitempty"`
	DefaultPersona string `yaml:"default_persona,omitempty"`
	DefaultBranch  string `yaml:"default_branch,omitempty"`
	Editor         string `yaml:"editor,omitempty"`

	// Extras holds arbitrary user-defined fields.
	Extras map[string]interface{} `yaml:",inline"`
}

// LoadPreferences reads and parses preferences.yaml.
func LoadPreferences() (*Preferences, error) {
	path, err := GetPreferencesPath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading preferences: %w", err)
	}

	var p Preferences
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing preferences: %w", err)
	}
	return &p, nil
}
