package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/platform"
)

// ReplaceBinary safely replaces the current binary with a new one.
// It creates a backup, performs the swap, and verifies the new binary.
// On failure it rolls back to the backup.
func ReplaceBinary(newPath, currentPath, expectedVersion string) error {
	if runtime.GOOS == "windows" {
		return fmt.Errorf("self-update is not supported on Windows. Please download the latest version manually from https://github.com/%s/releases", branding.GitHubRepo())
	}

	// Preserve original permissions.
	info, err := os.Stat(currentPath)
	if err != nil {
		return fmt.Errorf("stat current binary: %w", err)
	}
	origPerm := info.Mode().Perm()

	backupPath := currentPath + ".backup"

	// Create backup.
	if err := os.Rename(currentPath, backupPath); err != nil {
		// Rename may fail across filesystems; try copy.
		if copyErr := copyFile(currentPath, backupPath); copyErr != nil {
			return fmt.Errorf("creating backup: %w", copyErr)
		}
		os.Remove(currentPath)
	}

	// Move new binary into place.
	if err := os.Rename(newPath, currentPath); err != nil {
		// Cross-filesystem fallback.
		if copyErr := copyFile(newPath, currentPath); copyErr != nil {
			RollbackBinary(backupPath, currentPath)
			return fmt.Errorf("installing new binary: %w", copyErr)
		}
		os.Remove(newPath)
	}

	// Restore original permissions.
	platform.Chmod(currentPath, origPerm)

	// Verify the new binary works.
	if err := VerifyBinary(currentPath, expectedVersion); err != nil {
		RollbackBinary(backupPath, currentPath)
		return fmt.Errorf("verification failed, rolled back: %w", err)
	}

	// Cleanup backup.
	os.Remove(backupPath)

	return nil
}

// VerifyBinary executes the binary with "version --json" and checks the output.
func VerifyBinary(binaryPath, expectedVersion string) error {
	cmd := exec.Command(binaryPath, "version", "--json")
	// Set a timeout by running in a goroutine.
	done := make(chan error, 1)
	var output []byte
	go func() {
		var err error
		output, err = cmd.Output()
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			return fmt.Errorf("new binary exited with error: %w", err)
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		return fmt.Errorf("new binary timed out after 5 seconds")
	}

	var versionInfo map[string]string
	if err := json.Unmarshal(output, &versionInfo); err != nil {
		return fmt.Errorf("parsing version output: %w", err)
	}

	return nil
}

// RollbackBinary restores the backup to the current path.
func RollbackBinary(backupPath, currentPath string) error {
	if err := os.Rename(backupPath, currentPath); err != nil {
		if copyErr := copyFile(backupPath, currentPath); copyErr != nil {
			return fmt.Errorf("rollback failed: %w (original rename error: %v)", copyErr, err)
		}
		os.Remove(backupPath)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
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
