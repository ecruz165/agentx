package userdata

import (
	"os"

	"github.com/agentx-labs/agentx/internal/branding"
)

// Mode represents the operating mode of the CLI.
type Mode int

const (
	// ModeEndUser is for developers who installed the CLI without cloning the repo.
	// Catalog lives at ~/.agentx/catalog-repo/catalog/ and extensions are user-local.
	ModeEndUser Mode = iota
	// ModePlatformTeam is for developers working within the AgentX repository.
	// AGENTX_HOME is set; catalog and extensions live inside the repo.
	ModePlatformTeam
)

// DetectMode returns the current operating mode.
// If AGENTX_HOME is set, the CLI is in platform-team mode.
// Otherwise, it's in end-user mode.
func DetectMode() Mode {
	if os.Getenv(branding.EnvVar("HOME")) != "" {
		return ModePlatformTeam
	}
	return ModeEndUser
}

// String returns a human-readable name for the mode.
func (m Mode) String() string {
	switch m {
	case ModePlatformTeam:
		return "platform-team"
	case ModeEndUser:
		return "end-user"
	default:
		return "unknown"
	}
}
