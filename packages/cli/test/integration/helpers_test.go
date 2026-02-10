//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// testEnv holds paths to isolated test directories.
type testEnv struct {
	HomeDir      string // AGENTX_HOME — contains catalog/ and extensions/
	InstalledDir string // AGENTX_INSTALLED — where types get installed
	UserdataDir  string // AGENTX_USERDATA — userdata root (skills/, env/, etc.)
	ProjectDir   string // A mock project directory
}

// setupTestEnv creates isolated temp directories and sets environment variables
// so all AgentX operations are sandboxed. The env vars are restored after the test.
func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	env := &testEnv{
		HomeDir:      t.TempDir(),
		InstalledDir: t.TempDir(),
		UserdataDir:  t.TempDir(),
		ProjectDir:   t.TempDir(),
	}

	t.Setenv("AGENTX_HOME", env.HomeDir)
	t.Setenv("AGENTX_INSTALLED", env.InstalledDir)
	t.Setenv("AGENTX_USERDATA", env.UserdataDir)

	// Create essential userdata subdirectories.
	for _, sub := range []string{"skills", "env", "profiles"} {
		if err := os.MkdirAll(filepath.Join(env.UserdataDir, sub), 0755); err != nil {
			t.Fatalf("creating userdata/%s: %v", sub, err)
		}
	}

	return env
}

// setupCatalog creates a synthetic catalog inside homeDir with all type categories.
// Returns the catalog root path.
func setupCatalog(t *testing.T, homeDir string) string {
	t.Helper()

	catalogDir := filepath.Join(homeDir, "catalog")

	// --- Context ---
	writeManifest(t, catalogDir, "context/test-topic/docs", `name: docs
type: context
version: "1.0.0"
description: Test context documentation
format: md
sources:
  - ./
`)
	writeFile(t, filepath.Join(catalogDir, "context/test-topic/docs", "README.md"), "# Test Context\nSample context content.\n")

	// --- Persona ---
	writeManifest(t, catalogDir, "personas/test-persona", `name: test-persona
type: persona
version: "1.0.0"
description: A test persona
expertise:
  - testing
  - go
tone: professional
context:
  - context/test-topic/docs
`)

	// --- Skill (Node) ---
	writeManifest(t, catalogDir, "skills/test/mock-skill", `name: mock-skill
type: skill
version: "1.0.0"
description: A mock Node.js skill for testing
runtime: node
topic: test
cli_dependencies:
  - name: git
    min_version: "2.0.0"
inputs:
  - name: path
    type: string
    required: true
    description: Path to analyze
  - name: verbose
    type: string
    required: false
    default: "false"
    description: Enable verbose output
registry:
  tokens:
    - name: TEST_API_KEY
      required: true
      description: API key for testing
    - name: TEST_ENDPOINT
      required: false
      default: https://api.test.example.com
      description: Test API endpoint
  config:
    timeout: 30
    retries: 3
  state:
    - last-run.json
`)
	// Write a mock index.mjs that echoes back its inputs.
	writeFile(t, filepath.Join(catalogDir, "skills/test/mock-skill", "index.mjs"), `
const args = process.argv.slice(2);
if (args[0] === "run") {
  const input = JSON.parse(args[1] || "{}");
  console.log("SKILL_OUTPUT:" + JSON.stringify(input));
  process.exit(0);
} else {
  console.error("unknown command: " + args[0]);
  process.exit(1);
}
`)

	// --- Skill (Go runtime — stub) ---
	writeManifest(t, catalogDir, "skills/test/go-skill", `name: go-skill
type: skill
version: "1.0.0"
description: A Go runtime skill (stub)
runtime: go
topic: test
cli_dependencies: []
inputs:
  - name: text
    type: string
    required: true
    description: Text to process
`)

	// --- Workflow ---
	writeManifest(t, catalogDir, "workflows/test-workflow", `name: test-workflow
type: workflow
version: "1.0.0"
description: A test workflow with skill steps
runtime: node
steps:
  - id: analyze
    skill: skills/test/mock-skill
    inputs:
      path: .
  - id: analyze-again
    skill: skills/test/mock-skill
    inputs:
      path: ./src
`)

	// --- Prompt (with dependencies on all other types) ---
	writeManifest(t, catalogDir, "prompts/test-prompt", `name: test-prompt
type: prompt
version: "1.0.0"
description: A test prompt that references all type categories
persona: personas/test-persona
context:
  - context/test-topic/docs
skills:
  - skills/test/mock-skill
workflows:
  - workflows/test-workflow
`)

	// --- Template ---
	writeManifest(t, catalogDir, "templates/test-template", `name: test-template
type: template
version: "1.0.0"
description: A test template
format: hbs
variables:
  - name: project_name
    description: Name of the project
    required: true
`)
	writeFile(t, filepath.Join(catalogDir, "templates/test-template", "template.hbs"), "# {{project_name}}\n")

	return catalogDir
}

// setupExtension creates an extension directory with a type that overrides a catalog type.
func setupExtension(t *testing.T, homeDir, extName string) string {
	t.Helper()

	extDir := filepath.Join(homeDir, "extensions", extName)

	// Override the test-persona with a different version.
	writeManifest(t, extDir, "personas/test-persona", `name: test-persona
type: persona
version: "2.0.0"
description: Extension-overridden persona
expertise:
  - testing-override
tone: casual
`)

	return extDir
}

// writeManifest creates a manifest.yaml at catalogDir/<typePath>/manifest.yaml.
func writeManifest(t *testing.T, catalogDir, typePath, content string) {
	t.Helper()
	dir := filepath.Join(catalogDir, typePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating dir %s: %v", dir, err)
	}
	path := filepath.Join(dir, "manifest.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// writeFile creates a file at the given path with the given content.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating dir %s: %v", dir, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// assertFileExists fails the test if the file does not exist.
func assertFileExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Errorf("expected file to exist: %s (error: %v)", path, err)
	}
}

// assertFileNotExists fails the test if the file exists.
func assertFileNotExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err == nil {
		t.Errorf("expected file NOT to exist: %s", path)
	}
}

// assertDirExists fails the test if the directory does not exist.
func assertDirExists(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Errorf("expected directory to exist: %s (error: %v)", path, err)
		return
	}
	if !info.IsDir() {
		t.Errorf("expected %s to be a directory, but it is a file", path)
	}
}

// assertFileContains fails if the file doesn't exist or doesn't contain substr.
func assertFileContains(t *testing.T, path, substr string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Errorf("reading %s: %v", path, err)
		return
	}
	if !strings.Contains(string(data), substr) {
		t.Errorf("file %s does not contain %q.\nContents:\n%s", path, substr, string(data))
	}
}
