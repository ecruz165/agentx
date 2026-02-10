// Package cli defines the Cobra command tree for the agentx CLI. Each file
// in this package registers one top-level command (install, run, link, etc.)
// with the root command. Command implementations delegate to internal packages
// for business logic and only handle flag parsing, I/O formatting, and user interaction.
package cli