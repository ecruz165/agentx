package manifest

import (
	"fmt"
	"path/filepath"
	"testing"
)

const testdataDir = "testdata"

func testPath(name string) string {
	return filepath.Join(testdataDir, name)
}

func TestParse_BaseFields(t *testing.T) {
	tests := []struct {
		file    string
		name    string
		typ     string
		version string
	}{
		{"valid-context.yaml", "spring-boot-error-handling", TypeContext, "1.0.0"},
		{"valid-persona.yaml", "senior-java-dev", TypePersona, "1.2.0"},
		{"valid-skill.yaml", "commit-analyzer", TypeSkill, "2.1.0"},
		{"valid-workflow.yaml", "deploy-verify", TypeWorkflow, "1.0.0"},
		{"valid-prompt.yaml", "java-pr-review", TypePrompt, "1.0.0"},
		{"valid-template.yaml", "error-spike", TypeTemplate, "1.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			m, err := Parse(testPath(tt.file))
			if err != nil {
				t.Fatalf("Parse(%s) error: %v", tt.file, err)
			}
			if m.Name != tt.name {
				t.Errorf("Name = %q, want %q", m.Name, tt.name)
			}
			if m.Type != tt.typ {
				t.Errorf("Type = %q, want %q", m.Type, tt.typ)
			}
			if m.Version != tt.version {
				t.Errorf("Version = %q, want %q", m.Version, tt.version)
			}
		})
	}
}

func TestParse_FileNotFound(t *testing.T) {
	_, err := Parse(testPath("nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseFile_Context(t *testing.T) {
	result, err := ParseFile(testPath("valid-context.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*ContextManifest)
	if !ok {
		t.Fatalf("expected *ContextManifest, got %T", result)
	}
	if m.Name != "spring-boot-error-handling" {
		t.Errorf("Name = %q, want %q", m.Name, "spring-boot-error-handling")
	}
	if m.Format != "markdown" {
		t.Errorf("Format = %q, want %q", m.Format, "markdown")
	}
	if m.Tokens != 2400 {
		t.Errorf("Tokens = %d, want %d", m.Tokens, 2400)
	}
	if len(m.Sources) != 2 {
		t.Errorf("Sources len = %d, want 2", len(m.Sources))
	}
}

func TestParseFile_Persona(t *testing.T) {
	result, err := ParseFile(testPath("valid-persona.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*PersonaManifest)
	if !ok {
		t.Fatalf("expected *PersonaManifest, got %T", result)
	}
	if m.Name != "senior-java-dev" {
		t.Errorf("Name = %q, want %q", m.Name, "senior-java-dev")
	}
	if len(m.Expertise) != 3 {
		t.Errorf("Expertise len = %d, want 3", len(m.Expertise))
	}
	if m.Tone != "direct, pragmatic, opinionated" {
		t.Errorf("Tone = %q, want %q", m.Tone, "direct, pragmatic, opinionated")
	}
	if len(m.Context) != 2 {
		t.Errorf("Context len = %d, want 2", len(m.Context))
	}
	if m.Template != "persona.md" {
		t.Errorf("Template = %q, want %q", m.Template, "persona.md")
	}
}

func TestParseFile_Skill(t *testing.T) {
	result, err := ParseFile(testPath("valid-skill.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*SkillManifest)
	if !ok {
		t.Fatalf("expected *SkillManifest, got %T", result)
	}
	if m.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q", m.Runtime, "node")
	}
	if m.Topic != "scm" {
		t.Errorf("Topic = %q, want %q", m.Topic, "scm")
	}
	if len(m.CLIDependencies) != 1 {
		t.Fatalf("CLIDependencies len = %d, want 1", len(m.CLIDependencies))
	}
	if m.CLIDependencies[0].Name != "git" {
		t.Errorf("CLIDependencies[0].Name = %q, want %q", m.CLIDependencies[0].Name, "git")
	}
	if len(m.Inputs) != 2 {
		t.Errorf("Inputs len = %d, want 2", len(m.Inputs))
	}
	if m.Outputs == nil {
		t.Fatal("Outputs is nil, expected non-nil")
	}
	if m.Outputs.Format != "json" {
		t.Errorf("Outputs.Format = %q, want %q", m.Outputs.Format, "json")
	}
	if m.Registry == nil {
		t.Fatal("Registry is nil, expected non-nil")
	}
	if len(m.Registry.State) != 1 {
		t.Errorf("Registry.State len = %d, want 1", len(m.Registry.State))
	}
}

func TestParseFile_Workflow(t *testing.T) {
	result, err := ParseFile(testPath("valid-workflow.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*WorkflowManifest)
	if !ok {
		t.Fatalf("expected *WorkflowManifest, got %T", result)
	}
	if m.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q", m.Runtime, "node")
	}
	if len(m.Steps) != 2 {
		t.Fatalf("Steps len = %d, want 2", len(m.Steps))
	}
	if m.Steps[0].ID != "analyze-commits" {
		t.Errorf("Steps[0].ID = %q, want %q", m.Steps[0].ID, "analyze-commits")
	}
	if m.Steps[0].Skill != "skills/scm/git/commit-analyzer" {
		t.Errorf("Steps[0].Skill = %q, want %q", m.Steps[0].Skill, "skills/scm/git/commit-analyzer")
	}
}

func TestParseFile_Prompt(t *testing.T) {
	result, err := ParseFile(testPath("valid-prompt.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*PromptManifest)
	if !ok {
		t.Fatalf("expected *PromptManifest, got %T", result)
	}
	if m.Persona != "personas/senior-java-dev" {
		t.Errorf("Persona = %q, want %q", m.Persona, "personas/senior-java-dev")
	}
	if len(m.Skills) != 2 {
		t.Errorf("Skills len = %d, want 2", len(m.Skills))
	}
	if len(m.Workflows) != 1 {
		t.Errorf("Workflows len = %d, want 1", len(m.Workflows))
	}
	if m.Template != "prompt.hbs" {
		t.Errorf("Template = %q, want %q", m.Template, "prompt.hbs")
	}
}

func TestParseFile_Template(t *testing.T) {
	result, err := ParseFile(testPath("valid-template.yaml"))
	if err != nil {
		t.Fatalf("ParseFile error: %v", err)
	}
	m, ok := result.(*TemplateManifest)
	if !ok {
		t.Fatalf("expected *TemplateManifest, got %T", result)
	}
	if m.Format != "spl" {
		t.Errorf("Format = %q, want %q", m.Format, "spl")
	}
	if len(m.Variables) != 3 {
		t.Fatalf("Variables len = %d, want 3", len(m.Variables))
	}
	if m.Variables[0].Name != "index" {
		t.Errorf("Variables[0].Name = %q, want %q", m.Variables[0].Name, "index")
	}
	if m.Variables[0].Default != "main" {
		t.Errorf("Variables[0].Default = %q, want %q", m.Variables[0].Default, "main")
	}
}

func TestParseFile_TypeDispatch(t *testing.T) {
	tests := []struct {
		file     string
		expected string // type name of the returned struct
	}{
		{"valid-context.yaml", "*manifest.ContextManifest"},
		{"valid-persona.yaml", "*manifest.PersonaManifest"},
		{"valid-skill.yaml", "*manifest.SkillManifest"},
		{"valid-workflow.yaml", "*manifest.WorkflowManifest"},
		{"valid-prompt.yaml", "*manifest.PromptManifest"},
		{"valid-template.yaml", "*manifest.TemplateManifest"},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			result, err := ParseFile(testPath(tt.file))
			if err != nil {
				t.Fatalf("ParseFile(%s) error: %v", tt.file, err)
			}
			typeName := typeNameOf(result)
			if typeName != tt.expected {
				t.Errorf("type = %s, want %s", typeName, tt.expected)
			}
		})
	}
}

func TestParseFile_MissingType(t *testing.T) {
	_, err := ParseFile(testPath("invalid-missing-type.yaml"))
	if err == nil {
		t.Fatal("expected error for missing type field, got nil")
	}
}

func TestParseFile_BadType(t *testing.T) {
	_, err := ParseFile(testPath("invalid-bad-type.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid type value, got nil")
	}
}

func TestParseFile_InvalidYAML(t *testing.T) {
	_, err := ParseFile(testPath("invalid-not-yaml.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestParseFile_NotFound(t *testing.T) {
	_, err := ParseFile(testPath("nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseContext_Typed(t *testing.T) {
	m, err := ParseContext(testPath("valid-context.yaml"))
	if err != nil {
		t.Fatalf("ParseContext error: %v", err)
	}
	if m.Format != "markdown" {
		t.Errorf("Format = %q, want %q", m.Format, "markdown")
	}
	if len(m.Sources) != 2 {
		t.Errorf("Sources len = %d, want 2", len(m.Sources))
	}
}

func TestParseSkill_Typed(t *testing.T) {
	m, err := ParseSkill(testPath("valid-skill.yaml"))
	if err != nil {
		t.Fatalf("ParseSkill error: %v", err)
	}
	if m.Runtime != "node" {
		t.Errorf("Runtime = %q, want %q", m.Runtime, "node")
	}
	if m.Topic != "scm" {
		t.Errorf("Topic = %q, want %q", m.Topic, "scm")
	}
}

func TestParse_VendorNullable(t *testing.T) {
	// The valid-skill.yaml does not have a vendor field — should be nil.
	m, err := Parse(testPath("valid-skill.yaml"))
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if m.Vendor != nil {
		t.Errorf("Vendor = %v, want nil", *m.Vendor)
	}
}

// typeNameOf returns the type name of a value using fmt.Sprintf.
func typeNameOf(v interface{}) string {
	return fmt.Sprintf("%T", v)
}
