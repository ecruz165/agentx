package registry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallTypeCopiesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	installedRoot := filepath.Join(tmpDir, "installed")

	catalogDir := filepath.Join(testdataDir(), "catalog")
	resolved := &ResolvedType{
		TypePath:     "personas/test-persona",
		ManifestPath: filepath.Join(catalogDir, "personas", "test-persona", "manifest.yaml"),
		SourceDir:    filepath.Join(catalogDir, "personas", "test-persona"),
		SourceName:   "catalog",
		Category:     "persona",
	}

	if err := InstallType(resolved, installedRoot); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Verify manifest was copied.
	copiedManifest := filepath.Join(installedRoot, "personas", "test-persona", "manifest.yaml")
	if _, err := os.Stat(copiedManifest); err != nil {
		t.Errorf("manifest not copied: %v", err)
	}
}

func TestInstallTypeExcludesNodeModules(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a source with node_modules and .git.
	srcDir := filepath.Join(tmpDir, "src", "skills", "test-skill")
	if err := os.MkdirAll(filepath.Join(srcDir, "node_modules", "dep"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(srcDir, ".git", "objects"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "manifest.yaml"), []byte("name: test\ntype: skill\nversion: \"1.0.0\"\ndescription: test\nruntime: node\ntopic: t\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "index.mjs"), []byte("export default {}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, ".DS_Store"), []byte(""), 0644); err != nil {
		t.Fatal(err)
	}

	installedRoot := filepath.Join(tmpDir, "installed")
	resolved := &ResolvedType{
		TypePath:  "skills/test-skill",
		SourceDir: srcDir,
		Category:  "skill",
	}

	if err := InstallType(resolved, installedRoot); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// manifest.yaml and index.mjs should be copied.
	if _, err := os.Stat(filepath.Join(installedRoot, "skills", "test-skill", "manifest.yaml")); err != nil {
		t.Error("manifest.yaml should be copied")
	}
	if _, err := os.Stat(filepath.Join(installedRoot, "skills", "test-skill", "index.mjs")); err != nil {
		t.Error("index.mjs should be copied")
	}

	// node_modules, .git, .DS_Store should NOT be copied.
	if _, err := os.Stat(filepath.Join(installedRoot, "skills", "test-skill", "node_modules")); err == nil {
		t.Error("node_modules should not be copied")
	}
	if _, err := os.Stat(filepath.Join(installedRoot, "skills", "test-skill", ".git")); err == nil {
		t.Error(".git should not be copied")
	}
	if _, err := os.Stat(filepath.Join(installedRoot, "skills", "test-skill", ".DS_Store")); err == nil {
		t.Error(".DS_Store should not be copied")
	}
}

func TestInstallTypeIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	installedRoot := filepath.Join(tmpDir, "installed")

	catalogDir := filepath.Join(testdataDir(), "catalog")
	resolved := &ResolvedType{
		TypePath:     "personas/test-persona",
		ManifestPath: filepath.Join(catalogDir, "personas", "test-persona", "manifest.yaml"),
		SourceDir:    filepath.Join(catalogDir, "personas", "test-persona"),
		SourceName:   "catalog",
		Category:     "persona",
	}

	// Install twice — should not error.
	if err := InstallType(resolved, installedRoot); err != nil {
		t.Fatalf("first install: %v", err)
	}
	if err := InstallType(resolved, installedRoot); err != nil {
		t.Fatalf("second install: %v", err)
	}
}

func TestRemoveType(t *testing.T) {
	tmpDir := t.TempDir()
	installedRoot := filepath.Join(tmpDir, "installed")

	// Create installed type.
	typeDir := filepath.Join(installedRoot, "personas", "test-persona")
	if err := os.MkdirAll(typeDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(typeDir, "manifest.yaml"), []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveType("personas/test-persona", installedRoot); err != nil {
		t.Fatalf("RemoveType: %v", err)
	}

	if _, err := os.Stat(typeDir); err == nil {
		t.Error("type directory should be removed")
	}
}

func TestRemoveTypeNotInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	if err := RemoveType("personas/nonexistent", tmpDir); err == nil {
		t.Error("expected error for nonexistent type")
	}
}

func TestShouldExclude(t *testing.T) {
	tests := []struct {
		name     string
		expected bool
	}{
		{"node_modules", true},
		{".git", true},
		{".DS_Store", true},
		{"manifest.yaml", false},
		{"index.mjs", false},
		{"src", false},
	}

	for _, tt := range tests {
		if got := shouldExclude(tt.name); got != tt.expected {
			t.Errorf("shouldExclude(%q) = %v, want %v", tt.name, got, tt.expected)
		}
	}
}
