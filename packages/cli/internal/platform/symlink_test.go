package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCreateSymlink(t *testing.T) {
	tmp := t.TempDir()

	// Create a target file.
	targetPath := filepath.Join(tmp, "target.txt")
	if err := os.WriteFile(targetPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(tmp, "link.txt")
	if err := CreateSymlink(targetPath, linkPath); err != nil {
		t.Fatalf("CreateSymlink failed: %v", err)
	}

	// Verify the link exists and has the right content.
	data, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("reading link: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("link content = %q, want %q", string(data), "hello")
	}
}

func TestCreateSymlinkRelative(t *testing.T) {
	tmp := t.TempDir()

	// Create a target file.
	targetPath := filepath.Join(tmp, "profile.yaml")
	if err := os.WriteFile(targetPath, []byte("name: default"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create a relative symlink (like profile "active" -> "default.yaml").
	linkPath := filepath.Join(tmp, "active")
	if err := CreateSymlink("profile.yaml", linkPath); err != nil {
		t.Fatalf("CreateSymlink (relative) failed: %v", err)
	}

	// On Unix, verify it's actually a symlink.
	if runtime.GOOS != "windows" {
		target, err := os.Readlink(linkPath)
		if err != nil {
			t.Fatalf("Readlink failed: %v", err)
		}
		if target != "profile.yaml" {
			t.Errorf("symlink target = %q, want %q", target, "profile.yaml")
		}
	}
}

func TestRemoveSymlink(t *testing.T) {
	tmp := t.TempDir()

	targetPath := filepath.Join(tmp, "target.txt")
	if err := os.WriteFile(targetPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(tmp, "link.txt")
	if err := CreateSymlink(targetPath, linkPath); err != nil {
		t.Fatal(err)
	}

	if err := RemoveSymlink(linkPath); err != nil {
		t.Fatalf("RemoveSymlink failed: %v", err)
	}

	if _, err := os.Stat(linkPath); !os.IsNotExist(err) {
		t.Error("link still exists after RemoveSymlink")
	}
}

func TestReadSymlinkTarget(t *testing.T) {
	tmp := t.TempDir()

	targetPath := filepath.Join(tmp, "target.txt")
	if err := os.WriteFile(targetPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	linkPath := filepath.Join(tmp, "link.txt")
	if err := CreateSymlink(targetPath, linkPath); err != nil {
		t.Fatal(err)
	}

	got, err := ReadSymlinkTarget(linkPath)
	if err != nil {
		t.Fatalf("ReadSymlinkTarget failed: %v", err)
	}
	if got != targetPath {
		t.Errorf("ReadSymlinkTarget = %q, want %q", got, targetPath)
	}
}

func TestIsSymlinkSupported(t *testing.T) {
	result := IsSymlinkSupported()
	// On macOS and Linux, symlinks should always be supported.
	if runtime.GOOS != "windows" && !result {
		t.Error("IsSymlinkSupported returned false on Unix")
	}
}
