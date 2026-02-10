package integrations

// ToolName identifies a supported AI tool integration.
type ToolName string

const (
	ClaudeCode ToolName = "claude-code"
	Copilot    ToolName = "copilot"
	Augment    ToolName = "augment"
	OpenCode   ToolName = "opencode"
)

// ToolConfig maps a tool to its Node package directory and script names.
type ToolConfig struct {
	PackageDir     string
	GenerateScript string
	StatusScript   string
}

// AllTools returns all supported tool names.
func AllTools() []ToolName {
	return []ToolName{ClaudeCode, Copilot, Augment, OpenCode}
}

// toolRegistry maps each tool to its Node package location and scripts.
var toolRegistry = map[ToolName]ToolConfig{
	ClaudeCode: {
		PackageDir:     "packages/claudecode-cli",
		GenerateScript: "bin/generate.mjs",
		StatusScript:   "bin/status.mjs",
	},
	Copilot: {
		PackageDir:     "packages/copilot-cli",
		GenerateScript: "bin/generate.mjs",
		StatusScript:   "bin/status.mjs",
	},
	Augment: {
		PackageDir:     "packages/augment-cli",
		GenerateScript: "bin/generate.mjs",
		StatusScript:   "bin/status.mjs",
	},
	OpenCode: {
		PackageDir:     "packages/opencode-cli",
		GenerateScript: "bin/generate.mjs",
		StatusScript:   "bin/status.mjs",
	},
}

// ParseToolName converts a string to a ToolName, returning false if invalid.
func ParseToolName(s string) (ToolName, bool) {
	switch s {
	case "claude-code":
		return ClaudeCode, true
	case "copilot":
		return Copilot, true
	case "augment":
		return Augment, true
	case "opencode":
		return OpenCode, true
	default:
		return "", false
	}
}
