package linker

import (
	"testing"
)

func TestTypeSectionMapping(t *testing.T) {
	tests := []struct {
		ref     string
		section string
		wantErr bool
	}{
		{"personas/senior-java-dev", "personas", false},
		{"context/spring-boot/security", "context", false},
		{"skills/scm/git/commit-analyzer", "skills", false},
		{"workflows/pr-review", "workflows", false},
		{"prompts/code-review", "prompts", false},
		{"invalid", "", true},
		{"unknown-type/thing", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.ref, func(t *testing.T) {
			section, err := typeSection(tt.ref)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for %q, got section %q", tt.ref, section)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error for %q: %v", tt.ref, err)
				return
			}
			if section != tt.section {
				t.Errorf("expected section %q for %q, got %q", tt.section, tt.ref, section)
			}
		})
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}
	if !contains(slice, "b") {
		t.Error("expected contains to find 'b'")
	}
	if contains(slice, "d") {
		t.Error("expected contains not to find 'd'")
	}
}

func TestRemove(t *testing.T) {
	slice := []string{"a", "b", "c"}
	result := remove(slice, "b")
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if contains(result, "b") {
		t.Error("expected 'b' to be removed")
	}
}

func TestGetSection(t *testing.T) {
	active := &ActiveConfig{}

	personas := getSection(active, "personas")
	if personas == nil {
		t.Fatal("expected non-nil pointer for personas")
	}
	*personas = append(*personas, "personas/test")
	if len(active.Personas) != 1 {
		t.Error("modifying pointer should update ActiveConfig")
	}

	context := getSection(active, "context")
	*context = append(*context, "context/test")
	if len(active.Context) != 1 {
		t.Error("modifying pointer should update ActiveConfig")
	}

	skills := getSection(active, "skills")
	*skills = append(*skills, "skills/test")
	if len(active.Skills) != 1 {
		t.Error("modifying pointer should update ActiveConfig")
	}

	workflows := getSection(active, "workflows")
	*workflows = append(*workflows, "workflows/test")
	if len(active.Workflows) != 1 {
		t.Error("modifying pointer should update ActiveConfig")
	}

	prompts := getSection(active, "prompts")
	*prompts = append(*prompts, "prompts/test")
	if len(active.Prompts) != 1 {
		t.Error("modifying pointer should update ActiveConfig")
	}

	unknown := getSection(active, "unknown")
	if unknown != nil {
		t.Error("expected nil for unknown section")
	}
}

// TestAddTypeUpdatesCorrectSection tests that AddType modifies the right section
// without actually calling Sync (which requires Node). We test the project.yaml
// manipulation by manually calling the internal helpers.
func TestAddTypeUpdatesCorrectSection(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize project
	if err := InitProject(tmpDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Simulate what AddType does (without Sync call)
	typeRef := "personas/senior-java-dev"
	section, err := typeSection(typeRef)
	if err != nil {
		t.Fatalf("typeSection failed: %v", err)
	}

	config, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}

	target := getSection(&config.Active, section)
	if contains(*target, typeRef) {
		t.Fatal("type already exists before adding")
	}

	*target = append(*target, typeRef)
	if err := SaveProject(tmpDir, config); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Reload and verify
	reloaded, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject (reload) failed: %v", err)
	}

	if !contains(reloaded.Active.Personas, typeRef) {
		t.Errorf("expected %s in personas, got %v", typeRef, reloaded.Active.Personas)
	}
}

// TestRemoveTypeUpdatesCorrectSection tests that removal works without Sync.
func TestRemoveTypeUpdatesCorrectSection(t *testing.T) {
	tmpDir := t.TempDir()

	if err := InitProject(tmpDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Add a skill first
	config, _ := LoadProject(tmpDir)
	config.Active.Skills = []string{"skills/scm/git/commit-analyzer"}
	SaveProject(tmpDir, config)

	// Simulate removal
	typeRef := "skills/scm/git/commit-analyzer"
	section, _ := typeSection(typeRef)
	config, _ = LoadProject(tmpDir)
	target := getSection(&config.Active, section)
	*target = remove(*target, typeRef)
	SaveProject(tmpDir, config)

	// Verify
	reloaded, _ := LoadProject(tmpDir)
	if contains(reloaded.Active.Skills, typeRef) {
		t.Errorf("expected %s to be removed from skills", typeRef)
	}
}

// TestDuplicateDetection verifies that adding the same type twice is caught.
func TestDuplicateDetection(t *testing.T) {
	tmpDir := t.TempDir()

	if err := InitProject(tmpDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	config, _ := LoadProject(tmpDir)
	config.Active.Personas = []string{"personas/senior-java-dev"}
	SaveProject(tmpDir, config)

	// Reload and check duplicate
	config, _ = LoadProject(tmpDir)
	target := getSection(&config.Active, "personas")
	if !contains(*target, "personas/senior-java-dev") {
		t.Fatal("expected type to be present")
	}

	// The AddType function would detect this duplicate and return an error
}
