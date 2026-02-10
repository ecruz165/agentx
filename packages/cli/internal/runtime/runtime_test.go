package runtime

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/agentx-labs/agentx/internal/manifest"
)

func TestDispatchRuntime_Node(t *testing.T) {
	rt := DispatchRuntime("node")
	if _, ok := rt.(*NodeRuntime); !ok {
		t.Errorf("DispatchRuntime(\"node\") returned %T, want *NodeRuntime", rt)
	}
}

func TestDispatchRuntime_Go(t *testing.T) {
	rt := DispatchRuntime("go")
	if _, ok := rt.(*GoRuntime); !ok {
		t.Errorf("DispatchRuntime(\"go\") returned %T, want *GoRuntime", rt)
	}
}

func TestDispatchRuntime_Unknown(t *testing.T) {
	rt := DispatchRuntime("python")
	if _, ok := rt.(*unknownRuntime); !ok {
		t.Errorf("DispatchRuntime(\"python\") returned %T, want *unknownRuntime", rt)
	}

	// Verify it returns an error when run.
	_, err := rt.Run(context.Background(), "", nil, nil)
	if err == nil {
		t.Error("expected error from unknown runtime, got nil")
	}
}

func TestGoRuntime_NotSupported(t *testing.T) {
	rt := &GoRuntime{}
	_, err := rt.Run(context.Background(), "/tmp/test", &manifest.SkillManifest{}, nil)
	if err == nil {
		t.Fatal("expected error from GoRuntime, got nil")
	}
	if got := err.Error(); got != "go runtime is not yet supported: no Go skills currently exist" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestNodeRuntime_MissingEntryPoint(t *testing.T) {
	// Skip if Node.js is not available.
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping")
	}

	rt := &NodeRuntime{}
	dir := t.TempDir()

	m := &manifest.SkillManifest{
		BaseManifest: manifest.BaseManifest{
			Name: "test-skill",
			Type: "skill",
		},
		Runtime: "node",
	}

	_, err := rt.Run(context.Background(), dir, m, nil)
	if err == nil {
		t.Fatal("expected error for missing entry point, got nil")
	}
}

func TestNodeRuntime_ExecMockScript(t *testing.T) {
	// Skip if Node.js is not available.
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping")
	}

	// Create a temporary directory with a mock index.mjs.
	dir := t.TempDir()
	script := `
// Mock skill script: echo back the args it received.
const args = process.argv.slice(2);
if (args[0] === "run") {
	const input = JSON.parse(args[1] || "{}");
	console.log("SKILL_OUTPUT:" + JSON.stringify(input));
	process.exit(0);
} else {
	console.error("unknown command: " + args[0]);
	process.exit(1);
}
`
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	// Set up environment overrides so userdata paths resolve to temp dirs.
	userdataDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", userdataDir)

	// Create skill registry directory.
	registryDir := filepath.Join(userdataDir, "skills", "test-skill")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatal(err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	rt := &NodeRuntime{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	m := &manifest.SkillManifest{
		BaseManifest: manifest.BaseManifest{
			Name: "test-skill",
			Type: "skill",
		},
		Runtime: "node",
	}

	args := map[string]string{
		"repo":   "my-repo",
		"branch": "main",
	}

	output, err := rt.Run(context.Background(), dir, m, args)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if output.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d", output.ExitCode)
	}

	// Verify stdout contains our skill output.
	if got := stdoutBuf.String(); got == "" {
		t.Error("expected stdout output, got empty string")
	}
}

func TestNodeRuntime_NonZeroExit(t *testing.T) {
	// Skip if Node.js is not available.
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping")
	}

	dir := t.TempDir()
	script := `console.error("intentional failure"); process.exit(42);`
	if err := os.WriteFile(filepath.Join(dir, "index.mjs"), []byte(script), 0644); err != nil {
		t.Fatal(err)
	}

	userdataDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", userdataDir)

	registryDir := filepath.Join(userdataDir, "skills", "fail-skill")
	if err := os.MkdirAll(registryDir, 0755); err != nil {
		t.Fatal(err)
	}

	var stdoutBuf, stderrBuf bytes.Buffer

	rt := &NodeRuntime{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	m := &manifest.SkillManifest{
		BaseManifest: manifest.BaseManifest{
			Name: "fail-skill",
			Type: "skill",
		},
		Runtime: "node",
	}

	output, err := rt.Run(context.Background(), dir, m, nil)
	if err != nil {
		t.Fatalf("unexpected error (non-zero exit should not be an error): %v", err)
	}

	if output.ExitCode != 42 {
		t.Errorf("expected exit code 42, got %d", output.ExitCode)
	}
}

func TestSetEnv(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		key      string
		value    string
		expected []string
	}{
		{
			name:     "add new variable",
			env:      []string{"FOO=bar"},
			key:      "BAZ",
			value:    "qux",
			expected: []string{"FOO=bar", "BAZ=qux"},
		},
		{
			name:     "replace existing variable",
			env:      []string{"FOO=bar", "BAZ=old"},
			key:      "BAZ",
			value:    "new",
			expected: []string{"FOO=bar", "BAZ=new"},
		},
		{
			name:     "add to empty env",
			env:      nil,
			key:      "KEY",
			value:    "val",
			expected: []string{"KEY=val"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := setEnv(tt.env, tt.key, tt.value)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d entries, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, e := range tt.expected {
				if result[i] != e {
					t.Errorf("env[%d] = %q, want %q", i, result[i], e)
				}
			}
		})
	}
}

func TestLoadTokensEnv(t *testing.T) {
	data := []byte(`# Comment line
FOO=bar
BAZ=qux

# Another comment
EMPTY=
SPACED = value
`)
	env := loadTokensEnv(nil, data)

	expected := map[string]string{
		"FOO":    "bar",
		"BAZ":    "qux",
		"SPACED": "value",
	}

	envMap := make(map[string]string)
	for _, e := range env {
		parts := splitFirst(e, "=")
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	for k, v := range expected {
		if got, ok := envMap[k]; !ok {
			t.Errorf("missing env var %s", k)
		} else if got != v {
			t.Errorf("env var %s = %q, want %q", k, got, v)
		}
	}

	// EMPTY= should not be set (value is empty).
	if _, ok := envMap["EMPTY"]; ok {
		t.Error("EMPTY should not be set (empty value)")
	}
}

// splitFirst splits s on the first occurrence of sep.
func splitFirst(s, sep string) []string {
	i := 0
	for i < len(s) && string(s[i]) != sep {
		i++
	}
	if i == len(s) {
		return []string{s}
	}
	return []string{s[:i], s[i+1:]}
}
