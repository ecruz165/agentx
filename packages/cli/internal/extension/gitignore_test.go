package extension

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddToGitignore(t *testing.T) {
	dir := t.TempDir()

	// Create a baseline .gitignore.
	gitignorePath := filepath.Join(dir, ".gitignore")
	initial := "node_modules/\n.env\nextensions/*\n!extensions/.gitkeep\n"
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	// Add an extension.
	if err := AddToGitignore(dir, "acme-corp"); err != nil {
		t.Fatalf("AddToGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "!extensions/acme-corp") {
		t.Errorf("expected .gitignore to contain '!extensions/acme-corp', got:\n%s", string(content))
	}
}

func TestAddToGitignore_Idempotent(t *testing.T) {
	dir := t.TempDir()

	gitignorePath := filepath.Join(dir, ".gitignore")
	initial := "extensions/*\n!extensions/acme-corp\n"
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	// Adding the same line again should be a no-op.
	if err := AddToGitignore(dir, "acme-corp"); err != nil {
		t.Fatalf("AddToGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	count := strings.Count(string(content), "!extensions/acme-corp")
	if count != 1 {
		t.Errorf("expected exactly 1 occurrence, found %d in:\n%s", count, string(content))
	}
}

func TestAddToGitignore_CreatesFileIfMissing(t *testing.T) {
	dir := t.TempDir()

	if err := AddToGitignore(dir, "new-ext"); err != nil {
		t.Fatalf("AddToGitignore() error = %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, ".gitignore"))
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(string(content), "!extensions/new-ext") {
		t.Errorf("expected .gitignore to contain '!extensions/new-ext', got:\n%s", string(content))
	}
}

func TestAddToGitignore_NoTrailingNewline(t *testing.T) {
	dir := t.TempDir()

	gitignorePath := filepath.Join(dir, ".gitignore")
	initial := "node_modules/" // no trailing newline
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := AddToGitignore(dir, "test-ext"); err != nil {
		t.Fatalf("AddToGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(string(content), "\n")
	// Should have at least the original line and the new line.
	found := false
	for _, line := range lines {
		if strings.TrimSpace(line) == "!extensions/test-ext" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected '!extensions/test-ext' in .gitignore, got:\n%s", string(content))
	}
}

func TestRemoveFromGitignore(t *testing.T) {
	dir := t.TempDir()

	gitignorePath := filepath.Join(dir, ".gitignore")
	initial := "node_modules/\nextensions/*\n!extensions/.gitkeep\n!extensions/acme-corp\n!extensions/other\n"
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := RemoveFromGitignore(dir, "acme-corp"); err != nil {
		t.Fatalf("RemoveFromGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(content), "!extensions/acme-corp") {
		t.Errorf("expected '!extensions/acme-corp' to be removed, got:\n%s", string(content))
	}

	// Other extension should still be there.
	if !strings.Contains(string(content), "!extensions/other") {
		t.Errorf("expected '!extensions/other' to remain, got:\n%s", string(content))
	}
}

func TestRemoveFromGitignore_NotPresent(t *testing.T) {
	dir := t.TempDir()

	gitignorePath := filepath.Join(dir, ".gitignore")
	initial := "node_modules/\n"
	if err := os.WriteFile(gitignorePath, []byte(initial), 0o644); err != nil {
		t.Fatal(err)
	}

	// Removing a non-existent line should be a no-op.
	if err := RemoveFromGitignore(dir, "nonexistent"); err != nil {
		t.Fatalf("RemoveFromGitignore() error = %v", err)
	}

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		t.Fatal(err)
	}

	if string(content) != initial {
		t.Errorf("file should be unchanged, got:\n%s", string(content))
	}
}

func TestRemoveFromGitignore_NoFile(t *testing.T) {
	dir := t.TempDir()

	// Should not error when .gitignore doesn't exist.
	if err := RemoveFromGitignore(dir, "nonexistent"); err != nil {
		t.Fatalf("RemoveFromGitignore() error = %v", err)
	}
}
