package extension

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// gitignoreLine returns the gitignore exclusion line for an extension.
func gitignoreLine(name string) string {
	return "!extensions/" + name
}

// AddToGitignore appends an exclusion line for the named extension to .gitignore.
// The line !extensions/<name> tells git to track the extension despite the
// extensions/* ignore rule. If the line already exists, this is a no-op.
func AddToGitignore(repoRoot, name string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	line := gitignoreLine(name)

	content, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	// Check if line already exists.
	lines := strings.Split(string(content), "\n")
	for _, l := range lines {
		if strings.TrimSpace(l) == line {
			return nil // already present
		}
	}

	// Append the line. Ensure there's a newline before our addition.
	suffix := line + "\n"
	if len(content) > 0 && !strings.HasSuffix(string(content), "\n") {
		suffix = "\n" + suffix
	}

	f, err := os.OpenFile(gitignorePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("opening .gitignore for append: %w", err)
	}
	defer f.Close()

	if _, err := f.WriteString(suffix); err != nil {
		return fmt.Errorf("writing to .gitignore: %w", err)
	}

	return nil
}

// RemoveFromGitignore removes the exclusion line for the named extension
// from .gitignore. If the line is not present, this is a no-op.
func RemoveFromGitignore(repoRoot, name string) error {
	gitignorePath := filepath.Join(repoRoot, ".gitignore")
	line := gitignoreLine(name)

	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // no .gitignore, nothing to remove
		}
		return fmt.Errorf("reading .gitignore: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var result []string
	found := false

	for _, l := range lines {
		if strings.TrimSpace(l) == line {
			found = true
			continue // skip the line
		}
		result = append(result, l)
	}

	if !found {
		return nil // line wasn't present
	}

	output := strings.Join(result, "\n")
	if err := os.WriteFile(gitignorePath, []byte(output), 0o644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	return nil
}
