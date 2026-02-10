package userdata

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitGlobal_CreatesStructure(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	var buf bytes.Buffer
	if err := InitGlobal(&buf); err != nil {
		t.Fatalf("InitGlobal failed: %v", err)
	}

	output := buf.String()

	// Verify directories exist.
	assertDirExists(t, filepath.Join(tmp, "env"))
	assertDirExists(t, filepath.Join(tmp, "profiles"))
	assertDirExists(t, filepath.Join(tmp, "skills"))

	// Verify files exist.
	assertFileExists(t, filepath.Join(tmp, "env", "default.env"))
	assertFileExists(t, filepath.Join(tmp, "profiles", "default.yaml"))
	assertFileExists(t, filepath.Join(tmp, "preferences.yaml"))

	// Verify symlink.
	target, err := os.Readlink(filepath.Join(tmp, "profiles", "active"))
	if err != nil {
		t.Fatalf("reading active symlink: %v", err)
	}
	if target != "default.yaml" {
		t.Errorf("expected symlink target default.yaml, got %s", target)
	}

	// Verify permissions.
	assertDirPerm(t, filepath.Join(tmp, "env"), DirPermSecure)
	assertDirPerm(t, filepath.Join(tmp, "profiles"), DirPermSecure)
	assertDirPerm(t, filepath.Join(tmp, "skills"), DirPermNormal)

	// Verify output contains OK messages.
	if !strings.Contains(output, "[ OK ]") {
		t.Error("expected [ OK ] in output")
	}
}

func TestInitGlobal_Idempotent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	var buf1 bytes.Buffer
	if err := InitGlobal(&buf1); err != nil {
		t.Fatalf("first InitGlobal failed: %v", err)
	}

	// Run again — should succeed with SKIP messages.
	var buf2 bytes.Buffer
	if err := InitGlobal(&buf2); err != nil {
		t.Fatalf("second InitGlobal failed: %v", err)
	}

	output := buf2.String()
	if !strings.Contains(output, "[SKIP]") {
		t.Error("expected [SKIP] messages in second run")
	}

	// Verify files are unchanged.
	data, err := os.ReadFile(filepath.Join(tmp, "env", "default.env"))
	if err != nil {
		t.Fatalf("reading default.env: %v", err)
	}
	if !strings.Contains(string(data), "LOG_LEVEL=info") {
		t.Error("default.env content was corrupted")
	}
}

func TestEnsureSkillRegistry(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	// Must create skills/ first.
	os.MkdirAll(filepath.Join(tmp, "skills"), DirPermNormal)

	if err := EnsureSkillRegistry("cloud/aws/ssm-lookup"); err != nil {
		t.Fatalf("EnsureSkillRegistry failed: %v", err)
	}

	expected := filepath.Join(tmp, "skills", "cloud", "aws", "ssm-lookup")
	assertDirExists(t, expected)

	// Idempotent.
	if err := EnsureSkillRegistry("cloud/aws/ssm-lookup"); err != nil {
		t.Fatalf("second EnsureSkillRegistry failed: %v", err)
	}
}

func TestInitGlobal_DefaultEnvContent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	var buf bytes.Buffer
	if err := InitGlobal(&buf); err != nil {
		t.Fatalf("InitGlobal failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "env", "default.env"))
	if err != nil {
		t.Fatalf("reading default.env: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "LOG_LEVEL=info") {
		t.Error("missing LOG_LEVEL in default.env")
	}
	if !strings.Contains(content, "OUTPUT_FORMAT=json") {
		t.Error("missing OUTPUT_FORMAT in default.env")
	}
}

func TestInitGlobal_DefaultProfileContent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	var buf bytes.Buffer
	if err := InitGlobal(&buf); err != nil {
		t.Fatalf("InitGlobal failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "profiles", "default.yaml"))
	if err != nil {
		t.Fatalf("reading default.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: default") {
		t.Error("missing 'name: default' in default profile")
	}
}

func TestInitGlobal_PreferencesContent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("AGENTX_USERDATA", tmp)

	var buf bytes.Buffer
	if err := InitGlobal(&buf); err != nil {
		t.Fatalf("InitGlobal failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "preferences.yaml"))
	if err != nil {
		t.Fatalf("reading preferences.yaml: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "output_format: json") {
		t.Error("missing output_format in preferences.yaml")
	}
	if !strings.Contains(content, "color: true") {
		t.Error("missing color in preferences.yaml")
	}
}

// Helpers

func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("directory %s does not exist: %v", path, err)
	}
	if !info.IsDir() {
		t.Fatalf("%s is not a directory", path)
	}
}

func assertFileExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("file %s does not exist: %v", path, err)
	}
	if info.IsDir() {
		t.Fatalf("%s is a directory, expected file", path)
	}
}

func assertDirPerm(t *testing.T, path string, expected os.FileMode) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	actual := info.Mode().Perm()
	if actual != expected {
		t.Errorf("permissions on %s: expected %o, got %o", path, expected, actual)
	}
}
