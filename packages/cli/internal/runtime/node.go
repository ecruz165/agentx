package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/userdata"
)

// NodeRuntime executes Node.js-based skills.
type NodeRuntime struct {
	// Stdout and Stderr can be set for testing; defaults to os.Stdout/os.Stderr.
	Stdout io.Writer
	Stderr io.Writer
}

// Run executes a Node.js skill by invoking `node <skillPath>/index.mjs run <json-args>`.
// It sets AGENTX_USERDATA and AGENTX_SKILL_REGISTRY environment variables and
// streams stdout/stderr to the configured writers.
func (n *NodeRuntime) Run(ctx context.Context, skillPath string, m *manifest.SkillManifest, args map[string]string) (*Output, error) {
	// Verify Node.js is available.
	nodeBin, err := exec.LookPath("node")
	if err != nil {
		return nil, fmt.Errorf("node runtime requires Node.js: %w", err)
	}

	// Resolve the entry point.
	entryPoint := filepath.Join(skillPath, "index.mjs")
	if _, err := os.Stat(entryPoint); err != nil {
		return nil, fmt.Errorf("skill entry point not found at %s: %w", entryPoint, err)
	}

	// Serialize args to JSON.
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("serializing skill arguments: %w", err)
	}

	// Build environment variables.
	env, err := buildNodeEnv(skillPath, m)
	if err != nil {
		return nil, fmt.Errorf("building runtime environment: %w", err)
	}

	// Build the command.
	cmd := exec.CommandContext(ctx, nodeBin, entryPoint, "run", string(argsJSON))
	cmd.Dir = skillPath
	cmd.Env = env

	// Set up output capture while also streaming to configured writers.
	stdout := n.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := n.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = io.MultiWriter(stdout, &stdoutBuf)
	cmd.Stderr = io.MultiWriter(stderr, &stderrBuf)

	// Execute.
	err = cmd.Run()

	output := &Output{
		Stdout: stdoutBuf.String(),
		Stderr: stderrBuf.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			output.ExitCode = exitErr.ExitCode()
			return output, nil
		}
		return output, fmt.Errorf("executing node skill: %w", err)
	}

	output.ExitCode = 0
	return output, nil
}

// buildNodeEnv constructs the environment variables for a Node.js skill execution.
// It inherits the current process environment and adds AgentX-specific variables.
func buildNodeEnv(skillPath string, m *manifest.SkillManifest) ([]string, error) {
	env := os.Environ()

	// Set AGENTX_USERDATA.
	userdataRoot, err := userdata.GetUserdataRoot()
	if err != nil {
		return nil, fmt.Errorf("resolving userdata root: %w", err)
	}
	env = setEnv(env, "AGENTX_USERDATA", userdataRoot)

	// Set AGENTX_SKILL_REGISTRY to the skill's registry directory.
	// The registry path is derived from the skill name by stripping "skills/" prefix
	// if present, or using the skill name directly.
	registryName := m.Name
	if m.Topic != "" {
		registryName = m.Topic + "/" + m.Name
	}
	registryPath, err := userdata.GetSkillRegistryPath(registryName)
	if err != nil {
		return nil, fmt.Errorf("resolving skill registry path: %w", err)
	}
	env = setEnv(env, "AGENTX_SKILL_REGISTRY", registryPath)

	// Set AGENTX_SKILL_PATH for convenience.
	env = setEnv(env, "AGENTX_SKILL_PATH", skillPath)

	// Load tokens from the skill registry if tokens.env exists.
	tokensPath := filepath.Join(registryPath, "tokens.env")
	if data, err := os.ReadFile(tokensPath); err == nil {
		env = loadTokensEnv(env, data)
	}

	return env, nil
}

// setEnv sets or replaces an environment variable in the env slice.
func setEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, e := range env {
		if strings.HasPrefix(e, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// loadTokensEnv reads a tokens.env file and adds non-empty, non-comment lines
// to the environment slice.
func loadTokensEnv(env []string, data []byte) []string {
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key != "" && value != "" {
			env = setEnv(env, key, value)
		}
	}
	return env
}
