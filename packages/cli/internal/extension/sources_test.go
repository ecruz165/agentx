package extension

import (
	"path/filepath"
	"testing"
)

func TestBuildSources_DefaultOrder(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "alpha", Path: "extensions/alpha", Source: "https://github.com/a/alpha.git", Branch: "main"},
			{Name: "beta", Path: "extensions/beta", Source: "https://github.com/b/beta.git", Branch: "develop"},
		},
		Resolution: ResolutionConfig{
			Order: []string{"local", "catalog", "extensions"},
		},
	}

	repoRoot := "/repo"
	sources := BuildSources(cfg, repoRoot)

	if len(sources) != 4 {
		t.Fatalf("expected 4 sources (local + catalog + 2 extensions), got %d", len(sources))
	}

	// Source 0: local
	if sources[0].Name != "local" {
		t.Errorf("sources[0].Name = %q, want 'local'", sources[0].Name)
	}
	if sources[0].BasePath != "." {
		t.Errorf("sources[0].BasePath = %q, want '.'", sources[0].BasePath)
	}

	// Source 1: catalog
	if sources[1].Name != "catalog" {
		t.Errorf("sources[1].Name = %q, want 'catalog'", sources[1].Name)
	}
	expectedCatalog := filepath.Join(repoRoot, "catalog")
	if sources[1].BasePath != expectedCatalog {
		t.Errorf("sources[1].BasePath = %q, want %q", sources[1].BasePath, expectedCatalog)
	}

	// Source 2: alpha (first extension)
	if sources[2].Name != "alpha" {
		t.Errorf("sources[2].Name = %q, want 'alpha'", sources[2].Name)
	}
	expectedAlpha := filepath.Join(repoRoot, "extensions/alpha")
	if sources[2].BasePath != expectedAlpha {
		t.Errorf("sources[2].BasePath = %q, want %q", sources[2].BasePath, expectedAlpha)
	}

	// Source 3: beta (second extension)
	if sources[3].Name != "beta" {
		t.Errorf("sources[3].Name = %q, want 'beta'", sources[3].Name)
	}
	expectedBeta := filepath.Join(repoRoot, "extensions/beta")
	if sources[3].BasePath != expectedBeta {
		t.Errorf("sources[3].BasePath = %q, want %q", sources[3].BasePath, expectedBeta)
	}
}

func TestBuildSources_NoExtensions(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{},
		Resolution: ResolutionConfig{
			Order: []string{"local", "catalog", "extensions"},
		},
	}

	sources := BuildSources(cfg, "/repo")

	// "extensions" with empty list should contribute 0 sources.
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources (local + catalog), got %d", len(sources))
	}

	if sources[0].Name != "local" {
		t.Errorf("sources[0].Name = %q, want 'local'", sources[0].Name)
	}
	if sources[1].Name != "catalog" {
		t.Errorf("sources[1].Name = %q, want 'catalog'", sources[1].Name)
	}
}

func TestBuildSources_ExtensionsFirst(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "priority", Path: "extensions/priority", Source: "git@...", Branch: "main"},
		},
		Resolution: ResolutionConfig{
			Order: []string{"extensions", "catalog", "local"},
		},
	}

	sources := BuildSources(cfg, "/repo")

	if len(sources) != 3 {
		t.Fatalf("expected 3 sources, got %d", len(sources))
	}

	if sources[0].Name != "priority" {
		t.Errorf("sources[0].Name = %q, want 'priority'", sources[0].Name)
	}
	if sources[1].Name != "catalog" {
		t.Errorf("sources[1].Name = %q, want 'catalog'", sources[1].Name)
	}
	if sources[2].Name != "local" {
		t.Errorf("sources[2].Name = %q, want 'local'", sources[2].Name)
	}
}

func TestBuildSources_AbsolutePath(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "abs-ext", Path: "/absolute/path/to/ext", Source: "git@...", Branch: "main"},
		},
		Resolution: ResolutionConfig{
			Order: []string{"extensions"},
		},
	}

	sources := BuildSources(cfg, "/repo")

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	// Absolute paths should be used as-is.
	if sources[0].BasePath != "/absolute/path/to/ext" {
		t.Errorf("sources[0].BasePath = %q, want '/absolute/path/to/ext'", sources[0].BasePath)
	}
}

func TestBuildSources_EmptyPath(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "no-path", Source: "git@...", Branch: "main"},
		},
		Resolution: ResolutionConfig{
			Order: []string{"extensions"},
		},
	}

	sources := BuildSources(cfg, "/repo")

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	// When Path is empty, it should default to extensions/<name>.
	expected := filepath.Join("/repo", "extensions", "no-path")
	if sources[0].BasePath != expected {
		t.Errorf("sources[0].BasePath = %q, want %q", sources[0].BasePath, expected)
	}
}

func TestBuildSources_UnknownEntry(t *testing.T) {
	cfg := &ExtensionConfig{
		Resolution: ResolutionConfig{
			Order: []string{"custom-source"},
		},
	}

	sources := BuildSources(cfg, "/repo")

	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}

	if sources[0].Name != "custom-source" {
		t.Errorf("sources[0].Name = %q, want 'custom-source'", sources[0].Name)
	}

	expected := filepath.Join("/repo", "custom-source")
	if sources[0].BasePath != expected {
		t.Errorf("sources[0].BasePath = %q, want %q", sources[0].BasePath, expected)
	}
}
