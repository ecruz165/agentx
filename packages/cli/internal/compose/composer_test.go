package compose

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// createTestFixture builds a temporary directory tree that mimics the
// ~/.agentx/installed/ layout with prompt, persona, context, skill,
// and workflow manifests.
func createTestFixture(t *testing.T) string {
	t.Helper()

	root := t.TempDir()

	// --- Persona: personas/senior-java-dev ---
	writeFile(t, root, "personas/senior-java-dev/manifest.yaml", `
name: senior-java-dev
type: persona
version: "1.0.0"
description: Senior Java developer with Spring Boot expertise
expertise:
  - Java 17+
  - Spring Boot 3.x
  - Spring Security
  - JUnit 5
tone: professional
conventions:
  - Use constructor injection over field injection
  - Prefer records for DTOs
  - Always include proper error handling
context:
  - context/spring-boot/error-handling
  - context/spring-boot/security
`)

	// --- Context: context/spring-boot/error-handling ---
	writeFile(t, root, "context/spring-boot/error-handling/manifest.yaml", `
name: error-handling
type: context
version: "1.0.0"
description: Spring Boot error handling patterns
format: markdown
sources:
  - content.md
`)
	writeFile(t, root, "context/spring-boot/error-handling/content.md",
		"Use @ControllerAdvice for global exception handling.\n")

	// --- Context: context/spring-boot/security ---
	writeFile(t, root, "context/spring-boot/security/manifest.yaml", `
name: security
type: context
version: "1.0.0"
description: Spring Security configuration patterns
format: markdown
sources:
  - content.md
`)
	writeFile(t, root, "context/spring-boot/security/content.md",
		"Use SecurityFilterChain for HTTP security config.\n")

	// --- Skill: skills/scm/git/commit-analyzer ---
	writeFile(t, root, "skills/scm/git/commit-analyzer/manifest.yaml", `
name: commit-analyzer
type: skill
version: "1.0.0"
description: Analyzes git commit history for patterns and issues
runtime: node
topic: scm
`)

	// --- Workflow: workflows/code-review ---
	writeFile(t, root, "workflows/code-review/manifest.yaml", `
name: code-review
type: workflow
version: "1.0.0"
description: Automated code review workflow
runtime: node
steps:
  - id: analyze-commits
    skill: skills/scm/git/commit-analyzer
`)

	// --- Prompt: prompts/java-pr-review ---
	writeFile(t, root, "prompts/java-pr-review/manifest.yaml", `
name: java-pr-review
type: prompt
version: "1.0.0"
description: Java PR review prompt combining persona, context, and skills
persona: personas/senior-java-dev
context:
  - context/spring-boot/error-handling
  - context/spring-boot/security
skills:
  - skills/scm/git/commit-analyzer
workflows:
  - workflows/code-review
`)

	return root
}

// writeFile creates a file at the given relative path under root, creating
// intermediate directories as needed.
func writeFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	fullPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		t.Fatalf("creating directory for %s: %v", relPath, err)
	}
	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", relPath, err)
	}
}

func TestCompose_FullPrompt(t *testing.T) {
	root := createTestFixture(t)

	cp, err := Compose("prompts/java-pr-review", root)
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}

	if len(cp.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", cp.Warnings)
	}

	if cp.PromptName != "java-pr-review" {
		t.Errorf("PromptName = %q, want %q", cp.PromptName, "java-pr-review")
	}

	// Persona
	if cp.Persona == nil {
		t.Fatal("Persona is nil, expected non-nil")
	}
	if cp.Persona.Name != "senior-java-dev" {
		t.Errorf("Persona.Name = %q, want %q", cp.Persona.Name, "senior-java-dev")
	}
	if len(cp.Persona.Expertise) != 4 {
		t.Errorf("Persona.Expertise len = %d, want 4", len(cp.Persona.Expertise))
	}
	if cp.Persona.Tone != "professional" {
		t.Errorf("Persona.Tone = %q, want %q", cp.Persona.Tone, "professional")
	}
	if len(cp.Persona.Conventions) != 3 {
		t.Errorf("Persona.Conventions len = %d, want 3", len(cp.Persona.Conventions))
	}

	// Context
	if len(cp.Context) != 2 {
		t.Fatalf("Context len = %d, want 2", len(cp.Context))
	}
	if cp.Context[0].Name != "Error Handling" {
		t.Errorf("Context[0].Name = %q, want %q", cp.Context[0].Name, "Error Handling")
	}
	if !strings.Contains(cp.Context[0].Content, "@ControllerAdvice") {
		t.Errorf("Context[0].Content does not contain '@ControllerAdvice'")
	}
	if cp.Context[1].Name != "Security" {
		t.Errorf("Context[1].Name = %q, want %q", cp.Context[1].Name, "Security")
	}

	// Skills
	if len(cp.Skills) != 1 {
		t.Fatalf("Skills len = %d, want 1", len(cp.Skills))
	}
	if cp.Skills[0].Name != "commit-analyzer" {
		t.Errorf("Skills[0].Name = %q, want %q", cp.Skills[0].Name, "commit-analyzer")
	}
	if cp.Skills[0].Description != "Analyzes git commit history for patterns and issues" {
		t.Errorf("Skills[0].Description = %q", cp.Skills[0].Description)
	}

	// Workflows
	if len(cp.Workflows) != 1 {
		t.Fatalf("Workflows len = %d, want 1", len(cp.Workflows))
	}
	if cp.Workflows[0].Name != "code-review" {
		t.Errorf("Workflows[0].Name = %q, want %q", cp.Workflows[0].Name, "code-review")
	}
}

func TestCompose_MissingPersona(t *testing.T) {
	root := t.TempDir()

	// Create a prompt that references a persona that doesn't exist.
	writeFile(t, root, "prompts/test-prompt/manifest.yaml", `
name: test-prompt
type: prompt
version: "1.0.0"
description: Test prompt with missing persona
persona: personas/nonexistent
`)

	cp, err := Compose("prompts/test-prompt", root)
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}

	if cp.Persona != nil {
		t.Errorf("expected Persona to be nil for missing persona, got %+v", cp.Persona)
	}

	if len(cp.Warnings) == 0 {
		t.Error("expected at least one warning for missing persona")
	}

	found := false
	for _, w := range cp.Warnings {
		if strings.Contains(w, "nonexistent") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected warning mentioning 'nonexistent', got: %v", cp.Warnings)
	}
}

func TestCompose_MissingContext(t *testing.T) {
	root := t.TempDir()

	writeFile(t, root, "prompts/test-prompt/manifest.yaml", `
name: test-prompt
type: prompt
version: "1.0.0"
description: Test prompt with missing context
context:
  - context/missing-topic
`)

	cp, err := Compose("prompts/test-prompt", root)
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}

	if len(cp.Context) != 0 {
		t.Errorf("Context len = %d, want 0", len(cp.Context))
	}

	if len(cp.Warnings) == 0 {
		t.Error("expected at least one warning for missing context")
	}
}

func TestCompose_MissingContextSourceFile(t *testing.T) {
	root := t.TempDir()

	// Context manifest references a source file that doesn't exist.
	writeFile(t, root, "context/broken/manifest.yaml", `
name: broken
type: context
version: "1.0.0"
description: Context with missing source file
format: markdown
sources:
  - missing-content.md
`)

	writeFile(t, root, "prompts/test-prompt/manifest.yaml", `
name: test-prompt
type: prompt
version: "1.0.0"
description: Test prompt with broken context source
context:
  - context/broken
`)

	cp, err := Compose("prompts/test-prompt", root)
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}

	if len(cp.Context) != 0 {
		t.Errorf("Context len = %d, want 0 (source file is missing)", len(cp.Context))
	}

	if len(cp.Warnings) == 0 {
		t.Error("expected warning for missing context source file")
	}
}

func TestCompose_NoPersona(t *testing.T) {
	root := t.TempDir()

	// Prompt with no persona at all (not missing, just not referenced).
	writeFile(t, root, "prompts/simple/manifest.yaml", `
name: simple
type: prompt
version: "1.0.0"
description: Simple prompt with no persona
`)

	cp, err := Compose("prompts/simple", root)
	if err != nil {
		t.Fatalf("Compose error: %v", err)
	}

	if cp.Persona != nil {
		t.Errorf("expected nil persona, got %+v", cp.Persona)
	}
	if len(cp.Warnings) != 0 {
		t.Errorf("unexpected warnings: %v", cp.Warnings)
	}
}

func TestCompose_PromptNotInstalled(t *testing.T) {
	root := t.TempDir()

	_, err := Compose("prompts/not-installed", root)
	if err == nil {
		t.Fatal("expected error for non-installed prompt, got nil")
	}
}

func TestRender_FullOutput(t *testing.T) {
	cp := &ComposedPrompt{
		PromptName: "java-pr-review",
		Persona: &PersonaSection{
			Name:      "Senior Java Developer",
			Expertise: []string{"Java 17+", "Spring Boot 3.x"},
			Tone:      "professional",
			Conventions: []string{
				"Use constructor injection over field injection",
				"Prefer records for DTOs",
			},
		},
		Context: []ContextSection{
			{Name: "Error Handling", Content: "Handle errors with @ControllerAdvice.\n"},
			{Name: "Security", Content: "Use SecurityFilterChain.\n"},
		},
		Skills: []SkillRef{
			{Name: "commit-analyzer", Description: "Analyzes git commit history"},
		},
		Workflows: []WorkflowRef{
			{Name: "code-review", Description: "Automated code review workflow"},
		},
	}

	output := Render(cp)

	// Check persona section.
	if !strings.Contains(output, "# Persona: Senior Java Developer") {
		t.Error("output missing persona heading")
	}
	if !strings.Contains(output, "Expertise: Java 17+, Spring Boot 3.x") {
		t.Error("output missing expertise line")
	}
	if !strings.Contains(output, "Tone: professional") {
		t.Error("output missing tone line")
	}
	if !strings.Contains(output, "## Conventions") {
		t.Error("output missing conventions heading")
	}
	if !strings.Contains(output, "- Use constructor injection over field injection") {
		t.Error("output missing convention item")
	}

	// Check context section.
	if !strings.Contains(output, "## Context") {
		t.Error("output missing context heading")
	}
	if !strings.Contains(output, "### Error Handling") {
		t.Error("output missing error handling subheading")
	}
	if !strings.Contains(output, "@ControllerAdvice") {
		t.Error("output missing error handling content")
	}
	if !strings.Contains(output, "### Security") {
		t.Error("output missing security subheading")
	}

	// Check skills section.
	if !strings.Contains(output, "## Available Skills") {
		t.Error("output missing skills heading")
	}
	if !strings.Contains(output, "- commit-analyzer: Analyzes git commit history") {
		t.Error("output missing skill entry")
	}

	// Check workflows section.
	if !strings.Contains(output, "## Available Workflows") {
		t.Error("output missing workflows heading")
	}
	if !strings.Contains(output, "- code-review: Automated code review workflow") {
		t.Error("output missing workflow entry")
	}
}

func TestRender_EmptyPrompt(t *testing.T) {
	cp := &ComposedPrompt{
		PromptName: "empty-prompt",
	}

	output := Render(cp)

	// An empty prompt should produce an empty string (no sections).
	if output != "" {
		t.Errorf("expected empty output for empty prompt, got:\n%s", output)
	}
}

func TestRender_PersonaOnly(t *testing.T) {
	cp := &ComposedPrompt{
		PromptName: "persona-only",
		Persona: &PersonaSection{
			Name:      "Test Dev",
			Expertise: []string{"Go"},
			Tone:      "casual",
		},
	}

	output := Render(cp)

	if !strings.Contains(output, "# Persona: Test Dev") {
		t.Error("output missing persona heading")
	}
	if strings.Contains(output, "## Context") {
		t.Error("output should not contain context section")
	}
	if strings.Contains(output, "## Available Skills") {
		t.Error("output should not contain skills section")
	}
	if strings.Contains(output, "## Conventions") {
		t.Error("output should not contain conventions section when empty")
	}
}

func TestRender_SkillWithNoDescription(t *testing.T) {
	cp := &ComposedPrompt{
		PromptName: "test",
		Skills: []SkillRef{
			{Name: "my-skill", Description: ""},
		},
	}

	output := Render(cp)

	if !strings.Contains(output, "- my-skill\n") {
		t.Error("skill with no description should render without colon")
	}
	if strings.Contains(output, "- my-skill:") {
		t.Error("skill with no description should not have a colon")
	}
}

func TestFormatContextName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"error-handling", "Error Handling"},
		{"security", "Security"},
		{"spring-boot-config", "Spring Boot Config"},
		{"a", "A"},
		{"", ""},
	}

	for _, tt := range tests {
		got := formatContextName(tt.input)
		if got != tt.want {
			t.Errorf("formatContextName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
