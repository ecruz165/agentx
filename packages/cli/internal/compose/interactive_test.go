package compose

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPersonaManifest = `name: senior-java-dev
type: persona
version: "1.0.0"
description: A senior Java developer
expertise: [Java, Spring Boot, testing]
tone: professional
conventions:
  - Use constructor injection
  - Prefer immutable objects
`

const testSkillManifest = `name: commit-analyzer
type: skill
version: "1.0.0"
description: Analyzes git commits
runtime: node
topic: scm
`

const testContextManifest = `name: spring-boot-conventions
type: context
version: "1.0.0"
description: Spring Boot conventions context
format: markdown
sources:
  - content.md
`

// setupInteractiveEnv creates a minimal installed types directory structure.
func setupInteractiveEnv(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()

	// Create persona.
	personaDir := filepath.Join(tmp, "personas", "senior-java-dev")
	os.MkdirAll(personaDir, 0755)
	os.WriteFile(filepath.Join(personaDir, "manifest.yaml"), []byte(testPersonaManifest), 0644)

	// Create skill.
	skillDir := filepath.Join(tmp, "skills", "scm", "git", "commit-analyzer")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(testSkillManifest), 0644)

	// Create context.
	ctxDir := filepath.Join(tmp, "context", "scm", "git-conventions")
	os.MkdirAll(ctxDir, 0755)
	os.WriteFile(filepath.Join(ctxDir, "manifest.yaml"), []byte(testContextManifest), 0644)
	os.WriteFile(filepath.Join(ctxDir, "content.md"), []byte("Use conventional commits."), 0644)

	return tmp
}

func TestRunInteractive_SelectsPersonaTopicIntent(t *testing.T) {
	installedRoot := setupInteractiveEnv(t)

	// Simulate user input: select persona #1, topic #1, intent "code-review".
	input := "1\n1\ncode-review\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	result, err := RunInteractive(installedRoot, reader, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PersonaPath != "personas/senior-java-dev" {
		t.Errorf("expected persona path 'personas/senior-java-dev', got %q", result.PersonaPath)
	}
	if result.Topic != "scm" {
		t.Errorf("expected topic 'scm', got %q", result.Topic)
	}
	if result.Intent != "code-review" {
		t.Errorf("expected intent 'code-review', got %q", result.Intent)
	}
}

func TestRunInteractive_EmptyIntentDefaultsToGeneral(t *testing.T) {
	installedRoot := setupInteractiveEnv(t)

	input := "1\n1\n\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	result, err := RunInteractive(installedRoot, reader, &output)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Intent != "general-guidance" {
		t.Errorf("expected default intent 'general-guidance', got %q", result.Intent)
	}
}

func TestRunInteractive_InvalidSelection(t *testing.T) {
	installedRoot := setupInteractiveEnv(t)

	input := "99\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	_, err := RunInteractive(installedRoot, reader, &output)
	if err == nil {
		t.Fatal("expected error for invalid selection")
	}
	if !strings.Contains(err.Error(), "invalid selection") {
		t.Errorf("expected 'invalid selection' error, got: %v", err)
	}
}

func TestRunInteractive_NoPersonasInstalled(t *testing.T) {
	tmp := t.TempDir()

	input := "1\n"
	reader := strings.NewReader(input)
	var output bytes.Buffer

	_, err := RunInteractive(tmp, reader, &output)
	if err == nil {
		t.Fatal("expected error for no installed personas")
	}
	if !strings.Contains(err.Error(), "no installed personas") {
		t.Errorf("expected 'no installed personas' error, got: %v", err)
	}
}

func TestComposeFromInteractive_ProducesOutput(t *testing.T) {
	installedRoot := setupInteractiveEnv(t)

	result := &InteractiveResult{
		PersonaPath: "personas/senior-java-dev",
		Topic:       "scm",
		Intent:      "code-review",
	}

	cp, err := ComposeFromInteractive(result, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cp.PromptName != "scm code-review" {
		t.Errorf("expected prompt name 'scm code-review', got %q", cp.PromptName)
	}

	if cp.Persona == nil {
		t.Fatal("expected persona to be set")
	}
	if cp.Persona.Name != "senior-java-dev" {
		t.Errorf("expected persona name 'senior-java-dev', got %q", cp.Persona.Name)
	}

	// Render it to make sure it produces valid markdown.
	output := Render(cp)
	if !strings.Contains(output, "senior-java-dev") {
		t.Errorf("expected rendered output to contain persona name, got:\n%s", output)
	}
}

func TestSelectFromList_ValidInput(t *testing.T) {
	input := "2\n"
	var output bytes.Buffer

	items := []string{"alpha", "beta", "gamma"}
	idx, err := selectFromList(
		bufio.NewReader(strings.NewReader(input)),
		&output,
		"Pick one:",
		items,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if idx != 1 {
		t.Errorf("expected index 1 (beta), got %d", idx)
	}
}

func TestMatchesTopic_PathMatch(t *testing.T) {
	// We can't easily construct registry.ResolvedType in tests without the
	// full registry infrastructure. matchesTopic is indirectly tested via
	// ComposeFromInteractive which picks matching skills and context.
	installedRoot := setupInteractiveEnv(t)

	result := &InteractiveResult{
		PersonaPath: "personas/senior-java-dev",
		Topic:       "scm",
		Intent:      "code-review",
	}

	cp, err := ComposeFromInteractive(result, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find the scm skill.
	if len(cp.Skills) == 0 {
		t.Error("expected at least one skill matching topic 'scm'")
	}
}

func TestListInstalledPrompts_Empty(t *testing.T) {
	tmp := t.TempDir()
	prompts, err := ListInstalledPrompts(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(prompts) != 0 {
		t.Errorf("expected 0 prompts, got %d", len(prompts))
	}
}
