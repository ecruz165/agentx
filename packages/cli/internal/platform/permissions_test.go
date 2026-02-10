package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestChmod(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "test.txt")
	if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Chmod(path, 0600); err != nil {
		t.Fatalf("Chmod failed: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0600 {
			t.Errorf("permissions = %o, want %o", perm, 0600)
		}
	}
}

func TestChmodDir(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "secure")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}

	if err := Chmod(dir, 0700); err != nil {
		t.Fatalf("Chmod on dir failed: %v", err)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(dir)
		if err != nil {
			t.Fatal(err)
		}
		if perm := info.Mode().Perm(); perm != 0700 {
			t.Errorf("permissions = %o, want %o", perm, 0700)
		}
	}
}
