package updater

import (
	"fmt"
	"runtime"
	"strings"
)

// ArchiveName returns the expected archive filename for the current platform.
// Matches GoReleaser template: agentx_{os}_{arch}.tar.gz (or .zip for Windows).
func ArchiveName() string {
	ext := ".tar.gz"
	if runtime.GOOS == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("agentx_%s_%s%s", runtime.GOOS, runtime.GOARCH, ext)
}

// SelectAssetForPlatform finds the asset matching the current OS/arch.
func SelectAssetForPlatform(assets []Asset) (*Asset, error) {
	expected := ArchiveName()
	for i := range assets {
		if assets[i].Name == expected {
			return &assets[i], nil
		}
	}

	// Try a more flexible match: look for the os_arch pattern anywhere in the name.
	pattern := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	for i := range assets {
		if strings.Contains(assets[i].Name, pattern) && isArchive(assets[i].Name) {
			return &assets[i], nil
		}
	}

	return nil, fmt.Errorf("no asset found for %s/%s (expected %s)", runtime.GOOS, runtime.GOARCH, expected)
}

// IsWindows returns true if the current OS is Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

func isArchive(name string) bool {
	return strings.HasSuffix(name, ".tar.gz") || strings.HasSuffix(name, ".zip")
}
