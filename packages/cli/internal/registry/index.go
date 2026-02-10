package registry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

// CachedIndex holds a cached set of discovered types along with source
// modification times used for invalidation.
type CachedIndex struct {
	Types      []DiscoveredType  `json:"types"`
	SourceMods map[string]int64  `json:"source_mods"` // source name -> mtime unix
	CachedAt   time.Time         `json:"cached_at"`
}

// DefaultCachePath returns the default cache file path: ~/.agentx/registry-cache.json.
func DefaultCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".agentx", "registry-cache.json"), nil
}

// DiscoverAllCached returns discovered types, using a cache file when available
// and still valid. If the cache is stale or missing, it rebuilds from sources
// and writes a new cache file.
func DiscoverAllCached(sources []Source, cachePath string) ([]DiscoveredType, error) {
	cached, err := loadCache(cachePath)
	if err == nil && isCacheValid(cached, sources) {
		return cached.Types, nil
	}

	// Cache miss or invalid — rebuild.
	types, err := DiscoverAll(sources)
	if err != nil {
		return nil, err
	}

	// Write cache (best effort — search still works without caching).
	writeCache(cachePath, types, sources)

	return types, nil
}

// loadCache reads and parses the cache file.
func loadCache(path string) (*CachedIndex, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var idx CachedIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, err
	}
	return &idx, nil
}

// isCacheValid checks whether the cached source mtimes still match the
// current directory mtimes. Any change (or missing source) invalidates.
func isCacheValid(cached *CachedIndex, sources []Source) bool {
	if cached == nil || len(cached.SourceMods) == 0 {
		return false
	}
	// Source list must match exactly.
	if len(cached.SourceMods) != len(sources) {
		return false
	}
	for _, src := range sources {
		cachedMtime, ok := cached.SourceMods[src.Name]
		if !ok {
			return false
		}
		currentMtime := latestMtime(src.BasePath)
		if currentMtime != cachedMtime {
			return false
		}
	}
	return true
}

// latestMtime returns the latest modification time (unix seconds) across
// the source directory and its immediate category subdirectories. This is
// a lightweight check that catches new/removed types without a full walk.
func latestMtime(basePath string) int64 {
	var latest int64
	info, err := os.Stat(basePath)
	if err != nil {
		return 0
	}
	if t := info.ModTime().Unix(); t > latest {
		latest = t
	}

	// Check known category subdirectories.
	for _, cat := range knownCategories {
		catDir := filepath.Join(basePath, cat)
		if fi, err := os.Stat(catDir); err == nil {
			if t := fi.ModTime().Unix(); t > latest {
				latest = t
			}
			// Also walk one level deep for topic/vendor subdirs.
			entries, err := os.ReadDir(catDir)
			if err == nil {
				for _, entry := range entries {
					if entry.IsDir() {
						subDir := filepath.Join(catDir, entry.Name())
						if si, err := os.Stat(subDir); err == nil {
							if t := si.ModTime().Unix(); t > latest {
								latest = t
							}
						}
					}
				}
			}
		}
	}
	return latest
}

// writeCache serializes the discovered types and source mtimes to disk.
func writeCache(path string, types []DiscoveredType, sources []Source) {
	mods := make(map[string]int64, len(sources))
	for _, src := range sources {
		mods[src.Name] = latestMtime(src.BasePath)
	}

	idx := CachedIndex{
		Types:      types,
		SourceMods: mods,
		CachedAt:   time.Now(),
	}

	data, err := json.MarshalIndent(idx, "", "  ")
	if err != nil {
		return
	}

	// Ensure parent directory exists.
	dir := filepath.Dir(path)
	os.MkdirAll(dir, 0755)

	os.WriteFile(path, data, 0644)
}
