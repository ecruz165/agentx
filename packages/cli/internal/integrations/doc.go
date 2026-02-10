// Package integrations dispatches AI tool configuration generation to per-tool
// CLI packages (claudecode-cli, copilot-cli, augment-cli, opencode-cli). It defines the
// GenerateResult and StatusResult types and routes link sync and status requests
// to the appropriate tool generator based on the project configuration.
package integrations