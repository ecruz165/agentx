package extension

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ProjectConfigFile)

	content := `extensions:
  - name: acme-corp
    path: extensions/acme-corp
    source: https://github.com/acme/types.git
    branch: main
  - name: internal-types
    path: extensions/internal-types
    source: https://github.com/internal/types.git
    branch: develop
resolution:
  order: [local, catalog, extensions]
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Extensions) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(cfg.Extensions))
	}

	if cfg.Extensions[0].Name != "acme-corp" {
		t.Errorf("expected first extension name 'acme-corp', got %q", cfg.Extensions[0].Name)
	}

	if cfg.Extensions[1].Branch != "develop" {
		t.Errorf("expected second extension branch 'develop', got %q", cfg.Extensions[1].Branch)
	}

	if len(cfg.Resolution.Order) != 3 {
		t.Errorf("expected 3 resolution order entries, got %d", len(cfg.Resolution.Order))
	}

	expectedOrder := []string{"local", "catalog", "extensions"}
	for i, entry := range cfg.Resolution.Order {
		if entry != expectedOrder[i] {
			t.Errorf("resolution order[%d] = %q, want %q", i, entry, expectedOrder[i])
		}
	}
}

func TestLoadConfig_EmptyExtensions(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ProjectConfigFile)

	content := `extensions: []
resolution:
  order: [local, catalog, extensions]
`
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(cfg.Extensions) != 0 {
		t.Errorf("expected 0 extensions, got %d", len(cfg.Extensions))
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/" + ProjectConfigFile)
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ProjectConfigFile)

	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{
				Name:   "test-ext",
				Path:   "extensions/test-ext",
				Source: "https://github.com/test/ext.git",
				Branch: "main",
			},
		},
		Resolution: ResolutionConfig{
			Order: []string{"local", "catalog", "extensions"},
		},
	}

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	// Read it back.
	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() after save error = %v", err)
	}

	if len(loaded.Extensions) != 1 {
		t.Errorf("expected 1 extension, got %d", len(loaded.Extensions))
	}

	if loaded.Extensions[0].Name != "test-ext" {
		t.Errorf("expected extension name 'test-ext', got %q", loaded.Extensions[0].Name)
	}
}

func TestSaveConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, ProjectConfigFile)

	original := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "alpha", Path: "extensions/alpha", Source: "https://github.com/a/alpha.git", Branch: "main"},
			{Name: "beta", Path: "extensions/beta", Source: "https://github.com/b/beta.git", Branch: "develop"},
		},
		Resolution: ResolutionConfig{
			Order: []string{"local", "extensions", "catalog"},
		},
	}

	if err := SaveConfig(configPath, original); err != nil {
		t.Fatalf("SaveConfig() error = %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if len(loaded.Extensions) != len(original.Extensions) {
		t.Fatalf("extension count mismatch: got %d, want %d", len(loaded.Extensions), len(original.Extensions))
	}

	for i, ext := range loaded.Extensions {
		if ext.Name != original.Extensions[i].Name {
			t.Errorf("extension[%d].Name = %q, want %q", i, ext.Name, original.Extensions[i].Name)
		}
		if ext.Branch != original.Extensions[i].Branch {
			t.Errorf("extension[%d].Branch = %q, want %q", i, ext.Branch, original.Extensions[i].Branch)
		}
	}

	if len(loaded.Resolution.Order) != len(original.Resolution.Order) {
		t.Fatalf("resolution order length mismatch: got %d, want %d",
			len(loaded.Resolution.Order), len(original.Resolution.Order))
	}

	for i, entry := range loaded.Resolution.Order {
		if entry != original.Resolution.Order[i] {
			t.Errorf("resolution order[%d] = %q, want %q", i, entry, original.Resolution.Order[i])
		}
	}
}

func TestFindExtension(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "alpha", Path: "extensions/alpha"},
			{Name: "beta", Path: "extensions/beta"},
		},
	}

	ext := cfg.FindExtension("alpha")
	if ext == nil {
		t.Fatal("expected to find 'alpha', got nil")
	}
	if ext.Name != "alpha" {
		t.Errorf("expected name 'alpha', got %q", ext.Name)
	}

	ext = cfg.FindExtension("nonexistent")
	if ext != nil {
		t.Errorf("expected nil for nonexistent extension, got %v", ext)
	}
}

func TestAddExtension(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "existing", Path: "extensions/existing"},
		},
	}

	// Adding a new extension should succeed.
	err := cfg.AddExtension(Extension{Name: "new-ext", Path: "extensions/new-ext"})
	if err != nil {
		t.Fatalf("AddExtension() unexpected error = %v", err)
	}
	if len(cfg.Extensions) != 2 {
		t.Errorf("expected 2 extensions, got %d", len(cfg.Extensions))
	}

	// Adding a duplicate should fail.
	err = cfg.AddExtension(Extension{Name: "existing", Path: "extensions/existing"})
	if err == nil {
		t.Fatal("expected error for duplicate extension, got nil")
	}
}

func TestRemoveExtension(t *testing.T) {
	cfg := &ExtensionConfig{
		Extensions: []Extension{
			{Name: "alpha", Path: "extensions/alpha"},
			{Name: "beta", Path: "extensions/beta"},
			{Name: "gamma", Path: "extensions/gamma"},
		},
	}

	// Remove the middle one.
	if err := cfg.RemoveExtension("beta"); err != nil {
		t.Fatalf("RemoveExtension() error = %v", err)
	}
	if len(cfg.Extensions) != 2 {
		t.Errorf("expected 2 extensions after removal, got %d", len(cfg.Extensions))
	}
	if cfg.FindExtension("beta") != nil {
		t.Error("expected 'beta' to be removed")
	}

	// Removing a nonexistent extension should fail.
	if err := cfg.RemoveExtension("nonexistent"); err == nil {
		t.Fatal("expected error for nonexistent extension, got nil")
	}
}
