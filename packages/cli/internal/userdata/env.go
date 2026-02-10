package userdata

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// EnvEntry represents a single key-value pair from a .env file.
type EnvEntry struct {
	Key   string
	Value string
}

// ListEnvFiles discovers all .env files in the userdata directory.
// Returns shared env files (from env/) and skill-specific files (from skills/*/tokens.env).
func ListEnvFiles() (shared []string, skillSpecific []string, err error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return nil, nil, err
	}

	// Shared env files.
	envDir := filepath.Join(root, EnvDir)
	if entries, readErr := os.ReadDir(envDir); readErr == nil {
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".env") {
				shared = append(shared, filepath.Join(envDir, e.Name()))
			}
		}
	}

	// Skill-specific tokens.env files.
	skillsDir := filepath.Join(root, SkillsDir)
	_ = filepath.WalkDir(skillsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip errors
		}
		if !d.IsDir() && d.Name() == "tokens.env" {
			skillSpecific = append(skillSpecific, path)
		}
		return nil
	})

	return shared, skillSpecific, nil
}

// ResolveEnvTarget resolves a target string to a .env file path.
// If the target contains "/" it is treated as a skill path (-> skills/<path>/tokens.env).
// Otherwise it is treated as a vendor name (-> env/<target>.env).
func ResolveEnvTarget(target string) (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}

	if strings.Contains(target, "/") {
		// Skill path.
		return filepath.Join(root, SkillsDir, target, "tokens.env"), nil
	}
	// Vendor name.
	return filepath.Join(root, EnvDir, target+".env"), nil
}

// ParseEnvFile reads a .env file and returns key-value entries.
// It skips blank lines and lines starting with #.
func ParseEnvFile(path string) ([]EnvEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening env file %s: %w", path, err)
	}
	defer f.Close()

	var entries []EnvEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		entries = append(entries, EnvEntry{
			Key:   strings.TrimSpace(key),
			Value: strings.TrimSpace(value),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading env file %s: %w", path, err)
	}
	return entries, nil
}

// sensitivePatterns are substrings that indicate a value should be redacted.
var sensitivePatterns = []string{"TOKEN", "SECRET", "PASSWORD", "KEY", "CREDENTIAL"}

// RedactValue returns a redacted version of value if the key name contains
// a sensitive pattern (case-insensitive substring match).
// Values with 4+ chars show the first 4 chars + "***".
// Values with fewer than 4 chars are fully redacted as "***".
func RedactValue(key, value string) string {
	upper := strings.ToUpper(key)
	for _, pattern := range sensitivePatterns {
		if strings.Contains(upper, pattern) {
			if len(value) >= 4 {
				return value[:4] + "***"
			}
			return "***"
		}
	}
	return value
}

// OpenEditor opens the given file in the user's preferred editor.
// It checks $EDITOR and falls back to notepad on Windows or vi on Unix.
func OpenEditor(filePath string) error {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		if runtime.GOOS == "windows" {
			editor = "notepad"
		} else {
			editor = "vi"
		}
	}

	cmd := exec.Command(editor, filePath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("running editor %s: %w", editor, err)
	}
	return nil
}
