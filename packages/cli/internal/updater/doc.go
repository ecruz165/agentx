// Package updater implements the self-update mechanism for the agentx binary.
// It checks GitHub Releases (or a configured Nexus mirror) for new versions,
// downloads and verifies checksums, extracts the binary, and replaces the
// running executable. A daily-cached version check powers the startup banner.
package updater