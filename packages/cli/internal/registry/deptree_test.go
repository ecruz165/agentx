package registry

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDependencyTreePrompt(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	root, err := BuildDependencyTree("prompts/test-prompt", sources, "")
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	if root.TypePath != "prompts/test-prompt" {
		t.Errorf("root.TypePath = %q, want %q", root.TypePath, "prompts/test-prompt")
	}
	if root.Category != "prompt" {
		t.Errorf("root.Category = %q, want %q", root.Category, "prompt")
	}

	// Should have children: persona, context, skill, workflow.
	if len(root.Children) != 4 {
		t.Fatalf("root has %d children, want 4", len(root.Children))
	}

	// Check child categories.
	categories := make(map[string]bool)
	for _, child := range root.Children {
		categories[child.Category] = true
	}
	for _, expected := range []string{"persona", "context", "skill", "workflow"} {
		if !categories[expected] {
			t.Errorf("missing child category %q", expected)
		}
	}
}

func TestBuildDependencyTreeDedup(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	root, err := BuildDependencyTree("prompts/test-prompt", sources, "")
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	// The prompt references context/test-topic/docs directly AND
	// through personas/test-persona. One should be deduped.
	dedupCount := countDeduped(root)
	if dedupCount == 0 {
		t.Error("expected at least one deduped node for context/test-topic/docs")
	}
}

func TestBuildDependencyTreeWorkflowDedupSkill(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	root, err := BuildDependencyTree("workflows/test-workflow", sources, "")
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	// Workflow references basic-skill twice in steps. Second should be deduped.
	if len(root.Children) != 2 {
		t.Fatalf("workflow has %d children, want 2", len(root.Children))
	}
	if !root.Children[1].Deduped {
		t.Error("second skill reference should be deduped")
	}
}

func TestBuildDependencyTreeInstalledDetection(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	// Create a fake installed directory for context.
	tmpDir := t.TempDir()
	installedCtx := filepath.Join(tmpDir, "context", "test-topic", "docs")
	if err := os.MkdirAll(installedCtx, 0755); err != nil {
		t.Fatal(err)
	}

	root, err := BuildDependencyTree("personas/test-persona", sources, tmpDir)
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	// The context child should be marked as installed.
	if len(root.Children) != 1 {
		t.Fatalf("persona has %d children, want 1", len(root.Children))
	}
	if !root.Children[0].Installed {
		t.Error("context child should be marked as installed")
	}
}

func TestFlattenTree(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	root, err := BuildDependencyTree("prompts/test-prompt", sources, "")
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	flat := FlattenTree(root)

	// Should include context, persona, skill, workflow, prompt (5 unique types).
	if len(flat) != 5 {
		names := make([]string, len(flat))
		for i, f := range flat {
			names[i] = f.TypePath
		}
		t.Fatalf("FlattenTree returned %d types %v, want 5", len(flat), names)
	}

	// Dependencies should come before dependents (topological order).
	// Context should be before persona. Skill should be before workflow.
	// Prompt should be last.
	indexMap := make(map[string]int)
	for i, f := range flat {
		indexMap[f.TypePath] = i
	}

	if indexMap["context/test-topic/docs"] >= indexMap["personas/test-persona"] {
		t.Error("context should come before persona in topological order")
	}
	if indexMap["skills/test/basic-skill"] >= indexMap["workflows/test-workflow"] {
		t.Error("skill should come before workflow in topological order")
	}
	if indexMap["prompts/test-prompt"] != len(flat)-1 {
		t.Error("prompt should be last in topological order")
	}
}

func TestFlattenTreeExcludesInstalled(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	// Mark context as already installed.
	tmpDir := t.TempDir()
	installedCtx := filepath.Join(tmpDir, "context", "test-topic", "docs")
	if err := os.MkdirAll(installedCtx, 0755); err != nil {
		t.Fatal(err)
	}

	root, err := BuildDependencyTree("personas/test-persona", sources, tmpDir)
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	flat := FlattenTree(root)

	// Only persona should be in the list (context is installed).
	if len(flat) != 1 {
		t.Fatalf("FlattenTree returned %d types, want 1 (context should be excluded)", len(flat))
	}
	if flat[0].TypePath != "personas/test-persona" {
		t.Errorf("expected persona, got %s", flat[0].TypePath)
	}
}

func TestExtractDependenciesContext(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	manifestPath := filepath.Join(catalogDir, "context", "test-topic", "docs", "manifest.yaml")

	deps, err := extractDependencies(manifestPath)
	if err != nil {
		t.Fatalf("extractDependencies: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("context should have 0 deps, got %d", len(deps))
	}
}

func TestExtractDependenciesSkill(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	manifestPath := filepath.Join(catalogDir, "skills", "test", "basic-skill", "manifest.yaml")

	deps, err := extractDependencies(manifestPath)
	if err != nil {
		t.Fatalf("extractDependencies: %v", err)
	}
	if len(deps) != 0 {
		t.Errorf("skill should have 0 type deps, got %d", len(deps))
	}
}

func TestExtractDependenciesPersona(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	manifestPath := filepath.Join(catalogDir, "personas", "test-persona", "manifest.yaml")

	deps, err := extractDependencies(manifestPath)
	if err != nil {
		t.Fatalf("extractDependencies: %v", err)
	}
	if len(deps) != 1 || deps[0] != "context/test-topic/docs" {
		t.Errorf("persona deps = %v, want [context/test-topic/docs]", deps)
	}
}

func TestExtractDependenciesWorkflow(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	manifestPath := filepath.Join(catalogDir, "workflows", "test-workflow", "manifest.yaml")

	deps, err := extractDependencies(manifestPath)
	if err != nil {
		t.Fatalf("extractDependencies: %v", err)
	}
	// Workflow has 2 steps referencing the same skill — returns all references.
	// Deduplication happens at the tree level via BuildDependencyTree.
	if len(deps) != 2 || deps[0] != "skills/test/basic-skill" || deps[1] != "skills/test/basic-skill" {
		t.Errorf("workflow deps = %v, want [skills/test/basic-skill skills/test/basic-skill]", deps)
	}
}

func TestExtractDependenciesPrompt(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	manifestPath := filepath.Join(catalogDir, "prompts", "test-prompt", "manifest.yaml")

	deps, err := extractDependencies(manifestPath)
	if err != nil {
		t.Fatalf("extractDependencies: %v", err)
	}
	// Prompt: 1 persona + 1 context + 1 skill + 1 workflow = 4 deps.
	if len(deps) != 4 {
		t.Errorf("prompt deps = %v, want 4 items", deps)
	}
}

func TestPrintTree(t *testing.T) {
	node := &DependencyNode{
		TypePath: "prompts/test-prompt",
		Category: "prompt",
		Children: []*DependencyNode{
			{
				TypePath: "personas/test-persona",
				Category: "persona",
				Children: []*DependencyNode{
					{TypePath: "context/test-topic/docs", Category: "context"},
				},
			},
			{
				TypePath:  "context/test-topic/docs",
				Category:  "context",
				Deduped:   true,
			},
			{
				TypePath:  "skills/test/basic-skill",
				Category:  "skill",
				Installed: true,
			},
		},
	}

	var buf bytes.Buffer
	PrintTree(&buf, node, "", true)
	output := buf.String()

	if !strings.Contains(output, "prompt: test-prompt") {
		t.Error("output should contain root node")
	}
	if !strings.Contains(output, "(deduped)") {
		t.Error("output should contain deduped marker")
	}
	if !strings.Contains(output, "(already installed)") {
		t.Error("output should contain already installed marker")
	}
}

func countDeduped(node *DependencyNode) int {
	if node == nil {
		return 0
	}
	count := 0
	if node.Deduped {
		count = 1
	}
	for _, child := range node.Children {
		count += countDeduped(child)
	}
	return count
}
