package updater

import (
	"fmt"
	"io"
	"time"
)

// CheckAndPrintBanner checks the version cache and prints an update banner if
// a newer version is available. It never blocks — if the cache is stale, a
// background goroutine refreshes it for the next invocation.
func (u *Updater) CheckAndPrintBanner(w io.Writer, configDir string) {
	cache, err := LoadCache(configDir)
	if err != nil {
		// Silently ignore cache errors.
		return
	}

	// Print banner from existing cache if update is available.
	if cache != nil && cache.UpdateAvailable {
		PrintUpdateBanner(w, cache.CurrentVersion, cache.LatestVersion)
	}

	// Refresh cache in background if stale.
	if IsCacheStale(cache, DefaultCacheMaxAge) {
		go u.refreshCache(configDir)
	}
}

// PrintUpdateBanner prints the update notification to w.
func PrintUpdateBanner(w io.Writer, current, latest string) {
	fmt.Fprintf(w, "\nUpdate available: %s -> %s\n", current, latest)
	fmt.Fprintf(w, "    Run `agentx update` to upgrade\n\n")
}

// refreshCache fetches the latest version and updates the cache file.
// This runs in a background goroutine and never fails loudly.
func (u *Updater) refreshCache(configDir string) {
	release, err := u.CheckLatestVersion()
	if err != nil {
		return
	}

	available, err := IsUpdateAvailable(u.currentVersion, release.Version)
	if err != nil {
		return
	}

	cache := &VersionCache{
		LatestVersion:   release.Version,
		CurrentVersion:  u.currentVersion,
		CheckedAt:       time.Now(),
		UpdateAvailable: available,
	}

	// Silently ignore save errors.
	_ = SaveCache(configDir, cache)
}
