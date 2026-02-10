package runtime

import (
	"context"
	"fmt"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// GoRuntime executes Go-based skills.
// This is a placeholder implementation — no Go skills exist yet.
type GoRuntime struct{}

// Run returns an error indicating the Go runtime is not yet supported.
// When Go skills are added, this will invoke compiled binaries from
// ~/.agentx/bin/<skill-name> with the same environment variables as NodeRuntime.
func (g *GoRuntime) Run(_ context.Context, _ string, _ *manifest.SkillManifest, _ map[string]string) (*Output, error) {
	return nil, fmt.Errorf("go runtime is not yet supported: no Go skills currently exist")
}
