//go:build integration

package integration_test

import (
	"path/filepath"
	"testing"

	"github.com/agentx-labs/agentx/internal/linker"
	"github.com/agentx-labs/agentx/internal/registry"
)

func TestLinkAddUpdatesProjectYAML(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	// Initialize the project.
	if err := linker.InitProject(env.ProjectDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Verify project.yaml was created.
	projectYAML := linker.ProjectConfigPath(env.ProjectDir)
	assertFileExists(t, projectYAML)

	// Simulate AddType without Sync (Sync requires Node).
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	config.Active.Personas = append(config.Active.Personas, "personas/test-persona")
	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	// Reload and verify.
	reloaded, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject (reload): %v", err)
	}

	found := false
	for _, p := range reloaded.Active.Personas {
		if p == "personas/test-persona" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected personas/test-persona in active.personas, got %v", reloaded.Active.Personas)
	}
}

func TestLinkAddMultipleTypes(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	if err := linker.InitProject(env.ProjectDir, []string{"claude-code", "copilot"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Add multiple types (simulated without Sync).
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	config.Active.Personas = append(config.Active.Personas, "personas/test-persona")
	config.Active.Context = append(config.Active.Context, "context/test-topic/docs")
	config.Active.Skills = append(config.Active.Skills, "skills/test/mock-skill")
	config.Active.Workflows = append(config.Active.Workflows, "workflows/test-workflow")

	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	reloaded, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject (reload): %v", err)
	}

	if len(reloaded.Active.Personas) != 1 {
		t.Errorf("expected 1 persona, got %d", len(reloaded.Active.Personas))
	}
	if len(reloaded.Active.Context) != 1 {
		t.Errorf("expected 1 context, got %d", len(reloaded.Active.Context))
	}
	if len(reloaded.Active.Skills) != 1 {
		t.Errorf("expected 1 skill, got %d", len(reloaded.Active.Skills))
	}
	if len(reloaded.Active.Workflows) != 1 {
		t.Errorf("expected 1 workflow, got %d", len(reloaded.Active.Workflows))
	}
}

func TestLinkRemoveUpdatesProjectYAML(t *testing.T) {
	env := setupTestEnv(t)

	if err := linker.InitProject(env.ProjectDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Add a skill.
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	config.Active.Skills = []string{"skills/test/mock-skill", "skills/test/go-skill"}
	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	// Remove one skill (simulated without Sync).
	config, err = linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	newSkills := make([]string, 0)
	for _, s := range config.Active.Skills {
		if s != "skills/test/mock-skill" {
			newSkills = append(newSkills, s)
		}
	}
	config.Active.Skills = newSkills
	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	reloaded, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject (reload): %v", err)
	}

	if len(reloaded.Active.Skills) != 1 {
		t.Errorf("expected 1 skill after removal, got %d", len(reloaded.Active.Skills))
	}
	for _, s := range reloaded.Active.Skills {
		if s == "skills/test/mock-skill" {
			t.Error("skills/test/mock-skill should have been removed")
		}
	}
}

func TestLinkDuplicateDetection(t *testing.T) {
	env := setupTestEnv(t)

	if err := linker.InitProject(env.ProjectDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Add a persona.
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	config.Active.Personas = []string{"personas/test-persona"}
	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	// Try to add the same persona again via AddType (this calls Sync, which may fail without Node,
	// but the duplicate check happens before Sync).
	err = linker.AddType(env.ProjectDir, "personas/test-persona")
	if err == nil {
		t.Error("expected error for duplicate add, got nil")
	}
}

func TestLinkInitProjectCreatesStructure(t *testing.T) {
	env := setupTestEnv(t)

	tools := []string{"claude-code", "copilot", "augment"}
	if err := linker.InitProject(env.ProjectDir, tools); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Verify .agentx/ directory exists.
	assertDirExists(t, filepath.Join(env.ProjectDir, ".agentx"))

	// Verify project.yaml exists.
	assertFileExists(t, filepath.Join(env.ProjectDir, ".agentx", "project.yaml"))

	// Verify overrides directory exists.
	assertDirExists(t, filepath.Join(env.ProjectDir, ".agentx", "overrides"))

	// Verify project.yaml has the tools.
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	if len(config.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d: %v", len(config.Tools), config.Tools)
	}
}

func TestLinkInitAlreadyInitialized(t *testing.T) {
	env := setupTestEnv(t)

	if err := linker.InitProject(env.ProjectDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject (first): %v", err)
	}

	// Second init should still succeed (InitProject creates dirs with MkdirAll).
	// But the CLI checks for existing project.yaml and returns an error.
	// Here we test the raw InitProject function which overwrites.
	if err := linker.InitProject(env.ProjectDir, []string{"copilot"}); err != nil {
		t.Fatalf("InitProject (second): %v", err)
	}

	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	// Should have the new tools list.
	if len(config.Tools) != 1 || config.Tools[0] != "copilot" {
		t.Errorf("expected [copilot], got %v", config.Tools)
	}
}

func TestLinkProjectConfigPath(t *testing.T) {
	path := linker.ProjectConfigPath("/tmp/my-project")
	expected := filepath.Join("/tmp/my-project", ".agentx", "project.yaml")
	if path != expected {
		t.Errorf("ProjectConfigPath = %q, want %q", path, expected)
	}
}

func TestInstallThenLinkFlow(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install a persona and context.
	for _, typePath := range []string{"personas/test-persona", "context/test-topic/docs"} {
		resolved, err := registry.ResolveType(typePath, sources)
		if err != nil {
			t.Fatalf("ResolveType(%s): %v", typePath, err)
		}
		if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
			t.Fatalf("InstallType(%s): %v", typePath, err)
		}
	}

	// Init project.
	if err := linker.InitProject(env.ProjectDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}

	// Add the types to the project config (without Sync).
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	config.Active.Personas = []string{"personas/test-persona"}
	config.Active.Context = []string{"context/test-topic/docs"}
	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	// Verify both the installed types and the project config are consistent.
	assertDirExists(t, filepath.Join(env.InstalledDir, "personas/test-persona"))
	assertDirExists(t, filepath.Join(env.InstalledDir, "context/test-topic/docs"))

	reloaded, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject (reload): %v", err)
	}
	if len(reloaded.Active.Personas) != 1 {
		t.Errorf("expected 1 persona in project config, got %d", len(reloaded.Active.Personas))
	}
	if len(reloaded.Active.Context) != 1 {
		t.Errorf("expected 1 context in project config, got %d", len(reloaded.Active.Context))
	}
}
