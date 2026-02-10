package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverAllWithTestdata(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	if len(discovered) == 0 {
		t.Fatal("DiscoverAll returned no types, expected at least one")
	}

	// Verify we found types from each known category in testdata.
	categories := make(map[string]bool)
	for _, dt := range discovered {
		categories[dt.Category] = true
	}

	expectedCategories := []string{"persona", "skill", "context", "workflow", "prompt"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("expected category %q not found in discovered types", cat)
		}
	}
}

func TestDiscoverAllEnrichesMetadata(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	// Find the test-persona and verify its metadata.
	var persona *DiscoveredType
	for i := range discovered {
		if discovered[i].TypePath == "personas/test-persona" {
			persona = &discovered[i]
			break
		}
	}

	if persona == nil {
		t.Fatal("test-persona not found in discovered types")
	}

	if persona.Name != "test-persona" {
		t.Errorf("Name = %q, want %q", persona.Name, "test-persona")
	}
	if persona.Version != "1.0.0" {
		t.Errorf("Version = %q, want %q", persona.Version, "1.0.0")
	}
	if persona.Description != "A test persona for unit testing" {
		t.Errorf("Description = %q, want %q", persona.Description, "A test persona for unit testing")
	}
	if persona.Category != "persona" {
		t.Errorf("Category = %q, want %q", persona.Category, "persona")
	}
	if persona.Source != "catalog" {
		t.Errorf("Source = %q, want %q", persona.Source, "catalog")
	}
}

func TestDiscoverAllWithNestedDirectories(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")
	sources := []Source{{Name: "catalog", BasePath: catalogDir}}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	// The testdata has skills/test/basic-skill (nested) and context/test-topic/docs (nested).
	var foundSkill, foundContext bool
	for _, dt := range discovered {
		if dt.TypePath == "skills/test/basic-skill" {
			foundSkill = true
		}
		if dt.TypePath == "context/test-topic/docs" {
			foundContext = true
		}
	}

	if !foundSkill {
		t.Error("expected nested skill 'skills/test/basic-skill' not found")
	}
	if !foundContext {
		t.Error("expected nested context 'context/test-topic/docs' not found")
	}
}

func TestDiscoverAllSkipsDirectoriesWithoutManifests(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a skills directory with a subdirectory that has no manifest.
	noManifestDir := filepath.Join(tmpDir, "skills", "empty-skill")
	if err := os.MkdirAll(noManifestDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Create a random file that is not a manifest.
	if err := os.WriteFile(filepath.Join(noManifestDir, "README.md"), []byte("# Not a manifest"), 0644); err != nil {
		t.Fatal(err)
	}

	sources := []Source{{Name: "test", BasePath: tmpDir}}
	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("expected 0 discovered types, got %d", len(discovered))
	}
}

func TestDiscoverAllMultipleSources(t *testing.T) {
	catalogDir := filepath.Join(testdataDir(), "catalog")

	// Create a second source with an additional type.
	tmpDir := t.TempDir()
	extPersonaDir := filepath.Join(tmpDir, "ext", "personas", "ext-persona")
	if err := os.MkdirAll(extPersonaDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "name: ext-persona\ntype: persona\nversion: \"2.0.0\"\ndescription: Extension persona\ntags:\n  - extension\n  - testing\n"
	if err := os.WriteFile(filepath.Join(extPersonaDir, "manifest.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	sources := []Source{
		{Name: "catalog", BasePath: catalogDir},
		{Name: "ext", BasePath: filepath.Join(tmpDir, "ext")},
	}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	var foundExtPersona bool
	for _, dt := range discovered {
		if dt.TypePath == "personas/ext-persona" {
			foundExtPersona = true
			if dt.Source != "ext" {
				t.Errorf("ext-persona Source = %q, want %q", dt.Source, "ext")
			}
			if dt.Version != "2.0.0" {
				t.Errorf("ext-persona Version = %q, want %q", dt.Version, "2.0.0")
			}
			if len(dt.Tags) != 2 {
				t.Errorf("ext-persona Tags length = %d, want 2", len(dt.Tags))
			}
		}
	}

	if !foundExtPersona {
		t.Error("ext-persona not found in discovered types")
	}
}

func TestDiscoverAllPriorityDedup(t *testing.T) {
	// Create two sources with the same type path. The first source should win.
	tmpDir := t.TempDir()

	for _, src := range []struct {
		name    string
		version string
		desc    string
	}{
		{"first", "1.0.0", "First source version"},
		{"second", "2.0.0", "Second source version"},
	} {
		dir := filepath.Join(tmpDir, src.name, "personas", "dup-persona")
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
		m := "name: dup-persona\ntype: persona\nversion: \"" + src.version + "\"\ndescription: " + src.desc + "\n"
		if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(m), 0644); err != nil {
			t.Fatal(err)
		}
	}

	sources := []Source{
		{Name: "first", BasePath: filepath.Join(tmpDir, "first")},
		{Name: "second", BasePath: filepath.Join(tmpDir, "second")},
	}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll: %v", err)
	}

	count := 0
	for _, dt := range discovered {
		if dt.TypePath == "personas/dup-persona" {
			count++
			if dt.Source != "first" {
				t.Errorf("dup-persona Source = %q, want %q (first source has priority)", dt.Source, "first")
			}
			if dt.Version != "1.0.0" {
				t.Errorf("dup-persona Version = %q, want %q", dt.Version, "1.0.0")
			}
		}
	}

	if count != 1 {
		t.Errorf("expected 1 entry for dup-persona, got %d", count)
	}
}

func TestDiscoverAllHandlesInaccessibleSource(t *testing.T) {
	sources := []Source{
		{Name: "missing", BasePath: "/nonexistent/path/that/does/not/exist"},
	}

	discovered, err := DiscoverAll(sources)
	if err != nil {
		t.Fatalf("DiscoverAll should not error for inaccessible sources: %v", err)
	}

	if len(discovered) != 0 {
		t.Errorf("expected 0 discovered types for inaccessible source, got %d", len(discovered))
	}
}
