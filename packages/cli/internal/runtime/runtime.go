package runtime

import (
	"context"
	"fmt"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// Runtime defines the interface for executing a skill.
type Runtime interface {
	// Run executes a skill at the given path with the provided manifest and arguments.
	Run(ctx context.Context, skillPath string, manifest *manifest.SkillManifest, args map[string]string) (*Output, error)
}

// Output captures the result of a skill execution.
type Output struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Supported runtime identifiers.
const (
	RuntimeNode = "node"
	RuntimeGo   = "go"
)

// DispatchRuntime returns the appropriate Runtime implementation for the given
// runtime identifier. Returns an error-producing runtime for unknown values.
func DispatchRuntime(runtime string) Runtime {
	switch runtime {
	case RuntimeNode:
		return &NodeRuntime{}
	case RuntimeGo:
		return &GoRuntime{}
	default:
		return &unknownRuntime{name: runtime}
	}
}

// unknownRuntime is returned when the runtime identifier is not recognized.
type unknownRuntime struct {
	name string
}

func (u *unknownRuntime) Run(_ context.Context, _ string, _ *manifest.SkillManifest, _ map[string]string) (*Output, error) {
	return nil, fmt.Errorf("unknown runtime %q: supported runtimes are %q and %q", u.name, RuntimeNode, RuntimeGo)
}
