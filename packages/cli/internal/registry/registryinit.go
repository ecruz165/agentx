package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/userdata"
	"go.yaml.in/yaml/v3"
)

// InitSkillRegistry initializes the userdata registry for an installed skill.
// It creates the registry directory, generates tokens.env and config.yaml from
// the manifest's registry block, and creates declared subdirectories.
// Returns a list of warnings (e.g., required tokens without defaults).
func InitSkillRegistry(resolved *ResolvedType, installedRoot string) ([]string, error) {
	if resolved.Category != "skill" {
		return nil, nil
	}

	// Parse the manifest from the installed location.
	installedManifest := filepath.Join(installedRoot, resolved.TypePath, filepath.Base(resolved.ManifestPath))
	parsed, err := manifest.ParseFile(installedManifest)
	if err != nil {
		// Try original manifest path as fallback.
		parsed, err = manifest.ParseFile(resolved.ManifestPath)
		if err != nil {
			return nil, fmt.Errorf("parsing skill manifest: %w", err)
		}
	}

	skill, ok := parsed.(*manifest.SkillManifest)
	if !ok {
		return nil, fmt.Errorf("manifest at %s is not a skill", resolved.ManifestPath)
	}

	if skill.Registry == nil {
		return nil, nil // no registry declaration
	}

	// Strip "skills/" prefix for the registry path.
	registryPath := strings.TrimPrefix(resolved.TypePath, "skills/")
	if err := userdata.EnsureSkillRegistry(registryPath); err != nil {
		return nil, fmt.Errorf("creating skill registry: %w", err)
	}

	regDir, err := userdata.GetSkillRegistryPath(registryPath)
	if err != nil {
		return nil, err
	}

	var warnings []string

	// Generate tokens.env if tokens are declared.
	if len(skill.Registry.Tokens) > 0 {
		tokensPath := filepath.Join(regDir, "tokens.env")
		content := generateTokensEnv(skill.Registry.Tokens, skill.Name)
		if err := os.WriteFile(tokensPath, []byte(content), 0600); err != nil {
			return nil, fmt.Errorf("writing tokens.env: %w", err)
		}

		// Warn on required tokens without defaults.
		for _, t := range skill.Registry.Tokens {
			if t.Required && t.Default == "" {
				warnings = append(warnings, fmt.Sprintf("%s required — edit tokens.env", t.Name))
			}
		}
	}

	// Generate config.yaml if config defaults are declared.
	if len(skill.Registry.Config) > 0 {
		configPath := filepath.Join(regDir, "config.yaml")
		content, err := generateConfigYAML(skill.Registry.Config)
		if err != nil {
			return nil, fmt.Errorf("generating config.yaml: %w", err)
		}
		if err := os.WriteFile(configPath, content, 0644); err != nil {
			return nil, fmt.Errorf("writing config.yaml: %w", err)
		}
	}

	// Create subdirectories if declared.
	if len(skill.Registry.State) > 0 {
		if err := os.MkdirAll(filepath.Join(regDir, "state"), 0755); err != nil {
			return nil, fmt.Errorf("creating state dir: %w", err)
		}
	}
	if skill.Registry.Output != nil {
		if err := os.MkdirAll(filepath.Join(regDir, "output"), 0755); err != nil {
			return nil, fmt.Errorf("creating output dir: %w", err)
		}
	}
	if skill.Registry.Templates != nil {
		if err := os.MkdirAll(filepath.Join(regDir, "templates"), 0755); err != nil {
			return nil, fmt.Errorf("creating templates dir: %w", err)
		}
	}

	return warnings, nil
}

// generateTokensEnv generates a tokens.env file from registry token declarations.
func generateTokensEnv(tokens []manifest.RegistryToken, skillName string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# tokens.env — generated from %s manifest registry declaration\n", skillName)
	fmt.Fprintln(&b, "# Edit this file to configure skill tokens and secrets.")
	fmt.Fprintln(&b)

	for _, t := range tokens {
		if t.Description != "" {
			fmt.Fprintf(&b, "# %s\n", t.Description)
		}
		if t.Required {
			fmt.Fprintf(&b, "# (required)\n")
		}
		if t.Default != "" {
			fmt.Fprintf(&b, "%s=%s\n", t.Name, t.Default)
		} else {
			fmt.Fprintf(&b, "%s=\n", t.Name)
		}
		fmt.Fprintln(&b)
	}

	return b.String()
}

// generateConfigYAML generates a config.yaml from the registry config defaults.
func generateConfigYAML(config map[string]interface{}) ([]byte, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("marshaling config: %w", err)
	}

	var b strings.Builder
	fmt.Fprintln(&b, "# config.yaml — generated from manifest registry declaration")
	fmt.Fprintln(&b, "# Edit this file to customize skill configuration.")
	fmt.Fprintln(&b)
	b.Write(data)

	return []byte(b.String()), nil
}
