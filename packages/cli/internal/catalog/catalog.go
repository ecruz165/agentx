// Package catalog manages the AgentX catalog repository for end-user mode.
// It handles cloning, updating, and freshness tracking of the catalog.
package catalog

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/config"
	"github.com/agentx-labs/agentx/internal/userdata"
)

const (
	// freshnessFile is the name of the timestamp marker file.
	freshnessFile = ".catalog-updated"

	// DefaultMaxAge is the default staleness threshold (7 days).
	DefaultMaxAge = 7 * 24 * time.Hour

	// tmpSuffix is appended to the target dir during atomic clone.
	tmpSuffix = ".tmp"
)

// RepoURL returns the catalog repository URL, checking (in order):
// 1. <PREFIX>_CATALOG_REPO_URL env var
// 2. config key "catalog_repo"
// 3. branding.CatalogRepoURL() (from branding.yaml)
func RepoURL() string {
	if v := os.Getenv(branding.EnvVar("CATALOG_REPO_URL")); v != "" {
		return v
	}
	if v := config.Get("catalog_repo"); v != "" {
		return v
	}
	return branding.CatalogRepoURL()
}

// Clone performs a shallow clone of the catalog into targetDir.
// It attempts a sparse checkout (git >= 2.25.0) to only fetch the catalog/
// subdirectory. Falls back to a full shallow clone if sparse checkout is
// unavailable.
//
// The clone is atomic: it writes to a .tmp directory first, then renames
// on success. On failure the .tmp directory is cleaned up.
func Clone(targetDir string) error {
	if err := ensureGit(); err != nil {
		return err
	}

	repoURL := RepoURL()
	tmpDir := targetDir + tmpSuffix

	// Clean up any leftover tmp dir from a previous failed attempt.
	_ = os.RemoveAll(tmpDir)

	// Ensure parent directory exists.
	if err := os.MkdirAll(filepath.Dir(tmpDir), userdata.DirPermNormal); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	if err := trySparseClone(tmpDir, repoURL); err != nil {
		// Sparse clone failed — fall back to full shallow clone.
		_ = os.RemoveAll(tmpDir)
		if err := fullShallowClone(tmpDir, repoURL); err != nil {
			_ = os.RemoveAll(tmpDir)
			return fmt.Errorf("cloning catalog: %w", err)
		}
	}

	// Atomic rename.
	if err := os.RemoveAll(targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return fmt.Errorf("removing existing catalog dir: %w", err)
	}
	if err := os.Rename(tmpDir, targetDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		return fmt.Errorf("finalizing catalog clone: %w", err)
	}

	// Write freshness marker.
	WriteFreshnessMarker(targetDir)
	return nil
}

// Update pulls the latest changes in the catalog repo directory.
// If the catalog directory doesn't exist, it calls Clone instead.
func Update(catalogRepoDir string) error {
	if err := ensureGit(); err != nil {
		return err
	}

	// If the repo doesn't exist, clone it.
	gitDir := filepath.Join(catalogRepoDir, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return Clone(catalogRepoDir)
	}

	cmd := exec.Command("git", "pull", "--depth=1", "--rebase")
	cmd.Dir = catalogRepoDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pulling catalog updates: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	WriteFreshnessMarker(catalogRepoDir)
	return nil
}

// WriteFreshnessMarker writes the current Unix timestamp to the freshness file.
func WriteFreshnessMarker(catalogRepoDir string) {
	markerPath := filepath.Join(catalogRepoDir, freshnessFile)
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	_ = os.WriteFile(markerPath, []byte(ts), userdata.DirPermNormal)
}

// ReadFreshnessMarker reads the timestamp from the freshness file.
// Returns zero time if the file doesn't exist or can't be parsed.
func ReadFreshnessMarker(catalogRepoDir string) time.Time {
	markerPath := filepath.Join(catalogRepoDir, freshnessFile)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		return time.Time{}
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return time.Time{}
	}
	return time.Unix(ts, 0)
}

// IsStale returns true if the catalog was last updated more than maxAge ago.
// Returns true if the freshness marker doesn't exist.
func IsStale(catalogRepoDir string, maxAge time.Duration) bool {
	lastUpdated := ReadFreshnessMarker(catalogRepoDir)
	if lastUpdated.IsZero() {
		return true
	}
	return time.Since(lastUpdated) > maxAge
}

// trySparseClone attempts a sparse shallow clone that only checks out catalog/.
func trySparseClone(targetDir, repoURL string) error {
	// Step 1: Clone with --sparse --no-checkout --depth=1.
	cmd := exec.Command("git", "clone", "--depth=1", "--sparse", "--no-checkout", repoURL, targetDir)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sparse clone: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	// Step 2: Set sparse-checkout to only include catalog/.
	cmd = exec.Command("git", "sparse-checkout", "set", "catalog/")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("sparse-checkout set: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	// Step 3: Checkout.
	cmd = exec.Command("git", "checkout")
	cmd.Dir = targetDir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("checkout: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// fullShallowClone performs a regular --depth=1 clone (fallback for older git).
func fullShallowClone(targetDir, repoURL string) error {
	cmd := exec.Command("git", "clone", "--depth=1", repoURL, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shallow clone: %w\n%s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// ensureGit checks that git is available on PATH.
func ensureGit() error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is required but not found in PATH")
	}
	return nil
}
