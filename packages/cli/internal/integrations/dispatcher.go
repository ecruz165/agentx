package integrations

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/branding"
)

// GenerateResult mirrors the Node generate() return value.
type GenerateResult struct {
	Tool      ToolName `json:"tool"`
	Created   []string `json:"created"`
	Updated   []string `json:"updated"`
	Symlinked []string `json:"symlinked"`
	Warnings  []string `json:"warnings"`
}

// StatusResult mirrors the Node status() return value.
type StatusResult struct {
	Tool     string       `json:"tool"`
	Status   string       `json:"status"`
	Files    []string     `json:"files"`
	Symlinks SymlinkInfo  `json:"symlinks"`
}

// SymlinkInfo reports symlink health.
type SymlinkInfo struct {
	Total int `json:"total"`
	Valid int `json:"valid"`
}

// generateInput is the JSON struct sent to bin/generate.mjs via stdin.
type generateInput struct {
	ProjectConfig interface{} `json:"projectConfig"`
	InstalledPath string      `json:"installedPath"`
	ProjectPath   string      `json:"projectPath"`
}

// statusInput is the JSON struct sent to bin/status.mjs via stdin.
type statusInput struct {
	ProjectPath string `json:"projectPath"`
}

// GenerateConfigs calls each tool's Node generate script.
func GenerateConfigs(tools []ToolName, projectConfig interface{}, installedPath, projectPath string) ([]GenerateResult, error) {
	var results []GenerateResult

	input := generateInput{
		ProjectConfig: projectConfig,
		InstalledPath: installedPath,
		ProjectPath:   projectPath,
	}

	for _, tool := range tools {
		cfg, ok := toolRegistry[tool]
		if !ok {
			return nil, fmt.Errorf("unknown tool: %s", tool)
		}

		scriptPath, err := resolveScriptPath(tool, cfg.GenerateScript)
		if err != nil {
			return nil, fmt.Errorf("resolving %s generate script: %w", tool, err)
		}

		output, err := runNodeScript(scriptPath, input)
		if err != nil {
			return nil, fmt.Errorf("%s generate failed: %w", tool, err)
		}

		var result GenerateResult
		if err := json.Unmarshal(output, &result); err != nil {
			return nil, fmt.Errorf("parsing %s generate output: %w", tool, err)
		}
		result.Tool = tool
		results = append(results, result)
	}

	return results, nil
}

// GetStatus calls each tool's Node status script.
func GetStatus(tools []ToolName, projectPath string) ([]StatusResult, error) {
	var results []StatusResult

	input := statusInput{ProjectPath: projectPath}

	for _, tool := range tools {
		cfg, ok := toolRegistry[tool]
		if !ok {
			return nil, fmt.Errorf("unknown tool: %s", tool)
		}

		scriptPath, err := resolveScriptPath(tool, cfg.StatusScript)
		if err != nil {
			return nil, fmt.Errorf("resolving %s status script: %w", tool, err)
		}

		output, err := runNodeScript(scriptPath, input)
		if err != nil {
			return nil, fmt.Errorf("%s status failed: %w", tool, err)
		}

		var result StatusResult
		if err := json.Unmarshal(output, &result); err != nil {
			return nil, fmt.Errorf("parsing %s status output: %w", tool, err)
		}
		results = append(results, result)
	}

	return results, nil
}

// runNodeScript spawns node with the given script, writes JSON input to stdin,
// and returns the stdout bytes.
func runNodeScript(scriptPath string, input interface{}) ([]byte, error) {
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshaling input: %w", err)
	}

	cmd := exec.Command("node", scriptPath)
	cmd.Stdin = bytes.NewReader(inputJSON)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := stderr.String()
		if errMsg != "" {
			return nil, fmt.Errorf("%s: %s", err, errMsg)
		}
		return nil, err
	}

	return stdout.Bytes(), nil
}

// resolveScriptPath locates a tool's Node script.
// It checks AGENTX_HOME first (development/distribution), then falls back
// to a path relative to the running binary.
func resolveScriptPath(tool ToolName, script string) (string, error) {
	cfg := toolRegistry[tool]

	// Check <PREFIX>_HOME env var (development use)
	if home := os.Getenv(branding.EnvVar("HOME")); home != "" {
		path := filepath.Join(home, cfg.PackageDir, script)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	// Fallback: relative to the binary location
	exe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("locating executable: %w", err)
	}
	exeDir := filepath.Dir(exe)
	// In distribution, Node packages are at <binary-dir>/../lib/<package-dir>/
	path := filepath.Join(exeDir, "..", "lib", cfg.PackageDir, script)
	if _, err := os.Stat(path); err == nil {
		return path, nil
	}

	return "", fmt.Errorf("cannot find %s script for %s: set %s or ensure packages are installed", script, tool, branding.EnvVar("HOME"))
}
