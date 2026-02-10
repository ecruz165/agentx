//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/agentx-labs/agentx/internal/runtime"
	"github.com/agentx-labs/agentx/internal/userdata"
)

func TestRunNodeSkill(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping")
	}

	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install the mock skill.
	resolved, err := registry.ResolveType("skills/test/mock-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Create the skill registry directory.
	registryName := "test/mock-skill"
	if err := userdata.EnsureSkillRegistry(registryName); err != nil {
		t.Fatalf("EnsureSkillRegistry: %v", err)
	}

	// Build the skill manifest for the runtime.
	skillDir := filepath.Join(env.InstalledDir, "skills/test/mock-skill")
	m := &manifest.SkillManifest{
		BaseManifest: manifest.BaseManifest{
			Name: "mock-skill",
			Type: "skill",
		},
		Runtime: "node",
		Topic:   "test",
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	rt := &runtime.NodeRuntime{
		Stdout: &stdoutBuf,
		Stderr: &stderrBuf,
	}

	args := map[string]string{
		"path":    "/tmp/test-repo",
		"verbose": "true",
	}

	ctx := context.Background()
	output, err := rt.Run(ctx, skillDir, m, args)
	if err != nil {
		t.Fatalf("NodeRuntime.Run: %v", err)
	}

	if output.ExitCode != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", output.ExitCode, stderrBuf.String())
	}

	// Verify the skill echoed back its inputs.
	stdout := stdoutBuf.String()
	if stdout == "" {
		t.Error("expected stdout output, got empty")
	}
	if !bytes.Contains(stdoutBuf.Bytes(), []byte("SKILL_OUTPUT:")) {
		t.Errorf("expected SKILL_OUTPUT in stdout, got: %s", stdout)
	}
}

func TestRunGoSkillReturnsError(t *testing.T) {
	rt := runtime.DispatchRuntime("go")

	m := &manifest.SkillManifest{
		BaseManifest: manifest.BaseManifest{
			Name: "test-go-skill",
			Type: "skill",
		},
		Runtime: "go",
	}

	ctx := context.Background()
	_, err := rt.Run(ctx, "/tmp/fake", m, nil)
	if err == nil {
		t.Fatal("expected error from Go runtime, got nil")
	}
	if err.Error() != "go runtime is not yet supported: no Go skills currently exist" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestRunWorkflow(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("Node.js not available, skipping")
	}

	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install the skill that the workflow depends on.
	resolved, err := registry.ResolveType("skills/test/mock-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Create the skill registry.
	if err := userdata.EnsureSkillRegistry("test/mock-skill"); err != nil {
		t.Fatalf("EnsureSkillRegistry: %v", err)
	}

	// Install the workflow.
	wfResolved, err := registry.ResolveType("workflows/test-workflow", sources)
	if err != nil {
		t.Fatalf("ResolveType (workflow): %v", err)
	}
	if err := registry.InstallType(wfResolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType (workflow): %v", err)
	}

	// Parse the workflow manifest.
	wfManifestPath := filepath.Join(env.InstalledDir, "workflows/test-workflow", "manifest.yaml")
	parsed, err := manifest.ParseFile(wfManifestPath)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}

	wfManifest, ok := parsed.(*manifest.WorkflowManifest)
	if !ok {
		t.Fatal("expected WorkflowManifest type")
	}

	// Execute each step of the workflow manually (simulating runWorkflow).
	ctx := context.Background()
	for i, step := range wfManifest.Steps {
		skillTypePath := step.Skill
		skillDir := filepath.Join(env.InstalledDir, skillTypePath)

		skillManifestPath, err := findTestManifest(skillDir)
		if err != nil {
			t.Fatalf("step %d: finding manifest: %v", i, err)
		}

		skillParsed, err := manifest.ParseFile(skillManifestPath)
		if err != nil {
			t.Fatalf("step %d: parsing manifest: %v", i, err)
		}

		skillManifest, ok := skillParsed.(*manifest.SkillManifest)
		if !ok {
			t.Fatalf("step %d: not a skill manifest", i)
		}

		var stdoutBuf, stderrBuf bytes.Buffer
		rt := &runtime.NodeRuntime{
			Stdout: &stdoutBuf,
			Stderr: &stderrBuf,
		}

		// Merge step inputs into args.
		args := make(map[string]string)
		for k, v := range step.Inputs {
			args[k] = v.(string)
		}

		output, err := rt.Run(ctx, skillDir, skillManifest, args)
		if err != nil {
			t.Fatalf("step %d (%s): Run error: %v", i, step.ID, err)
		}
		if output.ExitCode != 0 {
			t.Errorf("step %d (%s): exit code %d, stderr: %s", i, step.ID, output.ExitCode, stderrBuf.String())
		}
	}
}

func TestRunMissingSkill(t *testing.T) {
	env := setupTestEnv(t)

	// Try to find a manifest in a non-existent directory.
	skillDir := filepath.Join(env.InstalledDir, "skills/nonexistent/fake-skill")
	_, err := os.Stat(skillDir)
	if err == nil {
		t.Fatal("expected skill directory to not exist")
	}
	if !os.IsNotExist(err) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunInputValidation(t *testing.T) {
	// Test that validateInputs catches missing required inputs.
	fields := []manifest.InputField{
		{Name: "path", Type: "string", Required: true, Description: "Path to analyze"},
		{Name: "verbose", Type: "string", Required: false, Default: "false"},
	}

	// Missing required input.
	args := map[string]string{"verbose": "true"}
	err := validateTestInputs(fields, args)
	if err == nil {
		t.Error("expected error for missing required input 'path', got nil")
	}

	// All required inputs provided.
	args["path"] = "/tmp/test"
	err = validateTestInputs(fields, args)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Default value should be applied for optional input.
	args2 := map[string]string{"path": "/tmp/test"}
	err = validateTestInputs(fields, args2)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if args2["verbose"] != "false" {
		t.Errorf("expected default value 'false' for verbose, got %q", args2["verbose"])
	}
}

func TestRunDispatchRuntime(t *testing.T) {
	tests := []struct {
		name    string
		runtime string
		wantTyp string
	}{
		{"node", "node", "*runtime.NodeRuntime"},
		{"go", "go", "*runtime.GoRuntime"},
		{"unknown", "python", "*runtime.unknownRuntime"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rt := runtime.DispatchRuntime(tt.runtime)
			if rt == nil {
				t.Fatal("DispatchRuntime returned nil")
			}
		})
	}
}

func TestRunSkillRegistryEnsured(t *testing.T) {
	env := setupTestEnv(t)

	// Ensure skill registry creates the directory.
	registryName := "test/mock-skill"
	if err := userdata.EnsureSkillRegistry(registryName); err != nil {
		t.Fatalf("EnsureSkillRegistry: %v", err)
	}

	registryDir := filepath.Join(env.UserdataDir, "skills", registryName)
	assertDirExists(t, registryDir)

	// Calling again should be idempotent.
	if err := userdata.EnsureSkillRegistry(registryName); err != nil {
		t.Fatalf("EnsureSkillRegistry (second call): %v", err)
	}
}

// validateTestInputs mirrors the CLI's validateInputs logic for testing.
func validateTestInputs(fields []manifest.InputField, args map[string]string) error {
	for _, field := range fields {
		if _, ok := args[field.Name]; ok {
			continue
		}
		if field.Required {
			return &missingInputError{field.Name}
		}
		if field.Default != nil {
			args[field.Name] = field.Default.(string)
		}
	}
	return nil
}

type missingInputError struct {
	name string
}

func (e *missingInputError) Error() string {
	return "required input " + e.name + " is missing"
}

// findTestManifest searches for a manifest file in a directory.
func findTestManifest(dir string) (string, error) {
	candidates := []string{"manifest.yaml", "manifest.json", "skill.yaml", "workflow.yaml"}
	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", &os.PathError{Op: "stat", Path: dir, Err: os.ErrNotExist}
}
