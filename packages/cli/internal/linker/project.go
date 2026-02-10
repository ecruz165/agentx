package linker

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

const (
	agentxDir      = ".agentx"
	projectFile    = "project.yaml"
	overridesDir   = "overrides"
)

// ProjectConfig represents the .agentx/project.yaml structure.
type ProjectConfig struct {
	Tools  []string     `yaml:"tools"`
	Active ActiveConfig `yaml:"active"`
}

// ActiveConfig lists the active type references for a project.
type ActiveConfig struct {
	Personas  []string `yaml:"personas,omitempty"`
	Context   []string `yaml:"context,omitempty"`
	Skills    []string `yaml:"skills,omitempty"`
	Workflows []string `yaml:"workflows,omitempty"`
	Prompts   []string `yaml:"prompts,omitempty"`
}

// ProjectConfigPath returns the full path to .agentx/project.yaml for a project.
func ProjectConfigPath(projectPath string) string {
	return filepath.Join(projectPath, agentxDir, projectFile)
}

// LoadProject reads and parses .agentx/project.yaml from the given project directory.
func LoadProject(projectPath string) (*ProjectConfig, error) {
	path := ProjectConfigPath(projectPath)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading project config: %w", err)
	}

	var config ProjectConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing project config: %w", err)
	}

	return &config, nil
}

// SaveProject writes the project config to .agentx/project.yaml.
func SaveProject(projectPath string, config *ProjectConfig) error {
	path := ProjectConfigPath(projectPath)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("marshaling project config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing project config: %w", err)
	}

	return nil
}

// InitProject creates the .agentx/ directory with project.yaml and overrides/.
func InitProject(projectPath string, tools []string) error {
	dir := filepath.Join(projectPath, agentxDir)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating .agentx directory: %w", err)
	}

	overrides := filepath.Join(dir, overridesDir)
	if err := os.MkdirAll(overrides, 0755); err != nil {
		return fmt.Errorf("creating overrides directory: %w", err)
	}

	config := &ProjectConfig{
		Tools:  tools,
		Active: ActiveConfig{},
	}

	return SaveProject(projectPath, config)
}
