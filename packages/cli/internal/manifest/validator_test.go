package manifest

import (
	"testing"
)

func TestValidateFile_ValidManifests(t *testing.T) {
	validFiles := []string{
		"valid-context.yaml",
		"valid-persona.yaml",
		"valid-skill.yaml",
		"valid-workflow.yaml",
		"valid-prompt.yaml",
		"valid-template.yaml",
	}

	for _, file := range validFiles {
		t.Run(file, func(t *testing.T) {
			result, err := ValidateFile(testPath(file))
			if err != nil {
				t.Fatalf("ValidateFile(%s) error: %v", file, err)
			}
			if !result.Valid {
				t.Errorf("expected valid, got invalid with %d issues:", len(result.Issues))
				for _, issue := range result.Issues {
					t.Errorf("  path=%s keyword=%s message=%s", issue.Path, issue.Keyword, issue.Message)
				}
			}
		})
	}
}

func TestValidateFile_InvalidManifests(t *testing.T) {
	invalidFiles := []struct {
		file string
		desc string
	}{
		{"invalid-missing-name.yaml", "missing required name field"},
		{"invalid-bad-type.yaml", "invalid type value"},
		{"invalid-bad-name-pattern.yaml", "name violates pattern"},
		{"invalid-skill-missing-runtime.yaml", "skill missing required runtime"},
	}

	for _, tt := range invalidFiles {
		t.Run(tt.file, func(t *testing.T) {
			result, err := ValidateFile(testPath(tt.file))
			if err != nil {
				t.Fatalf("ValidateFile(%s) unexpected error: %v", tt.file, err)
			}
			if result.Valid {
				t.Errorf("expected invalid for %s (%s), but got valid", tt.file, tt.desc)
			}
			if len(result.Issues) == 0 {
				t.Errorf("expected at least one issue for %s (%s)", tt.file, tt.desc)
			}
		})
	}
}

func TestValidateFile_InvalidYAML(t *testing.T) {
	_, err := ValidateFile(testPath("invalid-not-yaml.yaml"))
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
}

func TestValidateFile_NotFound(t *testing.T) {
	_, err := ValidateFile(testPath("nonexistent.yaml"))
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestValidate_IssueFields(t *testing.T) {
	// Test that validation issues have populated fields.
	result, err := ValidateFile(testPath("invalid-bad-name-pattern.yaml"))
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid result")
	}
	if len(result.Issues) == 0 {
		t.Fatal("expected at least one issue")
	}

	// At least one issue should have a non-empty message.
	hasMessage := false
	for _, issue := range result.Issues {
		if issue.Message != "" {
			hasMessage = true
			break
		}
	}
	if !hasMessage {
		t.Error("expected at least one issue with a non-empty message")
	}
}

func TestValidate_MissingType(t *testing.T) {
	// Missing type field should fail schema validation (not just parsing).
	result, err := ValidateFile(testPath("invalid-missing-type.yaml"))
	if err != nil {
		t.Fatalf("ValidateFile error: %v", err)
	}
	if result.Valid {
		t.Error("expected invalid result for missing type field")
	}
}

func TestValidate_SchemaCompiles(t *testing.T) {
	// Verify the embedded schema can be compiled.
	schema, err := getSchema()
	if err != nil {
		t.Fatalf("getSchema() error: %v", err)
	}
	if schema == nil {
		t.Fatal("getSchema() returned nil schema")
	}
}
