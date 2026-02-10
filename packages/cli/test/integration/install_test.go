//go:build integration

package integration_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/agentx-labs/agentx/internal/registry"
)

func TestInstallSingleType(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Resolve and install a single context type.
	resolved, err := registry.ResolveType("context/test-topic/docs", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Verify the type was copied to the installed directory.
	installedPath := filepath.Join(env.InstalledDir, "context/test-topic/docs")
	assertDirExists(t, installedPath)
	assertFileExists(t, filepath.Join(installedPath, "manifest.yaml"))
	assertFileExists(t, filepath.Join(installedPath, "README.md"))
}

func TestInstallWithDependencies(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Build install plan for a prompt (which depends on persona + context + skill + workflow).
	plan, err := registry.BuildInstallPlan("prompts/test-prompt", sources, env.InstalledDir, false)
	if err != nil {
		t.Fatalf("BuildInstallPlan: %v", err)
	}

	if len(plan.AllTypes) == 0 {
		t.Fatal("expected types to install, got 0")
	}

	// Verify the plan includes all dependency categories.
	categories := make(map[string]bool)
	for _, rt := range plan.AllTypes {
		categories[rt.Category] = true
	}

	expectedCategories := []string{"prompt", "persona", "context", "skill", "workflow"}
	for _, cat := range expectedCategories {
		if !categories[cat] {
			t.Errorf("expected category %q in install plan, not found", cat)
		}
	}

	// Install all types from the plan.
	for _, resolved := range plan.AllTypes {
		if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
			t.Fatalf("InstallType(%s): %v", resolved.TypePath, err)
		}
	}

	// Verify all types are installed.
	expectedPaths := []string{
		"prompts/test-prompt",
		"personas/test-persona",
		"context/test-topic/docs",
		"skills/test/mock-skill",
		"workflows/test-workflow",
	}
	for _, p := range expectedPaths {
		assertDirExists(t, filepath.Join(env.InstalledDir, p))
		assertFileExists(t, filepath.Join(env.InstalledDir, p, "manifest.yaml"))
	}
}

func TestInstallNoDeps(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Build install plan with noDeps=true.
	plan, err := registry.BuildInstallPlan("prompts/test-prompt", sources, env.InstalledDir, true)
	if err != nil {
		t.Fatalf("BuildInstallPlan (noDeps): %v", err)
	}

	// Should only have the prompt itself, no dependencies.
	if len(plan.AllTypes) != 1 {
		t.Errorf("expected 1 type with --no-deps, got %d", len(plan.AllTypes))
	}

	if plan.AllTypes[0].TypePath != "prompts/test-prompt" {
		t.Errorf("expected prompts/test-prompt, got %s", plan.AllTypes[0].TypePath)
	}
}

func TestExtensionPriority(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)
	setupExtension(t, env.HomeDir, "test-ext")

	// Extension source comes first (higher priority).
	sources := []registry.Source{
		{Name: "test-ext", BasePath: filepath.Join(env.HomeDir, "extensions", "test-ext")},
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	resolved, err := registry.ResolveType("personas/test-persona", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	// Should resolve from the extension, not catalog.
	if resolved.SourceName != "test-ext" {
		t.Errorf("expected source 'test-ext', got %q", resolved.SourceName)
	}

	// Install and verify the extension version is installed.
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	installedManifest := filepath.Join(env.InstalledDir, "personas/test-persona", "manifest.yaml")
	assertFileContains(t, installedManifest, `version: "2.0.0"`)
	assertFileContains(t, installedManifest, "Extension-overridden persona")
}

func TestRegistryInitOnInstall(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Resolve and install the skill.
	resolved, err := registry.ResolveType("skills/test/mock-skill", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}

	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType: %v", err)
	}

	// Initialize the skill registry (this is what install command does after copy).
	warnings, err := registry.InitSkillRegistry(resolved, env.InstalledDir)
	if err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	// Should warn about required token without default.
	foundTokenWarning := false
	for _, w := range warnings {
		if w == "TEST_API_KEY required \u2014 edit tokens.env" {
			foundTokenWarning = true
		}
	}
	if !foundTokenWarning {
		t.Errorf("expected warning about TEST_API_KEY, warnings: %v", warnings)
	}

	// Verify registry directory was created.
	registryDir := filepath.Join(env.UserdataDir, "skills", "test/mock-skill")
	assertDirExists(t, registryDir)

	// Verify tokens.env was created.
	tokensPath := filepath.Join(registryDir, "tokens.env")
	assertFileExists(t, tokensPath)
	assertFileContains(t, tokensPath, "TEST_API_KEY")
	assertFileContains(t, tokensPath, "TEST_ENDPOINT=https://api.test.example.com")

	// Verify config.yaml was created.
	configPath := filepath.Join(registryDir, "config.yaml")
	assertFileExists(t, configPath)
	assertFileContains(t, configPath, "timeout")
	assertFileContains(t, configPath, "retries")

	// Verify state directory was created.
	assertDirExists(t, filepath.Join(registryDir, "state"))
}

func TestInstallAlreadyInstalled(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// Install a type first.
	resolved, err := registry.ResolveType("context/test-topic/docs", sources)
	if err != nil {
		t.Fatalf("ResolveType: %v", err)
	}
	if err := registry.InstallType(resolved, env.InstalledDir); err != nil {
		t.Fatalf("InstallType (first): %v", err)
	}

	// Build install plan again — the type should be marked as already installed.
	tree, err := registry.BuildDependencyTree("context/test-topic/docs", sources, env.InstalledDir)
	if err != nil {
		t.Fatalf("BuildDependencyTree: %v", err)
	}

	if !tree.Installed {
		t.Error("expected root node to be marked as Installed")
	}

	// FlattenTree should exclude installed types.
	flat := registry.FlattenTree(tree)
	if len(flat) != 0 {
		t.Errorf("expected 0 types to install (all already installed), got %d", len(flat))
	}
}

func TestInstallResolveNotFound(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	_, err := registry.ResolveType("skills/nonexistent/fake-skill", sources)
	if err == nil {
		t.Fatal("expected error for nonexistent type, got nil")
	}
}

func TestDependencyTreeDeduplication(t *testing.T) {
	env := setupTestEnv(t)
	setupCatalog(t, env.HomeDir)

	sources := []registry.Source{
		{Name: "catalog", BasePath: filepath.Join(env.HomeDir, "catalog")},
	}

	// The test-prompt depends on test-persona and test-workflow.
	// test-workflow depends on mock-skill (twice, in two steps).
	// The mock-skill should only appear once in the flattened plan.
	plan, err := registry.BuildInstallPlan("prompts/test-prompt", sources, env.InstalledDir, false)
	if err != nil {
		t.Fatalf("BuildInstallPlan: %v", err)
	}

	// Count occurrences of the skill in the flattened types.
	skillCount := 0
	for _, rt := range plan.AllTypes {
		if rt.TypePath == "skills/test/mock-skill" {
			skillCount++
		}
	}
	if skillCount != 1 {
		t.Errorf("expected mock-skill to appear once in flattened plan, got %d", skillCount)
	}
}

func TestCatalogFallbackResolution(t *testing.T) {
	env := setupTestEnv(t)

	// Create a catalog with only a JSON manifest (tests manifest.json fallback).
	catalogDir := filepath.Join(env.HomeDir, "catalog")
	jsonCtxDir := filepath.Join(catalogDir, "context", "json-test")
	if err := os.MkdirAll(jsonCtxDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(jsonCtxDir, "manifest.json"),
		[]byte(`{"name":"json-test","type":"context","version":"1.0.0","description":"json manifest","format":"md","sources":["./"]}`),
		0644,
	); err != nil {
		t.Fatal(err)
	}

	sources := []registry.Source{
		{Name: "catalog", BasePath: catalogDir},
	}

	resolved, err := registry.ResolveType("context/json-test", sources)
	if err != nil {
		t.Fatalf("ResolveType with JSON manifest: %v", err)
	}

	if filepath.Base(resolved.ManifestPath) != "manifest.json" {
		t.Errorf("expected manifest.json, got %s", filepath.Base(resolved.ManifestPath))
	}
}
