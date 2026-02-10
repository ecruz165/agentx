package platform

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
)

// CreateSymlink creates a symbolic link from link pointing to target.
// On Unix systems, this uses os.Symlink directly.
// On Windows, it attempts os.Symlink first (requires developer mode),
// then falls back to copying the file and writing a .target sidecar.
func CreateSymlink(target, link string) error {
	if runtime.GOOS != "windows" {
		return os.Symlink(target, link)
	}

	// Try native symlink first (works if developer mode is enabled).
	if err := os.Symlink(target, link); err == nil {
		return nil
	}

	// Fallback: copy the target file and record the target in a sidecar.
	if err := copyFileForSymlink(target, link); err != nil {
		return fmt.Errorf("symlink fallback (copy) failed: %w", err)
	}

	// Write a sidecar file so ReadSymlinkTarget can recover the original target.
	sidecar := link + ".target"
	if err := os.WriteFile(sidecar, []byte(target), 0644); err != nil {
		// Non-fatal: the copy succeeded, just log the sidecar failure.
		return nil
	}

	return nil
}

// RemoveSymlink removes a symlink (or its fallback copy and sidecar).
func RemoveSymlink(path string) error {
	err := os.Remove(path)

	// Also clean up the sidecar if it exists.
	sidecar := path + ".target"
	os.Remove(sidecar) // best-effort

	return err
}

// ReadSymlinkTarget returns the target of a symlink.
// On Windows, if os.Readlink fails (because a copy fallback was used),
// it reads from the .target sidecar file.
func ReadSymlinkTarget(path string) (string, error) {
	target, err := os.Readlink(path)
	if err == nil {
		return target, nil
	}

	if runtime.GOOS != "windows" {
		return "", err
	}

	// Windows fallback: read sidecar .target file.
	sidecar := path + ".target"
	data, readErr := os.ReadFile(sidecar)
	if readErr != nil {
		return "", fmt.Errorf("readlink failed and no .target sidecar found: %w", err)
	}
	return strings.TrimSpace(string(data)), nil
}

// IsSymlinkSupported returns true if the current platform supports native symlinks.
// On Windows this attempts a test symlink to check developer mode.
func IsSymlinkSupported() bool {
	if runtime.GOOS != "windows" {
		return true
	}

	// Try creating a temporary symlink to test support.
	tmpDir := os.TempDir()
	target := tmpDir
	link := tmpDir + "/.agentx-symlink-test"
	defer os.Remove(link)

	if err := os.Symlink(target, link); err != nil {
		return false
	}
	return true
}

// copyFileForSymlink copies src to dst. If src is a relative path (as used
// for profile symlinks), it resolves relative to the directory containing dst.
func copyFileForSymlink(src, dst string) error {
	// For relative targets, resolve against the link's parent directory.
	resolvedSrc := src
	if !isAbsPath(src) {
		dir := parentDir(dst)
		resolvedSrc = dir + string(os.PathSeparator) + src
	}

	in, err := os.Open(resolvedSrc)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// isAbsPath checks if a path is absolute (handles both Unix and Windows).
func isAbsPath(path string) bool {
	if len(path) == 0 {
		return false
	}
	// Unix absolute
	if path[0] == '/' {
		return true
	}
	// Windows absolute (e.g., C:\)
	if len(path) >= 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		return true
	}
	return false
}

// parentDir returns the parent directory of a path.
func parentDir(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			return path[:i]
		}
	}
	return "."
}
