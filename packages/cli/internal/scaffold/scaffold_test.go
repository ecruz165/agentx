package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentx-labs/agentx/internal/manifest"
)

func TestNewScaffoldData(t *testing.T) {
	t.Run("skill with vendor", func(t *testing.T) {
		d := NewScaffoldData("ssm-lookup", "skill", "cloud", "aws", "node")
		if d.Name != "ssm-lookup" {
			t.Errorf("Name = %q, want %q", d.Name, "ssm-lookup")
		}
		if d.SkillPath != "cloud/aws/ssm-lookup" {
			t.Errorf("SkillPath = %q, want %q", d.SkillPath, "cloud/aws/ssm-lookup")
		}
		if d.PackageName != "@agentx/skill-cloud-ssm-lookup" {
			t.Errorf("PackageName = %q, want %q", d.PackageName, "@agentx/skill-cloud-ssm-lookup")
		}
		if d.ModuleName != "github.com/agentx-labs/agentx-skill-ssm-lookup" {
			t.Errorf("ModuleName = %q, want %q", d.ModuleName, "github.com/agentx-labs/agentx-skill-ssm-lookup")
		}
	})

	t.Run("skill without vendor", func(t *testing.T) {
		d := NewScaffoldData("token-counter", "skill", "ai", "", "go")
		if d.SkillPath != "ai/token-counter" {
			t.Errorf("SkillPath = %q, want %q", d.SkillPath, "ai/token-counter")
		}
		if d.Vendor != "" {
			t.Errorf("Vendor = %q, want empty", d.Vendor)
		}
	})

	t.Run("year is populated", func(t *testing.T) {
		d := NewScaffoldData("test", "skill", "t", "", "node")
		if d.Year == 0 {
			t.Error("Year should not be zero")
		}
	})
}

func TestGenerateSkillNode(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-tool")

	data := NewScaffoldData("my-tool", "skill", "cloud", "aws", "node")
	result, err := Generate("skill", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify expected files.
	expectedFiles := []string{"Makefile", "index.mjs", "package.json", "skill.yaml"}
	assertFiles(t, result, expectedFiles)

	// Verify manifest content.
	manifestContent := readGenerated(t, outDir, "skill.yaml")
	assertContains(t, manifestContent, "name: my-tool")
	assertContains(t, manifestContent, "type: skill")
	assertContains(t, manifestContent, "runtime: node")
	assertContains(t, manifestContent, "topic: cloud")
	assertContains(t, manifestContent, "vendor: aws")

	// Verify index.mjs has registry pattern.
	indexContent := readGenerated(t, outDir, "index.mjs")
	assertContains(t, indexContent, "const SKILL_TOPIC  = 'cloud'")
	assertContains(t, indexContent, "const SKILL_VENDOR = 'aws'")
	assertContains(t, indexContent, "const SKILL_NAME   = 'my-tool'")
	assertContains(t, indexContent, "saveOutput")
	assertContains(t, indexContent, "readState")

	// Verify package.json.
	pkgContent := readGenerated(t, outDir, "package.json")
	assertContains(t, pkgContent, "@agentx/skill-cloud-my-tool")
	assertContains(t, pkgContent, "dotenv")
	assertContains(t, pkgContent, "yaml")

	// Verify manifest passes schema validation.
	assertManifestValid(t, outDir, "skill.yaml")

	// Verify no warnings.
	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateSkillNodeWithoutVendor(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "token-counter")

	data := NewScaffoldData("token-counter", "skill", "ai", "", "node")
	result, err := Generate("skill", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	// Verify manifest has no vendor field.
	manifestContent := readGenerated(t, outDir, "skill.yaml")
	assertContains(t, manifestContent, "name: token-counter")
	assertNotContains(t, manifestContent, "vendor:")

	// Verify index.mjs path uses topic/name (no vendor).
	indexContent := readGenerated(t, outDir, "index.mjs")
	assertContains(t, indexContent, "const SKILL_VENDOR = ''")
	assertContains(t, indexContent, "`${SKILL_TOPIC}/${SKILL_NAME}`")

	assertManifestValid(t, outDir, "skill.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateSkillGo(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-tool")

	data := NewScaffoldData("my-tool", "skill", "cloud", "aws", "go")
	result, err := Generate("skill", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"Makefile", "go.mod", "main.go", "package.json", "skill.yaml"}
	assertFiles(t, result, expectedFiles)

	// Verify main.go has correct constants.
	mainContent := readGenerated(t, outDir, "main.go")
	assertContains(t, mainContent, `skillTopic  = "cloud"`)
	assertContains(t, mainContent, `skillVendor = "aws"`)
	assertContains(t, mainContent, `skillName   = "my-tool"`)
	assertContains(t, mainContent, "SaveOutput")

	// Verify go.mod.
	goModContent := readGenerated(t, outDir, "go.mod")
	assertContains(t, goModContent, "github.com/agentx-labs/agentx-skill-my-tool")

	// Verify manifest.
	manifestContent := readGenerated(t, outDir, "skill.yaml")
	assertContains(t, manifestContent, "runtime: go")

	assertManifestValid(t, outDir, "skill.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateWorkflow(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-flow")

	data := NewScaffoldData("my-flow", "workflow", "", "", "node")
	result, err := Generate("workflow", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"Makefile", "index.mjs", "package.json", "workflow.yaml"}
	assertFiles(t, result, expectedFiles)

	manifestContent := readGenerated(t, outDir, "workflow.yaml")
	assertContains(t, manifestContent, "name: my-flow")
	assertContains(t, manifestContent, "type: workflow")
	assertContains(t, manifestContent, "runtime: node")
	assertContains(t, manifestContent, "steps:")

	assertManifestValid(t, outDir, "workflow.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGeneratePrompt(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-prompt")

	data := NewScaffoldData("my-prompt", "prompt", "", "", "")
	result, err := Generate("prompt", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"prompt.hbs", "prompt.yaml"}
	assertFiles(t, result, expectedFiles)

	manifestContent := readGenerated(t, outDir, "prompt.yaml")
	assertContains(t, manifestContent, "name: my-prompt")
	assertContains(t, manifestContent, "type: prompt")
	assertContains(t, manifestContent, "persona:")

	// Verify .hbs file is copied verbatim with Handlebars syntax intact.
	hbsContent := readGenerated(t, outDir, "prompt.hbs")
	assertContains(t, hbsContent, "{{!-- AgentX Prompt Template --}}")

	assertManifestValid(t, outDir, "prompt.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGeneratePersona(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-persona")

	data := NewScaffoldData("my-persona", "persona", "", "", "")
	result, err := Generate("persona", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"persona.yaml"}
	assertFiles(t, result, expectedFiles)

	manifestContent := readGenerated(t, outDir, "persona.yaml")
	assertContains(t, manifestContent, "name: my-persona")
	assertContains(t, manifestContent, "type: persona")
	assertContains(t, manifestContent, "expertise:")
	assertContains(t, manifestContent, "tone: professional")

	assertManifestValid(t, outDir, "persona.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateContext(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-docs")

	data := NewScaffoldData("my-docs", "context", "", "", "")
	result, err := Generate("context", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"content.md", "context.yaml"}
	assertFiles(t, result, expectedFiles)

	manifestContent := readGenerated(t, outDir, "context.yaml")
	assertContains(t, manifestContent, "name: my-docs")
	assertContains(t, manifestContent, "type: context")
	assertContains(t, manifestContent, "format: markdown")
	assertContains(t, manifestContent, "sources:")

	contentMd := readGenerated(t, outDir, "content.md")
	assertContains(t, contentMd, "# my-docs")

	assertManifestValid(t, outDir, "context.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateTemplate(t *testing.T) {
	dir := t.TempDir()
	outDir := filepath.Join(dir, "my-template")

	data := NewScaffoldData("my-template", "template", "", "", "")
	result, err := Generate("template", data, outDir)
	if err != nil {
		t.Fatalf("Generate() error: %v", err)
	}

	expectedFiles := []string{"template.hbs", "template.yaml"}
	assertFiles(t, result, expectedFiles)

	manifestContent := readGenerated(t, outDir, "template.yaml")
	assertContains(t, manifestContent, "name: my-template")
	assertContains(t, manifestContent, "type: template")
	assertContains(t, manifestContent, "format: handlebars")
	assertContains(t, manifestContent, "variables:")

	// Verify .hbs file is copied verbatim with Handlebars syntax intact.
	hbsContent := readGenerated(t, outDir, "template.hbs")
	assertContains(t, hbsContent, "{{title}}")
	assertContains(t, hbsContent, "{{#if content}}")

	assertManifestValid(t, outDir, "template.yaml")

	if len(result.Warnings) > 0 {
		t.Errorf("unexpected warnings: %v", result.Warnings)
	}
}

func TestGenerateInvalidTemplateSet(t *testing.T) {
	dir := t.TempDir()
	data := NewScaffoldData("test", "nonexistent", "", "", "")
	_, err := Generate("nonexistent", data, dir)
	if err == nil {
		t.Fatal("expected error for invalid template set")
	}
}

func TestGenerateNonEmptyDir(t *testing.T) {
	dir := t.TempDir()
	// Create an existing file in the output dir.
	os.WriteFile(filepath.Join(dir, "existing.txt"), []byte("hello"), 0644)

	data := NewScaffoldData("test", "persona", "", "", "")
	_, err := Generate("persona", data, dir)
	if err == nil {
		t.Fatal("expected error for non-empty output directory")
	}
	if !strings.Contains(err.Error(), "not empty") {
		t.Errorf("error should mention non-empty dir, got: %v", err)
	}
}

func TestTemplateSetName(t *testing.T) {
	tests := []struct {
		typeName string
		runtime  string
		want     string
	}{
		{"skill", "node", "skill-node"},
		{"skill", "go", "skill-go"},
		{"workflow", "node", "workflow"},
		{"prompt", "", "prompt"},
		{"persona", "", "persona"},
		{"context", "", "context"},
		{"template", "", "template"},
	}

	for _, tt := range tests {
		got := templateSetName(tt.typeName, tt.runtime)
		if got != tt.want {
			t.Errorf("templateSetName(%q, %q) = %q, want %q", tt.typeName, tt.runtime, got, tt.want)
		}
	}
}

// ─── Test Helpers ──────────────────────────────────────────────────

func readGenerated(t *testing.T, dir, filename string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("reading %s: %v", filename, err)
	}
	return string(data)
}

func assertFiles(t *testing.T, result *Result, expected []string) {
	t.Helper()
	if len(result.Files) != len(expected) {
		t.Errorf("got %d files %v, want %d files %v", len(result.Files), result.Files, len(expected), expected)
		return
	}
	for i, f := range expected {
		if result.Files[i] != f {
			t.Errorf("file[%d] = %q, want %q", i, result.Files[i], f)
		}
	}
}

func assertContains(t *testing.T, content, substr string) {
	t.Helper()
	if !strings.Contains(content, substr) {
		t.Errorf("content does not contain %q\n--- content ---\n%s", substr, content)
	}
}

func assertNotContains(t *testing.T, content, substr string) {
	t.Helper()
	if strings.Contains(content, substr) {
		t.Errorf("content should not contain %q", substr)
	}
}

func assertManifestValid(t *testing.T, dir, filename string) {
	t.Helper()
	result, err := manifest.ValidateFile(filepath.Join(dir, filename))
	if err != nil {
		t.Fatalf("manifest validation error: %v", err)
	}
	if !result.Valid {
		var msgs []string
		for _, issue := range result.Issues {
			msg := issue.Message
			if issue.Path != "" {
				msg = issue.Path + ": " + msg
			}
			msgs = append(msgs, msg)
		}
		t.Errorf("generated manifest %s is invalid:\n  %s", filename, strings.Join(msgs, "\n  "))
	}
}
