package registry

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// excludedNames are files/directories excluded during type installation.
var excludedNames = map[string]bool{
	"node_modules": true,
	".git":         true,
	".DS_Store":    true,
}

// InstallType copies a resolved type's directory to the installed root.
// It excludes node_modules/, .git/, and .DS_Store.
func InstallType(resolved *ResolvedType, installedRoot string) error {
	dst := filepath.Join(installedRoot, resolved.TypePath)

	// Remove existing installation to ensure clean copy.
	if _, err := os.Stat(dst); err == nil {
		if err := os.RemoveAll(dst); err != nil {
			return fmt.Errorf("removing existing installation at %s: %w", dst, err)
		}
	}

	if err := copyDir(resolved.SourceDir, dst); err != nil {
		return fmt.Errorf("copying %s to %s: %w", resolved.SourceDir, dst, err)
	}

	return nil
}

// InstallNodeDeps runs npm install in the type directory if a package.json exists.
// It checks for Node.js availability first. If Node is not available, it returns
// a warning message instead of an error.
func InstallNodeDeps(typeDir string) (string, error) {
	pkgJSON := filepath.Join(typeDir, "package.json")
	if _, err := os.Stat(pkgJSON); err != nil {
		return "", nil // no package.json, nothing to do
	}

	// Check Node.js availability.
	if _, err := exec.LookPath("node"); err != nil {
		return "Node.js not found — skipping npm install (run `agentx doctor --check-runtime`)", nil
	}

	npmPath, err := exec.LookPath("npm")
	if err != nil {
		return "npm not found — skipping dependency installation", nil
	}

	cmd := exec.Command(npmPath, "install", "--prefer-offline")
	cmd.Dir = typeDir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("npm install in %s: %w", typeDir, err)
	}

	return "", nil
}

// RemoveType removes an installed type directory.
func RemoveType(typePath string, installedRoot string) error {
	dir := filepath.Join(installedRoot, typePath)

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("type %s is not installed", typePath)
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", dir)
	}

	if err := os.RemoveAll(dir); err != nil {
		return fmt.Errorf("removing %s: %w", dir, err)
	}

	return nil
}

// copyDir recursively copies src to dst, excluding entries in excludedNames.
func copyDir(src, dst string) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if shouldExclude(entry.Name()) {
			continue
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else if entry.Type().IsRegular() {
			if err := copyFile(srcPath, dstPath); err != nil {
				return err
			}
		}
		// Skip symlinks and other special files during copy.
	}

	return nil
}

// copyFile copies a single file from src to dst, preserving permissions.
func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, data, srcInfo.Mode())
}

// shouldExclude returns true if the name should be excluded during copy.
func shouldExclude(name string) bool {
	return excludedNames[name]
}
