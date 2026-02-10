package updater

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadCache_Missing(t *testing.T) {
	tmp := t.TempDir()
	cache, err := LoadCache(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cache != nil {
		t.Error("expected nil cache for missing file")
	}
}

func TestSaveAndLoadCache(t *testing.T) {
	tmp := t.TempDir()

	now := time.Now().Truncate(time.Second)
	original := &VersionCache{
		LatestVersion:  "1.2.0",
		CurrentVersion: "1.1.0",
		CheckedAt:      now,
		UpdateAvailable: true,
	}

	if err := SaveCache(tmp, original); err != nil {
		t.Fatalf("SaveCache failed: %v", err)
	}

	loaded, err := LoadCache(tmp)
	if err != nil {
		t.Fatalf("LoadCache failed: %v", err)
	}

	if loaded.LatestVersion != "1.2.0" {
		t.Errorf("LatestVersion = %q, want %q", loaded.LatestVersion, "1.2.0")
	}
	if loaded.CurrentVersion != "1.1.0" {
		t.Errorf("CurrentVersion = %q, want %q", loaded.CurrentVersion, "1.1.0")
	}
	if !loaded.UpdateAvailable {
		t.Error("UpdateAvailable should be true")
	}
}

func TestLoadCache_Corrupted(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, cacheFileName)
	os.WriteFile(path, []byte("not valid json{{{"), 0644)

	_, err := LoadCache(tmp)
	if err == nil {
		t.Error("expected error for corrupted cache")
	}
}

func TestIsCacheStale(t *testing.T) {
	tests := []struct {
		name     string
		cache    *VersionCache
		maxAge   time.Duration
		expected bool
	}{
		{
			"nil cache is stale",
			nil,
			24 * time.Hour,
			true,
		},
		{
			"fresh cache",
			&VersionCache{CheckedAt: time.Now()},
			24 * time.Hour,
			false,
		},
		{
			"stale cache",
			&VersionCache{CheckedAt: time.Now().Add(-25 * time.Hour)},
			24 * time.Hour,
			true,
		},
		{
			"exactly at boundary",
			&VersionCache{CheckedAt: time.Now().Add(-24*time.Hour - time.Second)},
			24 * time.Hour,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCacheStale(tt.cache, tt.maxAge)
			if result != tt.expected {
				t.Errorf("IsCacheStale = %v, want %v", result, tt.expected)
			}
		})
	}
}
