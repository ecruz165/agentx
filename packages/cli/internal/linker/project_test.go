package linker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInitProject(t *testing.T) {
	tmpDir := t.TempDir()

	tools := []string{"claude-code", "copilot", "augment"}
	if err := InitProject(tmpDir, tools); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Verify .agentx/ directory exists
	agentxPath := filepath.Join(tmpDir, ".agentx")
	if info, err := os.Stat(agentxPath); err != nil || !info.IsDir() {
		t.Error(".agentx directory not created")
	}

	// Verify overrides/ directory exists
	overridesPath := filepath.Join(agentxPath, "overrides")
	if info, err := os.Stat(overridesPath); err != nil || !info.IsDir() {
		t.Error("overrides directory not created")
	}

	// Verify project.yaml exists and is parseable
	config, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed after init: %v", err)
	}

	if len(config.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(config.Tools))
	}
	if config.Tools[0] != "claude-code" {
		t.Errorf("expected first tool to be claude-code, got %s", config.Tools[0])
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	// Initialize first
	if err := InitProject(tmpDir, []string{"claude-code"}); err != nil {
		t.Fatalf("InitProject failed: %v", err)
	}

	// Load, modify, save, reload
	config, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed: %v", err)
	}

	config.Active.Personas = []string{"personas/senior-java-dev"}
	config.Active.Skills = []string{"skills/scm/git/commit-analyzer"}
	config.Active.Context = []string{"context/spring-boot/security"}

	if err := SaveProject(tmpDir, config); err != nil {
		t.Fatalf("SaveProject failed: %v", err)
	}

	// Reload and verify
	reloaded, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject (reload) failed: %v", err)
	}

	if len(reloaded.Active.Personas) != 1 || reloaded.Active.Personas[0] != "personas/senior-java-dev" {
		t.Errorf("personas not preserved: %v", reloaded.Active.Personas)
	}
	if len(reloaded.Active.Skills) != 1 || reloaded.Active.Skills[0] != "skills/scm/git/commit-analyzer" {
		t.Errorf("skills not preserved: %v", reloaded.Active.Skills)
	}
	if len(reloaded.Active.Context) != 1 || reloaded.Active.Context[0] != "context/spring-boot/security" {
		t.Errorf("context not preserved: %v", reloaded.Active.Context)
	}
}

func TestLoadProjectNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := LoadProject(tmpDir)
	if err == nil {
		t.Error("expected error when project.yaml doesn't exist")
	}
}

func TestProjectConfigPath(t *testing.T) {
	path := ProjectConfigPath("/some/project")
	expected := filepath.Join("/some/project", ".agentx", "project.yaml")
	if path != expected {
		t.Errorf("expected %s, got %s", expected, path)
	}
}

func TestInitProjectIdempotent(t *testing.T) {
	tmpDir := t.TempDir()

	tools := []string{"claude-code"}

	// First init
	if err := InitProject(tmpDir, tools); err != nil {
		t.Fatalf("first InitProject failed: %v", err)
	}

	// Second init should overwrite without error
	if err := InitProject(tmpDir, tools); err != nil {
		t.Fatalf("second InitProject failed: %v", err)
	}

	config, err := LoadProject(tmpDir)
	if err != nil {
		t.Fatalf("LoadProject failed after double init: %v", err)
	}
	if len(config.Tools) != 1 {
		t.Errorf("expected 1 tool, got %d", len(config.Tools))
	}
}
