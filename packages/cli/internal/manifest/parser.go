package manifest

import (
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Parse reads a manifest file and returns only the base fields.
// Useful for quick type detection without full parsing.
func Parse(path string) (*BaseManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}

	var base BaseManifest
	if err := yaml.Unmarshal(data, &base); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}

	return &base, nil
}

// ParseFile reads a manifest file, detects its type, and returns the
// fully typed manifest struct. The returned interface{} will be one of:
// *ContextManifest, *PersonaManifest, *SkillManifest, *WorkflowManifest,
// *PromptManifest, or *TemplateManifest.
func ParseFile(path string) (interface{}, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}

	typeName, err := detectType(data)
	if err != nil {
		return nil, fmt.Errorf("detecting manifest type in %s: %w", path, err)
	}

	switch typeName {
	case TypeContext:
		return parseTyped[ContextManifest](data, path)
	case TypePersona:
		return parseTyped[PersonaManifest](data, path)
	case TypeSkill:
		return parseTyped[SkillManifest](data, path)
	case TypeWorkflow:
		return parseTyped[WorkflowManifest](data, path)
	case TypePrompt:
		return parseTyped[PromptManifest](data, path)
	case TypeTemplate:
		return parseTyped[TemplateManifest](data, path)
	default:
		return nil, fmt.Errorf("unknown manifest type %q in %s", typeName, path)
	}
}

// ParseContext reads a manifest file and parses it as a ContextManifest.
func ParseContext(path string) (*ContextManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[ContextManifest](data, path)
}

// ParsePersona reads a manifest file and parses it as a PersonaManifest.
func ParsePersona(path string) (*PersonaManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[PersonaManifest](data, path)
}

// ParseSkill reads a manifest file and parses it as a SkillManifest.
func ParseSkill(path string) (*SkillManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[SkillManifest](data, path)
}

// ParseWorkflow reads a manifest file and parses it as a WorkflowManifest.
func ParseWorkflow(path string) (*WorkflowManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[WorkflowManifest](data, path)
}

// ParsePrompt reads a manifest file and parses it as a PromptManifest.
func ParsePrompt(path string) (*PromptManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[PromptManifest](data, path)
}

// ParseTemplate reads a manifest file and parses it as a TemplateManifest.
func ParseTemplate(path string) (*TemplateManifest, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return parseTyped[TemplateManifest](data, path)
}

// parseTyped unmarshals YAML data into a typed manifest struct.
func parseTyped[T any](data []byte, path string) (*T, error) {
	var m T
	if err := yaml.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing manifest %s: %w", path, err)
	}
	return &m, nil
}

// detectType unmarshals YAML data into a generic map and extracts the type field.
func detectType(data []byte) (string, error) {
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return "", fmt.Errorf("unmarshaling YAML: %w", err)
	}

	typeVal, ok := raw["type"]
	if !ok {
		return "", fmt.Errorf("manifest missing required 'type' field")
	}

	typeName, ok := typeVal.(string)
	if !ok {
		return "", fmt.Errorf("manifest 'type' field is not a string")
	}

	return typeName, nil
}

// readFile reads the contents of a file at the given path.
func readFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file %s: %w", path, err)
	}
	return data, nil
}
