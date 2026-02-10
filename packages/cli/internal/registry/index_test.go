package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDiscoverAllCached_NoCacheFile(t *testing.T) {
	// Set up a minimal catalog source with a manifest.
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "skills", "test", "basic")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte("name: basic\ntype: skill\nversion: \"1.0.0\"\n"), 0644)

	sources := []Source{{Name: "catalog", BasePath: tmp}}
	cachePath := filepath.Join(t.TempDir(), "cache.json")

	types, err := DiscoverAllCached(sources, cachePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(types) == 0 {
		t.Fatal("expected at least one discovered type")
	}

	// Cache file should have been written.
	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("expected cache file to exist: %v", err)
	}
}

func TestDiscoverAllCached_UsesCacheOnSecondCall(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "skills", "test", "basic")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte("name: basic\ntype: skill\nversion: \"1.0.0\"\n"), 0644)

	sources := []Source{{Name: "catalog", BasePath: tmp}}
	cachePath := filepath.Join(t.TempDir(), "cache.json")

	// First call builds and caches.
	types1, err := DiscoverAllCached(sources, cachePath)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Second call should use cache (same results).
	types2, err := DiscoverAllCached(sources, cachePath)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if len(types1) != len(types2) {
		t.Errorf("expected same count: %d vs %d", len(types1), len(types2))
	}
}

func TestDiscoverAllCached_InvalidateOnSourceChange(t *testing.T) {
	tmp := t.TempDir()
	skillDir := filepath.Join(tmp, "skills", "test", "basic")
	os.MkdirAll(skillDir, 0755)
	os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte("name: basic\ntype: skill\nversion: \"1.0.0\"\n"), 0644)

	sources := []Source{{Name: "catalog", BasePath: tmp}}
	cachePath := filepath.Join(t.TempDir(), "cache.json")

	// Build initial cache.
	_, err := DiscoverAllCached(sources, cachePath)
	if err != nil {
		t.Fatalf("first call error: %v", err)
	}

	// Modify the source to invalidate cache.
	newSkillDir := filepath.Join(tmp, "skills", "test", "another")
	os.MkdirAll(newSkillDir, 0755)
	os.WriteFile(filepath.Join(newSkillDir, "manifest.yaml"), []byte("name: another\ntype: skill\nversion: \"1.0.0\"\n"), 0644)

	// Touch the skills dir to ensure mtime changes.
	now := time.Now().Add(2 * time.Second)
	os.Chtimes(filepath.Join(tmp, "skills", "test"), now, now)

	// Second call should rebuild (more types now).
	types, err := DiscoverAllCached(sources, cachePath)
	if err != nil {
		t.Fatalf("second call error: %v", err)
	}

	if len(types) < 2 {
		t.Errorf("expected at least 2 types after adding another, got %d", len(types))
	}
}

func TestIsCacheValid_EmptyCache(t *testing.T) {
	sources := []Source{{Name: "catalog", BasePath: t.TempDir()}}
	if isCacheValid(nil, sources) {
		t.Error("nil cache should be invalid")
	}
	if isCacheValid(&CachedIndex{}, sources) {
		t.Error("empty cache should be invalid")
	}
}

func TestIsCacheValid_SourceCountMismatch(t *testing.T) {
	cached := &CachedIndex{
		SourceMods: map[string]int64{"catalog": 100},
	}
	sources := []Source{
		{Name: "catalog", BasePath: t.TempDir()},
		{Name: "ext", BasePath: t.TempDir()},
	}
	if isCacheValid(cached, sources) {
		t.Error("mismatched source count should be invalid")
	}
}

func TestLoadCache_BadJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)
	_, err := loadCache(path)
	if err == nil {
		t.Error("expected error for bad JSON")
	}
}

func TestLoadCache_ValidJSON(t *testing.T) {
	idx := CachedIndex{
		Types: []DiscoveredType{
			{Name: "test", Category: "skill"},
		},
		SourceMods: map[string]int64{"catalog": 123},
		CachedAt:   time.Now(),
	}
	data, _ := json.Marshal(idx)
	path := filepath.Join(t.TempDir(), "cache.json")
	os.WriteFile(path, data, 0644)

	loaded, err := loadCache(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded.Types) != 1 {
		t.Errorf("expected 1 type, got %d", len(loaded.Types))
	}
	if loaded.Types[0].Name != "test" {
		t.Errorf("expected name 'test', got %q", loaded.Types[0].Name)
	}
}

func TestLatestMtime_NonexistentDir(t *testing.T) {
	mtime := latestMtime("/nonexistent/path")
	if mtime != 0 {
		t.Errorf("expected 0 for nonexistent path, got %d", mtime)
	}
}

func TestDefaultCachePath(t *testing.T) {
	path, err := DefaultCachePath()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
	if filepath.Base(path) != "registry-cache.json" {
		t.Errorf("expected filename 'registry-cache.json', got %q", filepath.Base(path))
	}
}
