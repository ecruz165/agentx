package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTypeFindsInFirstSource(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	resolved, err := ResolveType("personas/test-persona", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if resolved.TypePath != "personas/test-persona" {
		t.Errorf("TypePath = %q, want %q", resolved.TypePath, "personas/test-persona")
	}
	if resolved.Category != "persona" {
		t.Errorf("Category = %q, want %q", resolved.Category, "persona")
	}
	if resolved.SourceName != "catalog" {
		t.Errorf("SourceName = %q, want %q", resolved.SourceName, "catalog")
	}
	if filepath.Base(resolved.ManifestPath) != "manifest.yaml" {
		t.Errorf("ManifestPath base = %q, want %q", filepath.Base(resolved.ManifestPath), "manifest.yaml")
	}
}

func TestResolveTypeFallsThrough(t *testing.T) {
	// Create a second source with a persona that doesn't exist in the first.
	tmpDir := t.TempDir()
	extDir := filepath.Join(tmpDir, "ext", "personas", "ext-persona")
	if err := os.MkdirAll(extDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(extDir, "manifest.yaml"), []byte("name: ext-persona\ntype: persona\nversion: \"1.0.0\"\ndescription: ext\n"), 0644); err != nil {
		t.Fatal(err)
	}

	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{
		{Name: "catalog", BasePath: catalogDir},
		{Name: "ext", BasePath: filepath.Join(tmpDir, "ext")},
	}

	resolved, err := ResolveType("personas/ext-persona", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if resolved.SourceName != "ext" {
		t.Errorf("SourceName = %q, want %q", resolved.SourceName, "ext")
	}
}

func TestResolveTypePriorityOrder(t *testing.T) {
	// Create an override in a higher-priority source.
	tmpDir := t.TempDir()
	overrideDir := filepath.Join(tmpDir, "override", "personas", "test-persona")
	if err := os.MkdirAll(overrideDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overrideDir, "manifest.yaml"), []byte("name: test-persona\ntype: persona\nversion: \"2.0.0\"\ndescription: override\n"), 0644); err != nil {
		t.Fatal(err)
	}

	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{
		{Name: "override", BasePath: filepath.Join(tmpDir, "override")},
		{Name: "catalog", BasePath: catalogDir},
	}

	resolved, err := ResolveType("personas/test-persona", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if resolved.SourceName != "override" {
		t.Errorf("SourceName = %q, want %q (higher priority)", resolved.SourceName, "override")
	}
}

func TestResolveTypeManifestFallback(t *testing.T) {
	tmpDir := t.TempDir()

	// Test manifest.json fallback.
	jsonDir := filepath.Join(tmpDir, "src", "context", "json-ctx")
	if err := os.MkdirAll(jsonDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(jsonDir, "manifest.json"), []byte(`{"name":"json-ctx","type":"context","version":"1.0.0","description":"json","format":"md","sources":["./"]}`), 0644); err != nil {
		t.Fatal(err)
	}

	sources := []Source{{Name: "src", BasePath: filepath.Join(tmpDir, "src")}}
	resolved, err := ResolveType("context/json-ctx", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if filepath.Base(resolved.ManifestPath) != "manifest.json" {
		t.Errorf("ManifestPath base = %q, want %q", filepath.Base(resolved.ManifestPath), "manifest.json")
	}

	// Test <type>.yaml fallback.
	typeDir := filepath.Join(tmpDir, "src", "skills", "typed-skill")
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typeDir, "skill.yaml"), []byte("name: typed-skill\ntype: skill\nversion: \"1.0.0\"\ndescription: typed\nruntime: node\ntopic: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	resolved2, err := ResolveType("skills/typed-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType type fallback: %v", err)
	}
	if filepath.Base(resolved2.ManifestPath) != "skill.yaml" {
		t.Errorf("ManifestPath base = %q, want %q", filepath.Base(resolved2.ManifestPath), "skill.yaml")
	}
}

func TestResolveTypeNotFound(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	_, err := ResolveType("personas/nonexistent", sources)
	if err == nil {
		t.Fatal("expected error for nonexistent type")
	}
}

func TestCategoryFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"personas/senior-java-dev", "persona"},
		{"skills/scm/git/commit-analyzer", "skill"},
		{"context/spring-boot/security", "context"},
		{"workflows/deploy-verify", "workflow"},
		{"prompts/code-review/java-pr-review", "prompt"},
		{"templates/skill-template", "template"},
		{"unknown/something", ""},
	}

	for _, tt := range tests {
		got := categoryFromPath(tt.path)
		if got != tt.expected {
			t.Errorf("categoryFromPath(%q) = %q, want %q", tt.path, got, tt.expected)
		}
	}
}

func testdataDir() string {
	return filepath.Join("testdata")
}
