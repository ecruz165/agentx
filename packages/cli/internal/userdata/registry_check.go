package userdata

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/platform"
	"go.yaml.in/yaml/v3"
)

// CheckRegistry validates all installed skill registries.
// For each installed skill it verifies:
//   - Registry folder exists at userdataRoot/skills/<path>/
//   - Required tokens are set (non-empty) in tokens.env
//   - config.yaml has expected keys from manifest registry.config
//   - tokens.env file permissions are 0600
func CheckRegistry(w io.Writer, installedRoot, userdataRoot string) error {
	fmt.Fprintln(w, "Registry check:")

	// Find all installed skills by walking installedRoot/skills/.
	skillsInstalled := filepath.Join(installedRoot, "skills")
	if _, err := os.Stat(skillsInstalled); os.IsNotExist(err) {
		fmt.Fprintln(w, "  [MISS] No installed skills directory found")
		return nil
	}

	skills, err := discoverInstalledSkills(skillsInstalled)
	if err != nil {
		return fmt.Errorf("discovering installed skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(w, "  [ OK ] No installed skills to check")
		return nil
	}

	skillsUserdata := filepath.Join(userdataRoot, SkillsDir)

	for _, sp := range skills {
		fmt.Fprintln(w, sp.typePath)
		checkOneSkillRegistry(w, sp, skillsUserdata)
	}

	return nil
}

// installedSkill holds information about a discovered installed skill.
type installedSkill struct {
	typePath     string // e.g. "scm/git/commit-analyzer"
	manifestPath string // absolute path to the manifest file
}

// discoverInstalledSkills walks the installed skills directory and finds all
// skills with manifest files.
func discoverInstalledSkills(skillsInstalled string) ([]installedSkill, error) {
	var skills []installedSkill

	err := filepath.WalkDir(skillsInstalled, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if d.IsDir() {
			return nil
		}
		name := d.Name()
		if name != "manifest.yaml" && name != "skill.yaml" {
			return nil
		}

		dir := filepath.Dir(path)
		rel, err := filepath.Rel(skillsInstalled, dir)
		if err != nil {
			return nil
		}

		skills = append(skills, installedSkill{
			typePath:     rel,
			manifestPath: path,
		})
		return nil
	})

	return skills, err
}

// checkOneSkillRegistry validates the registry for a single installed skill.
func checkOneSkillRegistry(w io.Writer, sp installedSkill, skillsUserdata string) {
	regDir := filepath.Join(skillsUserdata, sp.typePath)

	// Check registry folder exists.
	if _, err := os.Stat(regDir); os.IsNotExist(err) {
		fmt.Fprintf(w, "  [MISS] Registry folder missing\n")
		return
	}
	fmt.Fprintf(w, "  [ OK ] Registry folder exists\n")

	// Parse the manifest to get registry declarations.
	parsed, err := manifest.ParseFile(sp.manifestPath)
	if err != nil {
		fmt.Fprintf(w, "  [WARN] Could not parse manifest: %v\n", err)
		return
	}

	skill, ok := parsed.(*manifest.SkillManifest)
	if !ok {
		fmt.Fprintf(w, "  [WARN] Manifest is not a skill type\n")
		return
	}

	// Check tokens.
	checkTokens(w, regDir, skill)

	// Check config.
	checkConfig(w, regDir, skill)

	// Check tokens.env permissions.
	checkTokensEnvPermissions(w, regDir)
}

// checkTokens validates that required tokens declared in the manifest are set
// in the skill's tokens.env file.
func checkTokens(w io.Writer, regDir string, skill *manifest.SkillManifest) {
	if skill.Registry == nil || len(skill.Registry.Tokens) == 0 {
		fmt.Fprintf(w, "  [ OK ] No required tokens\n")
		return
	}

	tokensPath := filepath.Join(regDir, "tokens.env")
	entries, err := ParseEnvFile(tokensPath)
	if err != nil {
		// tokens.env might not exist.
		if os.IsNotExist(unwrapPathError(err)) {
			fmt.Fprintf(w, "  [MISS] tokens.env not found\n")
			// Report each required token as missing.
			for _, t := range skill.Registry.Tokens {
				if t.Required {
					fmt.Fprintf(w, "  [WARN] Required token %s not set\n", t.Name)
				}
			}
			return
		}
		fmt.Fprintf(w, "  [WARN] Could not read tokens.env: %v\n", err)
		return
	}

	// Build a lookup of actual values.
	values := make(map[string]string)
	for _, e := range entries {
		values[e.Key] = e.Value
	}

	missingCount := 0
	for _, t := range skill.Registry.Tokens {
		if !t.Required {
			continue
		}
		val, exists := values[t.Name]
		if !exists || val == "" {
			fmt.Fprintf(w, "  [WARN] Required token %s not set\n", t.Name)
			missingCount++
		}
	}

	if missingCount == 0 {
		fmt.Fprintf(w, "  [ OK ] All required tokens set\n")
	}
}

// checkConfig validates that config.yaml exists and has the expected keys
// from the manifest registry.config block.
func checkConfig(w io.Writer, regDir string, skill *manifest.SkillManifest) {
	if skill.Registry == nil || len(skill.Registry.Config) == 0 {
		return // no config declared, nothing to check
	}

	configPath := filepath.Join(regDir, "config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(w, "  [MISS] config.yaml not found (expected %d keys)\n", len(skill.Registry.Config))
			return
		}
		fmt.Fprintf(w, "  [WARN] Could not read config.yaml: %v\n", err)
		return
	}

	var parsed map[string]interface{}
	if err := yaml.Unmarshal(data, &parsed); err != nil {
		fmt.Fprintf(w, "  [WARN] Could not parse config.yaml: %v\n", err)
		return
	}

	fmt.Fprintf(w, "  [ OK ] config.yaml present (%d keys)\n", len(parsed))
}

// checkTokensEnvPermissions checks that tokens.env has secure permissions (0600).
func checkTokensEnvPermissions(w io.Writer, regDir string) {
	tokensPath := filepath.Join(regDir, "tokens.env")
	info, err := os.Stat(tokensPath)
	if err != nil {
		return // file may not exist, already reported
	}

	perm := info.Mode().Perm()
	if perm != FilePermSecure {
		fmt.Fprintf(w, "  [WARN] tokens.env permissions: %04o (should be %04o)\n", perm, FilePermSecure)
	} else {
		fmt.Fprintf(w, "  [ OK ] tokens.env permissions: %04o\n", perm)
	}
}

// TraceEnv shows the env resolution order for a specific skill.
// For each token declared in the manifest:
//   - Show: default value -> profile env -> skill tokens.env -> final value
//   - Redact sensitive values
func TraceEnv(w io.Writer, skillPath, installedRoot, userdataRoot string) error {
	fmt.Fprintf(w, "Env trace for %s:\n", skillPath)

	// Find and parse the skill manifest from the installed directory.
	manifestPath, err := findSkillManifest(installedRoot, skillPath)
	if err != nil {
		return fmt.Errorf("finding skill manifest for %s: %w", skillPath, err)
	}

	parsed, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing skill manifest: %w", err)
	}

	skill, ok := parsed.(*manifest.SkillManifest)
	if !ok {
		return fmt.Errorf("manifest at %s is not a skill", manifestPath)
	}

	if skill.Registry == nil || len(skill.Registry.Tokens) == 0 {
		fmt.Fprintln(w, "  (no tokens declared in manifest)")
		return nil
	}

	// Load profile env values.
	profileEnv := loadProfileEnv(userdataRoot)

	// Load skill-specific tokens.env.
	registryEnv := loadRegistryEnv(userdataRoot, skillPath)

	for _, token := range skill.Registry.Tokens {
		fmt.Fprintf(w, "  %s:\n", token.Name)

		// Default value from manifest.
		defaultVal := token.Default
		if defaultVal == "" {
			fmt.Fprintln(w, "    Default:  (none)")
		} else {
			fmt.Fprintf(w, "    Default:  %s\n", RedactValue(token.Name, defaultVal))
		}

		// Profile env value.
		profileVal, profileHas := profileEnv[token.Name]
		if profileHas && profileVal != "" {
			fmt.Fprintf(w, "    Profile:  %s\n", RedactValue(token.Name, profileVal))
		} else {
			fmt.Fprintln(w, "    Profile:  (not set)")
		}

		// Registry tokens.env value.
		registryVal, registryHas := registryEnv[token.Name]
		if registryHas && registryVal != "" {
			fmt.Fprintf(w, "    Registry: %s\n", RedactValue(token.Name, registryVal))
		} else {
			fmt.Fprintln(w, "    Registry: (not set)")
		}

		// Determine final value and its source.
		finalVal, source := resolveTokenValue(token, profileVal, profileHas, registryVal, registryHas)
		if finalVal == "" {
			fmt.Fprintln(w, "    Final:    (not set)")
		} else {
			fmt.Fprintf(w, "    Final:    %s (from %s)\n", RedactValue(token.Name, finalVal), source)
		}
	}

	return nil
}

// findSkillManifest finds the manifest file for a skill in the installed directory.
// skillPath should be the path relative to skills/ (e.g. "scm/git/commit-analyzer").
func findSkillManifest(installedRoot, skillPath string) (string, error) {
	// Try both "skills/<path>" and just "<path>" prefixed forms.
	candidates := []string{
		filepath.Join(installedRoot, "skills", skillPath, "manifest.yaml"),
		filepath.Join(installedRoot, "skills", skillPath, "skill.yaml"),
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}

	return "", fmt.Errorf("no manifest found in installed directory for skill %s", skillPath)
}

// loadProfileEnv loads environment variables from the default.env file in the
// userdata env/ directory.
func loadProfileEnv(userdataRoot string) map[string]string {
	result := make(map[string]string)

	defaultEnvPath := filepath.Join(userdataRoot, EnvDir, DefaultEnvFile)
	entries, err := ParseEnvFile(defaultEnvPath)
	if err != nil {
		return result
	}

	for _, e := range entries {
		result[e.Key] = e.Value
	}

	return result
}

// loadRegistryEnv loads environment variables from the skill's tokens.env file.
func loadRegistryEnv(userdataRoot, skillPath string) map[string]string {
	result := make(map[string]string)

	tokensPath := filepath.Join(userdataRoot, SkillsDir, skillPath, "tokens.env")
	entries, err := ParseEnvFile(tokensPath)
	if err != nil {
		return result
	}

	for _, e := range entries {
		result[e.Key] = e.Value
	}

	return result
}

// resolveTokenValue determines the final value and its source by following the
// resolution chain: default -> profile -> registry (last non-empty wins).
func resolveTokenValue(token manifest.RegistryToken, profileVal string, profileHas bool, registryVal string, registryHas bool) (string, string) {
	finalVal := token.Default
	source := "default"

	if profileHas && profileVal != "" {
		finalVal = profileVal
		source = "profile"
	}

	if registryHas && registryVal != "" {
		finalVal = registryVal
		source = "registry"
	}

	return finalVal, source
}

// unwrapPathError extracts the underlying error from a wrapped error chain,
// useful for checking os.IsNotExist on wrapped errors.
func unwrapPathError(err error) error {
	for {
		if e, ok := err.(*os.PathError); ok {
			return e
		}
		u, ok := err.(interface{ Unwrap() error })
		if !ok {
			return err
		}
		err = u.Unwrap()
	}
}

// TokensEnvFile is the conventional name for the tokens file in a skill registry.
const TokensEnvFile = "tokens.env"

// ConfigYAMLFile is the conventional name for the config file in a skill registry.
const ConfigYAMLFile = "config.yaml"

// RegistryCheckResult captures the result of a single skill registry check.
// Exported for use by callers that want to programmatically inspect results.
type RegistryCheckResult struct {
	SkillPath       string
	FolderExists    bool
	MissingTokens   []string
	ConfigKeyCount  int
	ConfigMissing   bool
	PermissionsOK   bool
	ActualPerm      os.FileMode
	ParseError      string
}

// FixRegistry creates missing registry folders, templated tokens.env files,
// and config.yaml files for all installed skills.
func FixRegistry(w io.Writer, installedRoot, userdataRoot string) error {
	fmt.Fprintln(w, "Registry fix:")

	skillsInstalled := filepath.Join(installedRoot, "skills")
	if _, err := os.Stat(skillsInstalled); os.IsNotExist(err) {
		fmt.Fprintln(w, "  [MISS] No installed skills directory found")
		return nil
	}

	skills, err := discoverInstalledSkills(skillsInstalled)
	if err != nil {
		return fmt.Errorf("discovering installed skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(w, "  [ OK ] No installed skills to fix")
		return nil
	}

	skillsUserdata := filepath.Join(userdataRoot, SkillsDir)

	for _, sp := range skills {
		fmt.Fprintln(w, sp.typePath)
		fixOneSkillRegistry(w, sp, skillsUserdata)
	}

	return nil
}

// fixOneSkillRegistry creates missing registry artifacts for a single skill.
func fixOneSkillRegistry(w io.Writer, sp installedSkill, skillsUserdata string) {
	regDir := filepath.Join(skillsUserdata, sp.typePath)

	// Create registry folder if missing.
	if _, err := os.Stat(regDir); os.IsNotExist(err) {
		if mkErr := os.MkdirAll(regDir, DirPermNormal); mkErr != nil {
			fmt.Fprintf(w, "  [FAIL] Could not create registry folder: %v\n", mkErr)
			return
		}
		fmt.Fprintf(w, "  [FIX ] Created registry folder %s\n", regDir)
	} else {
		fmt.Fprintf(w, "  [ OK ] Registry folder exists\n")
	}

	// Parse the manifest.
	parsed, err := manifest.ParseFile(sp.manifestPath)
	if err != nil {
		fmt.Fprintf(w, "  [WARN] Could not parse manifest: %v\n", err)
		return
	}

	skill, ok := parsed.(*manifest.SkillManifest)
	if !ok {
		fmt.Fprintf(w, "  [WARN] Manifest is not a skill type\n")
		return
	}

	// Generate tokens.env if there are token declarations and the file is missing.
	if skill.Registry != nil && len(skill.Registry.Tokens) > 0 {
		tokensPath := filepath.Join(regDir, TokensEnvFile)
		if _, err := os.Stat(tokensPath); os.IsNotExist(err) {
			content := generateTokensEnvTemplate(skill.Registry.Tokens)
			if writeErr := os.WriteFile(tokensPath, []byte(content), FilePermSecure); writeErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not create tokens.env: %v\n", writeErr)
			} else {
				fmt.Fprintf(w, "  [FIX ] Created tokens.env with %d token(s)\n", len(skill.Registry.Tokens))
			}
		}
	}

	// Generate config.yaml if there are config declarations and the file is missing.
	if skill.Registry != nil && len(skill.Registry.Config) > 0 {
		configPath := filepath.Join(regDir, ConfigYAMLFile)
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			content := generateConfigTemplate(skill.Registry.Config)
			if writeErr := os.WriteFile(configPath, []byte(content), 0644); writeErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not create config.yaml: %v\n", writeErr)
			} else {
				fmt.Fprintf(w, "  [FIX ] Created config.yaml with %d key(s)\n", len(skill.Registry.Config))
			}
		}
	}

	// Fix tokens.env permissions if they are wrong.
	tokensPath := filepath.Join(regDir, TokensEnvFile)
	if info, err := os.Stat(tokensPath); err == nil {
		if info.Mode().Perm() != FilePermSecure {
			if chErr := platform.Chmod(tokensPath, FilePermSecure); chErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not fix tokens.env permissions: %v\n", chErr)
			} else {
				fmt.Fprintf(w, "  [FIX ] Fixed tokens.env permissions to %04o\n", FilePermSecure)
			}
		}
	}
}

// generateTokensEnvTemplate creates a templated tokens.env file content from
// the manifest's token declarations.
func generateTokensEnvTemplate(tokens []manifest.RegistryToken) string {
	var b strings.Builder
	b.WriteString("# Auto-generated by agentx doctor --fix\n")
	b.WriteString("# Fill in the values for required tokens.\n\n")
	for _, t := range tokens {
		if t.Description != "" {
			b.WriteString("# " + t.Description + "\n")
		}
		if t.Required {
			b.WriteString("# (required)\n")
		}
		if t.Default != "" {
			b.WriteString(t.Name + "=" + t.Default + "\n")
		} else {
			b.WriteString(t.Name + "=\n")
		}
	}
	return b.String()
}

// generateConfigTemplate creates a config.yaml file content from the manifest's
// config default values.
func generateConfigTemplate(config map[string]interface{}) string {
	var b strings.Builder
	b.WriteString("# Auto-generated by agentx doctor --fix\n")
	for key, val := range config {
		b.WriteString(fmt.Sprintf("%s: %v\n", key, val))
	}
	return b.String()
}

// CheckCLIDeps verifies that CLI dependencies declared in installed skill
// manifests are available on the system PATH.
func CheckCLIDeps(w io.Writer, installedRoot string) error {
	fmt.Fprintln(w, "CLI dependency check:")

	skillsInstalled := filepath.Join(installedRoot, "skills")
	if _, err := os.Stat(skillsInstalled); os.IsNotExist(err) {
		fmt.Fprintln(w, "  [MISS] No installed skills directory found")
		return nil
	}

	skills, err := discoverInstalledSkills(skillsInstalled)
	if err != nil {
		return fmt.Errorf("discovering installed skills: %w", err)
	}

	if len(skills) == 0 {
		fmt.Fprintln(w, "  [ OK ] No installed skills to check")
		return nil
	}

	missingCount := 0
	checkedCLIs := make(map[string]bool) // avoid duplicate checks

	for _, sp := range skills {
		parsed, err := manifest.ParseFile(sp.manifestPath)
		if err != nil {
			continue
		}
		skill, ok := parsed.(*manifest.SkillManifest)
		if !ok {
			continue
		}
		for _, dep := range skill.CLIDependencies {
			if _, seen := checkedCLIs[dep.Name]; seen {
				continue
			}
			path, lookErr := exec.LookPath(dep.Name)
			if lookErr != nil {
				fmt.Fprintf(w, "  [MISS] %s (required by %s)\n", dep.Name, sp.typePath)
				checkedCLIs[dep.Name] = false
				missingCount++
			} else {
				fmt.Fprintf(w, "  [ OK ] %s found at %s\n", dep.Name, path)
				checkedCLIs[dep.Name] = true
			}
		}
	}

	if missingCount > 0 {
		fmt.Fprintf(w, "\n  %d missing CLI dependency(ies). Run `agentx doctor --fix` to install.\n", missingCount)
	} else if len(checkedCLIs) == 0 {
		fmt.Fprintln(w, "  [ OK ] No CLI dependencies declared")
	} else {
		fmt.Fprintf(w, "  [ OK ] All %d CLI dependencies found\n", len(checkedCLIs))
	}

	return nil
}

// CheckLinks verifies that symlinks created by `agentx link` are intact.
// It reads .agentx/project.yaml from projectPath and checks each tool's
// symlink status.
func CheckLinks(w io.Writer, projectPath string) error {
	fmt.Fprintln(w, "Link check:")

	projectYAML := filepath.Join(projectPath, ".agentx", "project.yaml")
	if _, err := os.Stat(projectYAML); os.IsNotExist(err) {
		fmt.Fprintln(w, "  [MISS] No .agentx/project.yaml found (not a linked project)")
		return nil
	}

	data, err := os.ReadFile(projectYAML)
	if err != nil {
		return fmt.Errorf("reading project.yaml: %w", err)
	}

	var config struct {
		Tools  []string `yaml:"tools"`
		Active struct {
			Personas  []string `yaml:"personas,omitempty"`
			Context   []string `yaml:"context,omitempty"`
			Skills    []string `yaml:"skills,omitempty"`
			Workflows []string `yaml:"workflows,omitempty"`
			Prompts   []string `yaml:"prompts,omitempty"`
		} `yaml:"active"`
	}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("parsing project.yaml: %w", err)
	}

	// Check for each tool directory.
	brokenCount := 0
	for _, tool := range config.Tools {
		toolDir := toolConfigDir(projectPath, tool)
		if toolDir == "" {
			continue
		}

		entries, err := os.ReadDir(toolDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(w, "  [MISS] %s config directory not found\n", tool)
				brokenCount++
			}
			continue
		}

		symlinks := 0
		broken := 0
		for _, e := range entries {
			path := filepath.Join(toolDir, e.Name())
			target, err := platform.ReadSymlinkTarget(path)
			if err != nil {
				continue // not a symlink
			}
			symlinks++
			// Resolve relative target.
			resolvedTarget := target
			if !filepath.IsAbs(target) {
				resolvedTarget = filepath.Join(toolDir, target)
			}
			if _, statErr := os.Stat(resolvedTarget); statErr != nil {
				fmt.Fprintf(w, "  [FAIL] %s -> %s (broken)\n", path, target)
				broken++
				brokenCount++
			}
		}

		if broken == 0 && symlinks > 0 {
			fmt.Fprintf(w, "  [ OK ] %s: %d symlink(s) valid\n", tool, symlinks)
		} else if symlinks == 0 {
			fmt.Fprintf(w, "  [INFO] %s: no symlinks found\n", tool)
		}
	}

	if brokenCount > 0 {
		fmt.Fprintf(w, "\n  %d broken link(s) found. Run `agentx link sync` to repair.\n", brokenCount)
	}

	return nil
}

// toolConfigDir returns the expected configuration directory for an AI tool.
func toolConfigDir(projectPath, tool string) string {
	switch tool {
	case "claude-code":
		return filepath.Join(projectPath, ".claude")
	case "copilot":
		return filepath.Join(projectPath, ".github")
	case "augment":
		return filepath.Join(projectPath, ".augment")
	default:
		return ""
	}
}

// SkillRegistryStatus returns a structured result for a single skill's registry.
// This is useful for callers that want to build custom output or aggregate results.
func SkillRegistryStatus(skillPath, installedRoot, userdataRoot string) (*RegistryCheckResult, error) {
	result := &RegistryCheckResult{
		SkillPath: skillPath,
	}

	regDir := filepath.Join(userdataRoot, SkillsDir, skillPath)
	if _, err := os.Stat(regDir); os.IsNotExist(err) {
		result.FolderExists = false
		return result, nil
	}
	result.FolderExists = true

	// Find and parse manifest.
	manifestPath, err := findSkillManifest(installedRoot, skillPath)
	if err != nil {
		result.ParseError = err.Error()
		return result, nil
	}

	parsed, err := manifest.ParseFile(manifestPath)
	if err != nil {
		result.ParseError = err.Error()
		return result, nil
	}

	skill, ok := parsed.(*manifest.SkillManifest)
	if !ok {
		result.ParseError = "manifest is not a skill type"
		return result, nil
	}

	// Check tokens.
	if skill.Registry != nil && len(skill.Registry.Tokens) > 0 {
		tokensPath := filepath.Join(regDir, TokensEnvFile)
		entries, err := ParseEnvFile(tokensPath)
		values := make(map[string]string)
		if err == nil {
			for _, e := range entries {
				values[e.Key] = e.Value
			}
		}
		for _, t := range skill.Registry.Tokens {
			if t.Required {
				v, exists := values[t.Name]
				if !exists || v == "" {
					result.MissingTokens = append(result.MissingTokens, t.Name)
				}
			}
		}
	}

	// Check config.
	if skill.Registry != nil && len(skill.Registry.Config) > 0 {
		configPath := filepath.Join(regDir, ConfigYAMLFile)
		data, err := os.ReadFile(configPath)
		if err != nil {
			result.ConfigMissing = true
		} else {
			var m map[string]interface{}
			if yamlErr := yaml.Unmarshal(data, &m); yamlErr == nil {
				result.ConfigKeyCount = len(m)
			}
		}
	}

	// Check permissions.
	tokensPath := filepath.Join(regDir, TokensEnvFile)
	if info, err := os.Stat(tokensPath); err == nil {
		result.ActualPerm = info.Mode().Perm()
		result.PermissionsOK = result.ActualPerm == FilePermSecure
	} else {
		result.PermissionsOK = true // no file to check
	}

	return result, nil
}
