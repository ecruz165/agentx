package registry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitSkillRegistryCreatesTokensEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", filepath.Join(tmpDir, "userdata"))
	t.Setenv("AGENTX_INSTALLED", filepath.Join(tmpDir, "installed"))

	// Copy the skill manifest to the installed location.
	installedRoot := filepath.Join(tmpDir, "installed")
	skillDir := filepath.Join(installedRoot, "skills", "test", "basic-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcManifest := filepath.Join(testdataDir(), "catalog", "skills", "test", "basic-skill", "manifest.yaml")
	data, err := os.ReadFile(srcManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	resolved := &ResolvedType{
		TypePath:     "skills/test/basic-skill",
		ManifestPath: filepath.Join(skillDir, "manifest.yaml"),
		SourceDir:    skillDir,
		SourceName:   "catalog",
		Category:     "skill",
	}

	warnings, err := InitSkillRegistry(resolved, installedRoot)
	if err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	// Check tokens.env was created.
	tokensPath := filepath.Join(tmpDir, "userdata", "skills", "test", "basic-skill", "tokens.env")
	tokensData, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatalf("reading tokens.env: %v", err)
	}

	content := string(tokensData)
	if !strings.Contains(content, "TEST_API_KEY=") {
		t.Error("tokens.env should contain TEST_API_KEY")
	}
	if !strings.Contains(content, "TEST_ENDPOINT=https://api.test.example.com") {
		t.Error("tokens.env should contain TEST_ENDPOINT with default")
	}
	if !strings.Contains(content, "(required)") {
		t.Error("tokens.env should mark required tokens")
	}

	// Check warning for required token without default.
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0], "TEST_API_KEY") {
		t.Errorf("warning should mention TEST_API_KEY, got %q", warnings[0])
	}
}

func TestInitSkillRegistryCreatesConfigYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", filepath.Join(tmpDir, "userdata"))
	t.Setenv("AGENTX_INSTALLED", filepath.Join(tmpDir, "installed"))

	installedRoot := filepath.Join(tmpDir, "installed")
	skillDir := filepath.Join(installedRoot, "skills", "test", "basic-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcManifest := filepath.Join(testdataDir(), "catalog", "skills", "test", "basic-skill", "manifest.yaml")
	data, err := os.ReadFile(srcManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	resolved := &ResolvedType{
		TypePath:     "skills/test/basic-skill",
		ManifestPath: filepath.Join(skillDir, "manifest.yaml"),
		SourceDir:    skillDir,
		SourceName:   "catalog",
		Category:     "skill",
	}

	if _, err := InitSkillRegistry(resolved, installedRoot); err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	configPath := filepath.Join(tmpDir, "userdata", "skills", "test", "basic-skill", "config.yaml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("reading config.yaml: %v", err)
	}

	content := string(configData)
	if !strings.Contains(content, "timeout") {
		t.Error("config.yaml should contain timeout")
	}
	if !strings.Contains(content, "retries") {
		t.Error("config.yaml should contain retries")
	}
}

func TestInitSkillRegistryCreatesSubdirs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", filepath.Join(tmpDir, "userdata"))
	t.Setenv("AGENTX_INSTALLED", filepath.Join(tmpDir, "installed"))

	installedRoot := filepath.Join(tmpDir, "installed")
	skillDir := filepath.Join(installedRoot, "skills", "test", "basic-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	srcManifest := filepath.Join(testdataDir(), "catalog", "skills", "test", "basic-skill", "manifest.yaml")
	data, err := os.ReadFile(srcManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}

	resolved := &ResolvedType{
		TypePath:     "skills/test/basic-skill",
		ManifestPath: filepath.Join(skillDir, "manifest.yaml"),
		SourceDir:    skillDir,
		SourceName:   "catalog",
		Category:     "skill",
	}

	if _, err := InitSkillRegistry(resolved, installedRoot); err != nil {
		t.Fatalf("InitSkillRegistry: %v", err)
	}

	regDir := filepath.Join(tmpDir, "userdata", "skills", "test", "basic-skill")

	// state/ should be created (manifest declares state files).
	if _, err := os.Stat(filepath.Join(regDir, "state")); err != nil {
		t.Error("state/ directory should be created")
	}

	// output/ should be created (manifest declares output schema).
	if _, err := os.Stat(filepath.Join(regDir, "output")); err != nil {
		t.Error("output/ directory should be created")
	}
}

func TestInitSkillRegistrySkipsNonSkill(t *testing.T) {
	resolved := &ResolvedType{
		TypePath: "personas/test-persona",
		Category: "persona",
	}

	warnings, err := InitSkillRegistry(resolved, "/tmp/fake")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for non-skill, got %v", warnings)
	}
}

func TestInitSkillRegistrySkipsNoRegistry(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("AGENTX_USERDATA", filepath.Join(tmpDir, "userdata"))
	t.Setenv("AGENTX_INSTALLED", filepath.Join(tmpDir, "installed"))

	// Create a skill manifest without a registry block.
	installedRoot := filepath.Join(tmpDir, "installed")
	skillDir := filepath.Join(installedRoot, "skills", "no-reg-skill")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "name: no-reg-skill\ntype: skill\nversion: \"1.0.0\"\ndescription: no registry\nruntime: node\ntopic: test\n"
	if err := os.WriteFile(filepath.Join(skillDir, "manifest.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}

	resolved := &ResolvedType{
		TypePath:     "skills/no-reg-skill",
		ManifestPath: filepath.Join(skillDir, "manifest.yaml"),
		SourceDir:    skillDir,
		SourceName:   "catalog",
		Category:     "skill",
	}

	warnings, err := InitSkillRegistry(resolved, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("expected no warnings for skill without registry, got %v", warnings)
	}

	// No registry directory should be created under userdata.
	regDir := filepath.Join(tmpDir, "userdata", "skills", "no-reg-skill")
	if _, err := os.Stat(regDir); err == nil {
		t.Error("registry directory should not be created for skill without registry block")
	}
}
