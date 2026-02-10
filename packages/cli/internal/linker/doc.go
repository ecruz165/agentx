// Package linker manages the project-level .agentx/project.yaml configuration
// and orchestrates AI tool config generation. It adds and removes type references,
// triggers link sync to regenerate tool-specific files (CLAUDE.md,
// copilot-instructions.md, etc.), and reports link status across configured tools.
package linker