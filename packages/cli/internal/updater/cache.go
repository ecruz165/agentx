package updater

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	cacheFileName = "version-check.json"
	// DefaultCacheMaxAge is the default maximum age for the version cache.
	DefaultCacheMaxAge = 24 * time.Hour
)

// VersionCache holds cached version check results.
type VersionCache struct {
	LatestVersion  string    `json:"latest_version"`
	CurrentVersion string    `json:"current_version"`
	CheckedAt      time.Time `json:"checked_at"`
	UpdateAvailable bool     `json:"update_available"`
}

// LoadCache reads the version cache from the config directory.
// Returns nil, nil if the cache file does not exist (first run).
func LoadCache(configDir string) (*VersionCache, error) {
	path := filepath.Join(configDir, cacheFileName)

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading version cache: %w", err)
	}

	var cache VersionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("parsing version cache: %w", err)
	}
	return &cache, nil
}

// SaveCache writes the version cache to the config directory.
func SaveCache(configDir string, cache *VersionCache) error {
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling version cache: %w", err)
	}

	path := filepath.Join(configDir, cacheFileName)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing version cache: %w", err)
	}
	return nil
}

// IsCacheStale returns true if the cache is older than maxAge or nil.
func IsCacheStale(cache *VersionCache, maxAge time.Duration) bool {
	if cache == nil {
		return true
	}
	return time.Since(cache.CheckedAt) > maxAge
}
