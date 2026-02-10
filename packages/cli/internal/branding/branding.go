// Package branding provides compile-time identity values for the CLI.
//
// Forkers edit branding.yaml at the repo root, then run `make build`.
// The Makefile syncs branding.yaml into this package before compilation,
// and Go's //go:embed bakes it into the binary.
package branding

import (
	_ "embed"
	"strings"
	"sync"

	"go.yaml.in/yaml/v3"
)

//go:embed branding.yaml
var rawBranding []byte

// B holds the parsed branding values, loaded once on first access.
var (
	once     sync.Once
	defaults brand
)

type brand struct {
	CLIName        string `yaml:"cli_name"`
	DisplayName    string `yaml:"display_name"`
	Description    string `yaml:"description"`
	HomeDir        string `yaml:"home_dir"`
	EnvPrefix      string `yaml:"env_prefix"`
	GoModule       string `yaml:"go_module"`
	GitHubRepo     string `yaml:"github_repo"`
	CatalogRepoURL string `yaml:"catalog_repo_url"`
}

func load() {
	once.Do(func() {
		// Set hard defaults in case the embedded file is missing/empty.
		defaults = brand{
			CLIName:        "agentx",
			DisplayName:    "AgentX",
			Description:    "Supply chain manager for AI agent configurations",
			HomeDir:        ".agentx",
			EnvPrefix:      "AGENTX",
			GoModule:       "github.com/agentx-labs/agentx",
			GitHubRepo:     "ecruz165/agentx",
			CatalogRepoURL: "https://github.com/ecruz165/agentx.git",
		}
		// Overlay with embedded YAML values.
		_ = yaml.Unmarshal(rawBranding, &defaults)
	})
}

// CLIName returns the root command name (e.g., "agentx").
func CLIName() string { load(); return defaults.CLIName }

// DisplayName returns the human-readable product name (e.g., "AgentX").
func DisplayName() string { load(); return defaults.DisplayName }

// Description returns the short product description.
func Description() string { load(); return defaults.Description }

// HomeDir returns the dot-directory name under $HOME (e.g., ".agentx").
func HomeDir() string { load(); return defaults.HomeDir }

// EnvPrefix returns the environment variable prefix (e.g., "AGENTX").
func EnvPrefix() string { load(); return defaults.EnvPrefix }

// GoModule returns the Go module path (e.g., "github.com/agentx-labs/agentx").
// Used by scripts/rebrand.sh — not consumed at runtime.
func GoModule() string { load(); return defaults.GoModule }

// GitHubRepo returns the "owner/repo" string (e.g., "ecruz165/agentx").
func GitHubRepo() string { load(); return defaults.GitHubRepo }

// CatalogRepoURL returns the default git URL for catalog cloning.
func CatalogRepoURL() string { load(); return defaults.CatalogRepoURL }

// EnvVar returns a fully qualified env var name, e.g., EnvVar("HOME") → "AGENTX_HOME".
func EnvVar(suffix string) string {
	load()
	return defaults.EnvPrefix + "_" + strings.ToUpper(suffix)
}
