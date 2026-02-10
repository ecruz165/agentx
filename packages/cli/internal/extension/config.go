package extension

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// ProjectConfigFile is the brand-agnostic filename for the project-level
// configuration (extensions, resolution order). It lives at the repo root.
const ProjectConfigFile = "project.yaml"

// ExtensionConfig represents the project.yaml configuration file.
type ExtensionConfig struct {
	Extensions []Extension      `yaml:"extensions"`
	Resolution ResolutionConfig `yaml:"resolution"`
}

// Extension represents a single extension entry in project.yaml.
type Extension struct {
	Name   string `yaml:"name"`
	Path   string `yaml:"path"`
	Source string `yaml:"source"`
	Branch string `yaml:"branch"`
}

// ResolutionConfig holds the type resolution order.
type ResolutionConfig struct {
	Order []string `yaml:"order"`
}

// LoadConfig reads and parses a project.yaml file.
func LoadConfig(path string) (*ExtensionConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	var cfg ExtensionConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config %s: %w", path, err)
	}

	return &cfg, nil
}

// SaveConfig writes the configuration back to a project.yaml file.
func SaveConfig(path string, cfg *ExtensionConfig) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing config %s: %w", path, err)
	}

	return nil
}

// FindExtension returns the extension with the given name, or nil if not found.
func (c *ExtensionConfig) FindExtension(name string) *Extension {
	for i := range c.Extensions {
		if c.Extensions[i].Name == name {
			return &c.Extensions[i]
		}
	}
	return nil
}

// AddExtension appends a new extension to the config if it does not already exist.
// Returns an error if an extension with the same name is already present.
func (c *ExtensionConfig) AddExtension(ext Extension) error {
	if c.FindExtension(ext.Name) != nil {
		return fmt.Errorf("extension %q already exists", ext.Name)
	}
	c.Extensions = append(c.Extensions, ext)
	return nil
}

// RemoveExtension removes an extension by name.
// Returns an error if the extension is not found.
func (c *ExtensionConfig) RemoveExtension(name string) error {
	for i, ext := range c.Extensions {
		if ext.Name == name {
			c.Extensions = append(c.Extensions[:i], c.Extensions[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("extension %q not found in configuration", name)
}
