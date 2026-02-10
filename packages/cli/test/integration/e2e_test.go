//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentx-labs/agentx/internal/linker"
	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/agentx-labs/agentx/internal/userdata"
)

// TestFullFlowInstallAndLink tests the complete flow:
// init project -> install types with deps -> link types to project -> verify state.
func TestFullFlowInstallAndLink(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Step 1: Initialize a project with all tools.
	if err := linker.InitProject(env.ProjectDir, []string{"claude-code", "copilot", "augment"}); err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	assertFileExists(t, linker.ProjectConfigPath(env.ProjectDir))

	// Step 2: Install a prompt with all its dependencies.
	plan, err := registry.BuildInstallPlan("prompts/test-prompt", sources, env.InstalledDir, false)
	if err != nil {
		t.Fatalf("BuildInstallPlan: %v", err)
	}

	for _, resolved := range plan.AllTypes {
		if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
			t.Fatalf("InstallType(%s): %v", resolved.TypePath, err)
		}

		// Initialize skill registries.
		if resolved.Category == "skill" {
			if _, err := registry.InitSkillRegistry(resolved, env.InstalledDir); err != nil {
				t.Fatalf("InitSkillRegistry(%s): %v", resolved.TypePath, err)
			}
		}
	}

	// Step 3: Verify all types were installed.
	expectedTypes := []string{
		"prompts/test-prompt",
		"personas/test-persona",
		"context/test-topic/docs",
		"skills/test/mock-skill",
		"workflows/test-workflow",
	}
	for _, tp := range expectedTypes {
		assertDirExists(t, filepath.Join(env.InstalledDir, tp))
	}

	// Step 4: Link all types to the project.
	config, err := linker.LoadProject(env.ProjectDir)
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}

	config.Active.Personas = []string{"personas/test-persona"}
	config.Active.Context = []string{"context/test-topic/docs"}
	config.Active.Skills = []string{"skills/test/mock-skill"}
	config.Active.Workflows = []string{"workflows/test-workflow"}
	config.Active.Prompts = []string{"prompts/test-prompt"}

	if err := linker.SaveProject(env.ProjectDir, config); err != nil {
		t.Fatalf("SaveProject: %v", err)
	}

	// Step 5: Verify project.yaml is consistent.
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
	if len(reloaded.Active.Prompts) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(reloaded.Active.Prompts))
	}
	if len(reloaded.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(reloaded.Tools))
	}
}

// TestFullFlowSkillRegistryAfterInstall verifies that the skill registry
// is properly initialized after installing a skill with registry declarations.
func TestFullFlowSkillRegistryAfterInstall(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install the skill.
	resolved, err := registry.ResolveType("skills/test/mock-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Initialize registry.
	warnings, err := registry.InitSkillRegistry(resolved, env.InstalledDir)
	if err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	// Verify registry structure.
	regDir := filepath.Join(env.UserdataDir, "skills", "test/mock-skill")
	assertDirExists(t, regDir)
	assertFileExists(t, filepath.Join(regDir, "tokens.env"))
	assertFileExists(t, filepath.Join(regDir, "config.yaml"))
	assertDirExists(t, filepath.Join(regDir, "state"))

	// Verify tokens.env has the declared tokens.
	assertFileContains(t, filepath.Join(regDir, "tokens.env"), "TEST_API_KEY")
	assertFileContains(t, filepath.Join(regDir, "tokens.env"), "TEST_ENDPOINT=https://api.test.example.com")

	// Verify config.yaml has the declared config.
	assertFileContains(t, filepath.Join(regDir, "config.yaml"), "timeout")

	// Verify warning about required token.
	hasWarning := false
	for _, w := range warnings {
		if w == "TEST_API_KEY required \u2014 edit tokens.env" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected warning about TEST_API_KEY, got: %v", warnings)
	}
}

// TestFullFlowExtensionOverride verifies that extensions take priority over catalog.
func TestFullFlowExtensionOverride(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)
	setupExtension(t, env.HomeDir, "corp-ext")

	// Extension comes first for higher priority.
	sources := []registry.Source{
		{Name: "corp-ext", BasePath: filepath.Join(env.HomeDir, "extensions", "corp-ext")},
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install persona -- should come from extension.
	resolved, err := registry.ResolveType("personas/test-persona", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if resolved.SourceName != "corp-ext" {
		t.Fatalf("expected source 'corp-ext', got %q", resolved.SourceName)
	}

	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Install context -- should come from catalog (not in extension).
	ctxResolved, err := registry.ResolveType("context/test-topic/docs", sources)
	if err != nil {
		t.Fatalf("ResolveType (context): %v", err)
	}

	if ctxResolved.SourceName != "catalog" {
		t.Errorf("expected context from 'catalog', got %q", ctxResolved.SourceName)
	}

	if err := registry.InstallType(ctxResolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType (context): %v", err)
	}

	// Verify extension version was installed.
	assertFileContains(t, filepath.Join(env.InstalledDir, "personas/test-persona", "manifest.yaml"), `version: "2.0.0"`)
}

// TestFullFlowUserdataInit verifies the global userdata initialization flow.
func TestFullFlowUserdataInit(t *testing.T) {
	env := setupTestEnv(t)

	// InitGlobal creates the full userdata structure.
	if err := userdata.InitGlobal(os.Stdout); err != nil {
		t.Fatalf("InitGlobal: %v", err)
	}

	// Verify directory structure.
	assertDirExists(t, filepath.Join(env.UserdataDir, "env"))
	assertDirExists(t, filepath.Join(env.UserdataDir, "profiles"))
	assertDirExists(t, filepath.Join(env.UserdataDir, "skills"))

	// Verify default files.
	assertFileExists(t, filepath.Join(env.UserdataDir, "env", "default.env"))
	assertFileExists(t, filepath.Join(env.UserdataDir, "profiles", "default.yaml"))
	assertFileExists(t, filepath.Join(env.UserdataDir, "preferences.yaml"))

	// Verify active symlink exists.
	activePath := filepath.Join(env.UserdataDir, "profiles", "active")
	info, err := os.Lstat(activePath)
	if err != nil {
		t.Fatalf("Lstat active symlink: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("expected 'active' to be a symlink")
	}

	// Verify the symlink target.
	target, err := os.Readlink(activePath)
	if err != nil {
		t.Fatalf("Readlink: %v", err)
	}
	if target != "default.yaml" {
		t.Errorf("expected symlink target 'default.yaml', got %q", target)
	}

	// Calling InitGlobal again should be idempotent (skip existing).
	if err := userdata.InitGlobal(os.Stdout); err != nil {
		t.Fatalf("InitGlobal (second call): %v", err)
	}
}

// TestFullFlowRegistryCheckAfterInstall verifies the doctor registry check
// works correctly after installing skills.
func TestFullFlowRegistryCheckAfterInstall(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install the skill.
	resolved, err := registry.ResolveType("skills/test/mock-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}
	if _, err := registry.InitSkillRegistry(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	// Run the registry check.
	result, err := userdata.SkillRegistryStatus("test/mock-skill", env.InstalledDir, env.UserdataDir)
	if err != nil {
		t.Fatalf("SkillRegistryStatus: %v", err)
	}

	if !result.FolderExists {
		t.Error("expected registry folder to exist")
	}

	// TEST_API_KEY is required but has no default -- should be in missing tokens.
	foundMissing := false
	for _, m := range result.MissingTokens {
		if m == "TEST_API_KEY" {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Errorf("expected TEST_API_KEY in missing tokens, got: %v", result.MissingTokens)
	}
}

// TestFullFlowReinstallOverwrites verifies that reinstalling a type
// cleanly replaces the existing installation.
func TestFullFlowReinstallOverwrites(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install the context.
	resolved, err := registry.ResolveType("context/test-topic/docs", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType (first): %v", err)
	}

	installedDir := filepath.Join(env.InstalledDir, "context/test-topic/docs")

	// Add a rogue file to the installed directory.
	rogueFile := filepath.Join(installedDir, "rogue.txt")
	if err := os.WriteFile(rogueFile, []byte("rogue"), 0644); err != nil {
		t.Fatalf("writing rogue file: %v", err)
	}
	assertFileExists(t, rogueFile)

	// Reinstall -- should remove the rogue file (clean copy).
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType (reinstall): %v", err)
	}

	// The rogue file should be gone after reinstall.
	assertFileNotExists(t, rogueFile)

	// The manifest should still be there.
	assertFileExists(t, filepath.Join(installedDir, "manifest.yaml"))
}
