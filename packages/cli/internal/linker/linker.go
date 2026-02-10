package linker

import (
	"fmt"
	"strings"

	"github.com/agentx-labs/agentx/internal/integrations"
	"github.com/agentx-labs/agentx/internal/userdata"
)

// typeSection maps a type reference path prefix to the ActiveConfig section name.
func typeSection(typeRef string) (string, error) {
	parts := strings.SplitN(typeRef, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid type reference %q: expected <type>/<name> (e.g., personas/senior-java-dev)", typeRef)
	}

	switch parts[0] {
	case "personas":
		return "personas", nil
	case "context":
		return "context", nil
	case "skills":
		return "skills", nil
	case "workflows":
		return "workflows", nil
	case "prompts":
		return "prompts", nil
	default:
		return "", fmt.Errorf("unknown type prefix %q in reference %q", parts[0], typeRef)
	}
}

// getSection returns a pointer to the slice in ActiveConfig for the given section name.
func getSection(active *ActiveConfig, section string) *[]string {
	switch section {
	case "personas":
		return &active.Personas
	case "context":
		return &active.Context
	case "skills":
		return &active.Skills
	case "workflows":
		return &active.Workflows
	case "prompts":
		return &active.Prompts
	default:
		return nil
	}
}

// contains checks if a string slice contains a value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// remove returns a new slice with the given value removed.
func remove(slice []string, val string) []string {
	result := make([]string, 0, len(slice))
	for _, s := range slice {
		if s != val {
			result = append(result, s)
		}
	}
	return result
}

// AddType adds a type reference to the correct section of project.yaml and runs sync.
func AddType(projectPath, typeRef string) error {
	section, err := typeSection(typeRef)
	if err != nil {
		return err
	}

	config, err := LoadProject(projectPath)
	if err != nil {
		return err
	}

	target := getSection(&config.Active, section)
	if target == nil {
		return fmt.Errorf("internal error: unknown section %q", section)
	}

	if contains(*target, typeRef) {
		return fmt.Errorf("%s is already linked", typeRef)
	}

	*target = append(*target, typeRef)

	if err := SaveProject(projectPath, config); err != nil {
		return err
	}

	return Sync(projectPath)
}

// RemoveType removes a type reference from project.yaml and runs sync.
func RemoveType(projectPath, typeRef string) error {
	section, err := typeSection(typeRef)
	if err != nil {
		return err
	}

	config, err := LoadProject(projectPath)
	if err != nil {
		return err
	}

	target := getSection(&config.Active, section)
	if target == nil {
		return fmt.Errorf("internal error: unknown section %q", section)
	}

	if !contains(*target, typeRef) {
		return fmt.Errorf("%s is not currently linked", typeRef)
	}

	*target = remove(*target, typeRef)

	if err := SaveProject(projectPath, config); err != nil {
		return err
	}

	return Sync(projectPath)
}

// Sync loads project.yaml and regenerates all tool configurations.
func Sync(projectPath string) error {
	config, err := LoadProject(projectPath)
	if err != nil {
		return err
	}

	installedPath, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	tools := make([]integrations.ToolName, 0, len(config.Tools))
	for _, t := range config.Tools {
		tool, ok := integrations.ParseToolName(t)
		if !ok {
			return fmt.Errorf("unknown tool in project.yaml: %s", t)
		}
		tools = append(tools, tool)
	}

	// Convert ProjectConfig to a map for the Node generators.
	configMap := map[string]interface{}{
		"tools": config.Tools,
		"active": map[string]interface{}{
			"personas":  config.Active.Personas,
			"context":   config.Active.Context,
			"skills":    config.Active.Skills,
			"workflows": config.Active.Workflows,
			"prompts":   config.Active.Prompts,
		},
	}

	results, err := integrations.GenerateConfigs(tools, configMap, installedPath, projectPath)
	if err != nil {
		return fmt.Errorf("generating configs: %w", err)
	}

	// Print summary
	for _, r := range results {
		fmt.Printf("  %s: %d created, %d updated, %d symlinked\n",
			r.Tool, len(r.Created), len(r.Updated), len(r.Symlinked))
		for _, w := range r.Warnings {
			fmt.Printf("    warning: %s\n", w)
		}
	}

	return nil
}

// Status calls each tool's status check and returns the results.
func Status(projectPath string) ([]integrations.StatusResult, error) {
	config, err := LoadProject(projectPath)
	if err != nil {
		return nil, err
	}

	tools := make([]integrations.ToolName, 0, len(config.Tools))
	for _, t := range config.Tools {
		tool, ok := integrations.ParseToolName(t)
		if !ok {
			return nil, fmt.Errorf("unknown tool in project.yaml: %s", t)
		}
		tools = append(tools, tool)
	}

	return integrations.GetStatus(tools, projectPath)
}
