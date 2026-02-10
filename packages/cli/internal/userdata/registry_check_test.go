package userdata

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// setupInstalledSkill creates a mock installed skill with a manifest in the temp dir.
// Returns the installed root path and the skill path (e.g. "scm/git/commit-analyzer").
func setupInstalledSkill(t *testing.T, installedRoot, skillPath, manifestContent string) {
	t.Helper()
	dir := filepath.Join(installedRoot, "skills", skillPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating skill install dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "manifest.yaml"), []byte(manifestContent), 0644); err != nil {
		t.Fatalf("writing manifest: %v", err)
	}
}

// setupRegistryDir creates a mock registry directory for a skill.
func setupRegistryDir(t *testing.T, userdataRoot, skillPath string) string {
	t.Helper()
	dir := filepath.Join(userdataRoot, "skills", skillPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("creating registry dir: %v", err)
	}
	return dir
}

const testSkillManifestNoRegistry = `name: simple-skill
type: skill
version: "1.0.0"
description: A simple skill with no registry
runtime: node
topic: testing
`

const testSkillManifestWithTokens = `name: commit-analyzer
type: skill
version: "1.0.0"
description: Analyzes git commits
runtime: node
topic: scm
registry:
  tokens:
    - name: GITHUB_TOKEN
      required: true
      description: GitHub personal access token
    - name: LOG_LEVEL
      required: false
      default: info
      description: Logging level
`

const testSkillManifestWithConfig = `name: deploy-tool
type: skill
version: "1.0.0"
description: Deploys to cloud
runtime: node
topic: cloud
registry:
  tokens:
    - name: AWS_ACCESS_KEY_ID
      required: true
    - name: AWS_SECRET_ACCESS_KEY
      required: true
  config:
    region: us-east-1
    timeout: 30
    retries: 3
`

func TestCheckRegistry_NoInstalledSkills(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	// No installed/skills/ directory at all.
	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] No installed skills directory found") {
		t.Errorf("expected missing skills directory message, got:\n%s", output)
	}
}

func TestCheckRegistry_EmptyInstalledDir(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	// Create empty skills/ directory.
	os.MkdirAll(filepath.Join(installedRoot, "skills"), 0755)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] No installed skills to check") {
		t.Errorf("expected no installed skills message, got:\n%s", output)
	}
}

func TestCheckRegistry_MissingRegistryFolder(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	// Set up an installed skill but no registry folder.
	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] Registry folder missing") {
		t.Errorf("expected missing registry folder, got:\n%s", output)
	}
}

func TestCheckRegistry_RegistryFolderExists(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	// Set up installed skill and registry dir.
	setupInstalledSkill(t, installedRoot, "testing/simple", testSkillManifestNoRegistry)
	setupRegistryDir(t, userdataRoot, "testing/simple")

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] Registry folder exists") {
		t.Errorf("expected registry folder exists, got:\n%s", output)
	}
}

func TestCheckRegistry_RequiredTokensMissing(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Create tokens.env with empty required token.
	tokensContent := "GITHUB_TOKEN=\nLOG_LEVEL=debug\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0600)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[WARN] Required token GITHUB_TOKEN not set") {
		t.Errorf("expected warning about missing GITHUB_TOKEN, got:\n%s", output)
	}
}

func TestCheckRegistry_AllRequiredTokensSet(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Create tokens.env with all tokens set.
	tokensContent := "GITHUB_TOKEN=ghp_abc123\nLOG_LEVEL=debug\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0600)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] All required tokens set") {
		t.Errorf("expected all tokens set, got:\n%s", output)
	}
}

func TestCheckRegistry_ConfigPresent(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "cloud/aws/deploy", testSkillManifestWithConfig)
	regDir := setupRegistryDir(t, userdataRoot, "cloud/aws/deploy")

	// Create tokens.env with required tokens set.
	tokensContent := "AWS_ACCESS_KEY_ID=AKIA123\nAWS_SECRET_ACCESS_KEY=secret123\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0600)

	// Create config.yaml with 3 keys.
	configContent := "region: us-east-1\ntimeout: 30\nretries: 3\n"
	os.WriteFile(filepath.Join(regDir, "config.yaml"), []byte(configContent), 0644)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] config.yaml present (3 keys)") {
		t.Errorf("expected config with 3 keys, got:\n%s", output)
	}
}

func TestCheckRegistry_ConfigMissing(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "cloud/aws/deploy", testSkillManifestWithConfig)
	regDir := setupRegistryDir(t, userdataRoot, "cloud/aws/deploy")

	// Create tokens.env but no config.yaml.
	tokensContent := "AWS_ACCESS_KEY_ID=AKIA123\nAWS_SECRET_ACCESS_KEY=secret123\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0600)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] config.yaml not found") {
		t.Errorf("expected missing config, got:\n%s", output)
	}
}

func TestCheckRegistry_TokensEnvBadPermissions(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Create tokens.env with bad permissions (0644 instead of 0600).
	tokensContent := "GITHUB_TOKEN=ghp_abc123\nLOG_LEVEL=debug\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0644)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[WARN] tokens.env permissions: 0644 (should be 0600)") {
		t.Errorf("expected permissions warning, got:\n%s", output)
	}
}

func TestCheckRegistry_TokensEnvGoodPermissions(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Create tokens.env with correct permissions.
	tokensContent := "GITHUB_TOKEN=ghp_abc123\nLOG_LEVEL=debug\n"
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte(tokensContent), 0600)

	var buf bytes.Buffer
	err := CheckRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] tokens.env permissions: 0600") {
		t.Errorf("expected permissions OK, got:\n%s", output)
	}
}

// TraceEnv tests

func TestTraceEnv_NoTokensDeclared(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "testing/simple", testSkillManifestNoRegistry)

	var buf bytes.Buffer
	err := TraceEnv(&buf, "testing/simple", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(no tokens declared in manifest)") {
		t.Errorf("expected no tokens message, got:\n%s", output)
	}
}

func TestTraceEnv_ResolutionChain(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Set up profile env with GITHUB_TOKEN.
	envDir := filepath.Join(userdataRoot, "env")
	os.MkdirAll(envDir, 0700)
	os.WriteFile(filepath.Join(envDir, "default.env"), []byte("GITHUB_TOKEN=ghp_profile_val123\n"), 0600)

	// Set up registry tokens.env (without GITHUB_TOKEN, so profile wins).
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte("LOG_LEVEL=debug\n"), 0600)

	var buf bytes.Buffer
	err := TraceEnv(&buf, "scm/git/commit-analyzer", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check GITHUB_TOKEN resolution.
	if !strings.Contains(output, "GITHUB_TOKEN:") {
		t.Errorf("expected GITHUB_TOKEN section, got:\n%s", output)
	}
	// Profile should have a value (redacted since it contains TOKEN).
	if !strings.Contains(output, "Profile:  ghp_***") {
		t.Errorf("expected redacted profile value, got:\n%s", output)
	}
	// Final should come from profile.
	if !strings.Contains(output, "(from profile)") {
		t.Errorf("expected final from profile, got:\n%s", output)
	}
}

func TestTraceEnv_RegistryOverridesProfile(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Set up profile env.
	envDir := filepath.Join(userdataRoot, "env")
	os.MkdirAll(envDir, 0700)
	os.WriteFile(filepath.Join(envDir, "default.env"), []byte("GITHUB_TOKEN=ghp_profile_val\n"), 0600)

	// Registry also has GITHUB_TOKEN -> should win.
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte("GITHUB_TOKEN=ghp_registry_val\n"), 0600)

	var buf bytes.Buffer
	err := TraceEnv(&buf, "scm/git/commit-analyzer", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "(from registry)") {
		t.Errorf("expected final from registry, got:\n%s", output)
	}
}

func TestTraceEnv_DefaultValueUsed(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// No profile env, no registry tokens.env.
	// LOG_LEVEL has default "info" in the manifest.

	var buf bytes.Buffer
	err := TraceEnv(&buf, "scm/git/commit-analyzer", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// LOG_LEVEL should show default value.
	if !strings.Contains(output, "LOG_LEVEL:") {
		t.Errorf("expected LOG_LEVEL section, got:\n%s", output)
	}
	if !strings.Contains(output, "Default:  info") {
		t.Errorf("expected default value 'info', got:\n%s", output)
	}
	if !strings.Contains(output, "(from default)") {
		t.Errorf("expected final from default, got:\n%s", output)
	}
}

func TestTraceEnv_RedactionOfSensitiveValues(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Put a long token value.
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte("GITHUB_TOKEN=ghp_abc123def456\n"), 0600)

	var buf bytes.Buffer
	err := TraceEnv(&buf, "scm/git/commit-analyzer", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// GITHUB_TOKEN contains "TOKEN" which is a sensitive pattern.
	// The value "ghp_abc123def456" should be redacted to "ghp_***".
	if !strings.Contains(output, "ghp_***") {
		t.Errorf("expected redacted token value, got:\n%s", output)
	}

	// The full value should NOT appear.
	if strings.Contains(output, "ghp_abc123def456") {
		t.Errorf("full token value should be redacted, got:\n%s", output)
	}
}

func TestTraceEnv_ManifestNotFound(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	// No skill installed.
	var buf bytes.Buffer
	err := TraceEnv(&buf, "nonexistent/skill", installedRoot, userdataRoot)
	if err == nil {
		t.Fatal("expected error for missing manifest")
	}
	if !strings.Contains(err.Error(), "no manifest found") {
		t.Errorf("expected 'no manifest found' error, got: %v", err)
	}
}

func TestTraceEnv_TokenNotSetAnywhere(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// GITHUB_TOKEN has no default, no profile, no registry -> should be (not set).
	var buf bytes.Buffer
	err := TraceEnv(&buf, "scm/git/commit-analyzer", installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Check that the GITHUB_TOKEN final value is (not set).
	lines := strings.Split(output, "\n")
	inGithubToken := false
	for _, line := range lines {
		if strings.Contains(line, "GITHUB_TOKEN:") {
			inGithubToken = true
			continue
		}
		if inGithubToken && strings.Contains(line, "Final:") {
			if !strings.Contains(line, "(not set)") {
				t.Errorf("expected GITHUB_TOKEN final to be (not set), got: %s", line)
			}
			break
		}
		// Stop if we hit another token section.
		if inGithubToken && !strings.HasPrefix(strings.TrimSpace(line), "") {
			break
		}
	}
}

// Test the unwrapPathError helper.
func TestUnwrapPathError(t *testing.T) {
	// A regular os.PathError should be unwrapped to itself.
	pathErr := &os.PathError{Op: "open", Path: "/nonexistent", Err: os.ErrNotExist}
	result := unwrapPathError(pathErr)
	if result != pathErr {
		t.Errorf("expected same PathError, got different error")
	}
	if !os.IsNotExist(result) {
		t.Errorf("expected IsNotExist to be true")
	}
}

// Test resolveTokenValue logic.
func TestResolveTokenValue(t *testing.T) {
	tests := []struct {
		name        string
		tokenDef    string
		profileVal  string
		profileHas  bool
		registryVal string
		registryHas bool
		wantVal     string
		wantSource  string
	}{
		{
			name:       "default only",
			tokenDef:   "defaultval",
			wantVal:    "defaultval",
			wantSource: "default",
		},
		{
			name:       "profile overrides default",
			tokenDef:   "defaultval",
			profileVal: "profileval",
			profileHas: true,
			wantVal:    "profileval",
			wantSource: "profile",
		},
		{
			name:        "registry overrides profile",
			tokenDef:    "defaultval",
			profileVal:  "profileval",
			profileHas:  true,
			registryVal: "registryval",
			registryHas: true,
			wantVal:     "registryval",
			wantSource:  "registry",
		},
		{
			name:        "empty registry does not override profile",
			tokenDef:    "defaultval",
			profileVal:  "profileval",
			profileHas:  true,
			registryVal: "",
			registryHas: true,
			wantVal:     "profileval",
			wantSource:  "profile",
		},
		{
			name:       "empty profile does not override default",
			tokenDef:   "defaultval",
			profileVal: "",
			profileHas: true,
			wantVal:    "defaultval",
			wantSource: "default",
		},
		{
			name:       "no value anywhere",
			tokenDef:   "",
			wantVal:    "",
			wantSource: "default",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := struct {
				Name     string
				Required bool
				Default  string
			}{Default: tt.tokenDef}
			// Use the manifest type directly.
			mToken := newTestRegistryToken(token.Default)
			gotVal, gotSource := resolveTokenValue(mToken, tt.profileVal, tt.profileHas, tt.registryVal, tt.registryHas)
			if gotVal != tt.wantVal {
				t.Errorf("resolveTokenValue value = %q, want %q", gotVal, tt.wantVal)
			}
			if gotSource != tt.wantSource {
				t.Errorf("resolveTokenValue source = %q, want %q", gotSource, tt.wantSource)
			}
		})
	}
}

// newTestRegistryToken is a test helper to create a manifest.RegistryToken
// with just a default value. We import the type through the function signature.
func newTestRegistryToken(defaultVal string) manifest.RegistryToken {
	return manifest.RegistryToken{
		Name:    "TEST_TOKEN",
		Default: defaultVal,
	}
}

// ── FixRegistry tests ──

func TestFixRegistry_CreatesMissingFolder(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)

	var buf bytes.Buffer
	err := FixRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[FIX ] Created registry folder") {
		t.Errorf("expected registry folder creation, got:\n%s", output)
	}

	// Verify folder was created.
	regDir := filepath.Join(userdataRoot, "skills", "scm/git/commit-analyzer")
	if _, err := os.Stat(regDir); os.IsNotExist(err) {
		t.Error("expected registry folder to exist after fix")
	}
}

func TestFixRegistry_CreatesTokensEnv(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)

	var buf bytes.Buffer
	err := FixRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[FIX ] Created tokens.env") {
		t.Errorf("expected tokens.env creation, got:\n%s", output)
	}

	// Verify tokens.env exists.
	tokensPath := filepath.Join(userdataRoot, "skills", "scm/git/commit-analyzer", "tokens.env")
	data, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatalf("reading tokens.env: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "GITHUB_TOKEN=") {
		t.Errorf("expected GITHUB_TOKEN in tokens.env, got:\n%s", content)
	}
	if !strings.Contains(content, "LOG_LEVEL=info") {
		t.Errorf("expected LOG_LEVEL=info (default) in tokens.env, got:\n%s", content)
	}

	// Verify permissions.
	info, _ := os.Stat(tokensPath)
	if info.Mode().Perm() != FilePermSecure {
		t.Errorf("expected permissions %o, got %o", FilePermSecure, info.Mode().Perm())
	}
}

func TestFixRegistry_CreatesConfigYAML(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "cloud/aws/deploy", testSkillManifestWithConfig)

	var buf bytes.Buffer
	err := FixRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[FIX ] Created config.yaml") {
		t.Errorf("expected config.yaml creation, got:\n%s", output)
	}

	// Verify config.yaml exists.
	configPath := filepath.Join(userdataRoot, "skills", "cloud/aws/deploy", "config.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("expected config.yaml to exist after fix")
	}
}

func TestFixRegistry_SkipsExistingFiles(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Pre-create tokens.env with custom content.
	os.WriteFile(filepath.Join(regDir, "tokens.env"), []byte("GITHUB_TOKEN=my-custom-token\n"), 0600)

	var buf bytes.Buffer
	err := FixRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Tokens.env should NOT be overwritten.
	data, _ := os.ReadFile(filepath.Join(regDir, "tokens.env"))
	if !strings.Contains(string(data), "my-custom-token") {
		t.Errorf("expected existing tokens.env to be preserved, got:\n%s", string(data))
	}
}

func TestFixRegistry_FixesPermissions(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")
	userdataRoot := filepath.Join(tmp, "userdata")

	setupInstalledSkill(t, installedRoot, "scm/git/commit-analyzer", testSkillManifestWithTokens)
	regDir := setupRegistryDir(t, userdataRoot, "scm/git/commit-analyzer")

	// Create tokens.env with wrong permissions.
	tokensPath := filepath.Join(regDir, "tokens.env")
	os.WriteFile(tokensPath, []byte("GITHUB_TOKEN=abc\n"), 0644)

	var buf bytes.Buffer
	err := FixRegistry(&buf, installedRoot, userdataRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[FIX ] Fixed tokens.env permissions") {
		t.Errorf("expected permissions fix, got:\n%s", output)
	}

	info, _ := os.Stat(tokensPath)
	if info.Mode().Perm() != FilePermSecure {
		t.Errorf("expected permissions %o after fix, got %o", FilePermSecure, info.Mode().Perm())
	}
}

// ── CheckCLIDeps tests ──

const testSkillManifestWithCLIDeps = `name: git-analyzer
type: skill
version: "1.0.0"
description: Analyzes git repos
runtime: node
topic: scm
cli_dependencies:
  - name: git
  - name: agentx_nonexistent_tool_xyz123
`

func TestCheckCLIDeps_FindsInstalledCLI(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")

	setupInstalledSkill(t, installedRoot, "scm/git/analyzer", testSkillManifestWithCLIDeps)

	var buf bytes.Buffer
	err := CheckCLIDeps(&buf, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// git should be found on any dev machine.
	if !strings.Contains(output, "[ OK ] git found at") {
		t.Errorf("expected git to be found, got:\n%s", output)
	}
}

func TestCheckCLIDeps_ReportsMissing(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")

	setupInstalledSkill(t, installedRoot, "scm/git/analyzer", testSkillManifestWithCLIDeps)

	var buf bytes.Buffer
	err := CheckCLIDeps(&buf, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] agentx_nonexistent_tool_xyz123") {
		t.Errorf("expected missing CLI reported, got:\n%s", output)
	}
	if !strings.Contains(output, "1 missing CLI dependency") {
		t.Errorf("expected summary with 1 missing, got:\n%s", output)
	}
}

func TestCheckCLIDeps_NoCLIDeps(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")

	setupInstalledSkill(t, installedRoot, "testing/simple", testSkillManifestNoRegistry)

	var buf bytes.Buffer
	err := CheckCLIDeps(&buf, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] No CLI dependencies declared") {
		t.Errorf("expected no CLI deps message, got:\n%s", output)
	}
}

func TestCheckCLIDeps_NoInstalledSkills(t *testing.T) {
	tmp := t.TempDir()
	installedRoot := filepath.Join(tmp, "installed")

	var buf bytes.Buffer
	err := CheckCLIDeps(&buf, installedRoot)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] No installed skills directory found") {
		t.Errorf("expected missing skills directory message, got:\n%s", output)
	}
}

// ── CheckLinks tests ──

func TestCheckLinks_NoProjectYAML(t *testing.T) {
	tmp := t.TempDir()

	var buf bytes.Buffer
	err := CheckLinks(&buf, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[MISS] No .agentx/project.yaml found") {
		t.Errorf("expected no project.yaml message, got:\n%s", output)
	}
}

func TestCheckLinks_ValidSymlinks(t *testing.T) {
	tmp := t.TempDir()

	// Create .agentx/project.yaml.
	agentxDir := filepath.Join(tmp, ".agentx")
	os.MkdirAll(agentxDir, 0755)
	projectContent := "tools:\n  - claude-code\nactive:\n  skills:\n    - skills/test\n"
	os.WriteFile(filepath.Join(agentxDir, "project.yaml"), []byte(projectContent), 0644)

	// Create .claude/ directory with a valid symlink.
	claudeDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(claudeDir, 0755)

	// Create a real target file.
	targetFile := filepath.Join(tmp, "real-file.md")
	os.WriteFile(targetFile, []byte("content"), 0644)

	// Create symlink to it.
	os.Symlink(targetFile, filepath.Join(claudeDir, "linked.md"))

	var buf bytes.Buffer
	err := CheckLinks(&buf, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[ OK ] claude-code: 1 symlink(s) valid") {
		t.Errorf("expected valid symlink message, got:\n%s", output)
	}
}

func TestCheckLinks_BrokenSymlinks(t *testing.T) {
	tmp := t.TempDir()

	// Create .agentx/project.yaml.
	agentxDir := filepath.Join(tmp, ".agentx")
	os.MkdirAll(agentxDir, 0755)
	projectContent := "tools:\n  - claude-code\nactive:\n  skills:\n    - skills/test\n"
	os.WriteFile(filepath.Join(agentxDir, "project.yaml"), []byte(projectContent), 0644)

	// Create .claude/ directory with a broken symlink.
	claudeDir := filepath.Join(tmp, ".claude")
	os.MkdirAll(claudeDir, 0755)
	os.Symlink("/nonexistent/path/to/file", filepath.Join(claudeDir, "broken.md"))

	var buf bytes.Buffer
	err := CheckLinks(&buf, tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "[FAIL]") && !strings.Contains(output, "(broken)") {
		t.Errorf("expected broken link reported, got:\n%s", output)
	}
	if !strings.Contains(output, "1 broken link(s) found") {
		t.Errorf("expected broken link summary, got:\n%s", output)
	}
}

// ── generateTokensEnvTemplate tests ──

func TestGenerateTokensEnvTemplate(t *testing.T) {
	tokens := []manifest.RegistryToken{
		{Name: "API_KEY", Required: true, Description: "API key for the service"},
		{Name: "LOG_LEVEL", Required: false, Default: "info", Description: "Logging level"},
	}

	content := generateTokensEnvTemplate(tokens)

	if !strings.Contains(content, "API_KEY=") {
		t.Errorf("expected API_KEY= in template, got:\n%s", content)
	}
	if !strings.Contains(content, "LOG_LEVEL=info") {
		t.Errorf("expected LOG_LEVEL=info in template, got:\n%s", content)
	}
	if !strings.Contains(content, "# (required)") {
		t.Errorf("expected required marker, got:\n%s", content)
	}
	if !strings.Contains(content, "# API key for the service") {
		t.Errorf("expected description comment, got:\n%s", content)
	}
}
