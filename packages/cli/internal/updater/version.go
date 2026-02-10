package updater

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// CompareVersions compares two version strings using semver.
// Returns -1 if current < latest, 0 if equal, 1 if current > latest.
// Handles "v" prefix tolerance (strips leading "v" before parsing).
func CompareVersions(current, latest string) (int, error) {
	cv, err := parseSemver(current)
	if err != nil {
		return 0, fmt.Errorf("parsing current version %q: %w", current, err)
	}
	lv, err := parseSemver(latest)
	if err != nil {
		return 0, fmt.Errorf("parsing latest version %q: %w", latest, err)
	}
	return cv.Compare(lv), nil
}

// IsUpdateAvailable returns true if latest is newer than current.
func IsUpdateAvailable(current, latest string) (bool, error) {
	cmp, err := CompareVersions(current, latest)
	if err != nil {
		return false, err
	}
	return cmp == -1, nil
}

// parseSemver strips a leading "v" and parses the version string.
func parseSemver(version string) (*semver.Version, error) {
	version = strings.TrimPrefix(version, "v")
	return semver.NewVersion(version)
}
