package updater

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestReplaceBinary_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		// Test the Windows guard returns error on non-Windows by simulating.
		// We can only verify the actual error on Windows.
		t.Skip("Windows-specific test")
	}

	err := ReplaceBinary("/tmp/new", "/tmp/current", "1.0.0")
	if err == nil {
		t.Error("expected error on Windows")
	}
}

func TestRollbackBinary(t *testing.T) {
	tmp := t.TempDir()

	backupPath := filepath.Join(tmp, "agentx.backup")
	currentPath := filepath.Join(tmp, "agentx")

	// Create a backup file.
	os.WriteFile(backupPath, []byte("original binary"), 0755)

	err := RollbackBinary(backupPath, currentPath)
	if err != nil {
		t.Fatalf("RollbackBinary failed: %v", err)
	}

	// Verify the original is restored.
	data, err := os.ReadFile(currentPath)
	if err != nil {
		t.Fatalf("reading restored binary: %v", err)
	}
	if string(data) != "original binary" {
		t.Errorf("restored content mismatch: %s", data)
	}

	// Backup should be removed.
	if _, err := os.Stat(backupPath); !os.IsNotExist(err) {
		t.Error("backup file was not cleaned up")
	}
}

func TestCopyFile(t *testing.T) {
	tmp := t.TempDir()

	src := filepath.Join(tmp, "src")
	dst := filepath.Join(tmp, "dst")

	os.WriteFile(src, []byte("copy test"), 0644)

	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}

	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading dst: %v", err)
	}
	if string(data) != "copy test" {
		t.Errorf("content mismatch: %s", data)
	}
}
