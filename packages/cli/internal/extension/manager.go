package extension

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// ExtensionStatus represents the status of a single extension.
type ExtensionStatus struct {
	Name   string
	Path   string
	Source string
	Branch string
	Status string // "ok", "uninitialized", "modified", "missing"
}

// Add adds a new extension as a git submodule, updates project.yaml,
// and adds a .gitignore exclusion line.
func Add(repoRoot, name, gitURL, branch string) error {
	if branch == "" {
		branch = "main"
	}

	configPath := filepath.Join(repoRoot, ProjectConfigFile)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Check if extension already exists.
	if cfg.FindExtension(name) != nil {
		return fmt.Errorf("extension %q already exists", name)
	}

	submodulePath := filepath.Join("extensions", name)

	// Run git submodule add.
	cmd := exec.Command("git", "submodule", "add", "-b", branch, gitURL, submodulePath)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git submodule add failed: %w\n%s", err, string(output))
	}

	// Update project.yaml.
	ext := Extension{
		Name:   name,
		Path:   submodulePath,
		Source: gitURL,
		Branch: branch,
	}
	if err := cfg.AddExtension(ext); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}
	if err := SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Update .gitignore.
	if err := AddToGitignore(repoRoot, name); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	return nil
}

// Remove removes an extension: deinitializes the submodule, removes it from
// git, updates project.yaml, and cleans up .gitignore.
func Remove(repoRoot, name string) error {
	configPath := filepath.Join(repoRoot, ProjectConfigFile)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	if cfg.FindExtension(name) == nil {
		return fmt.Errorf("extension %q not found in configuration", name)
	}

	submodulePath := filepath.Join("extensions", name)

	// Step 1: git submodule deinit -f
	cmd := exec.Command("git", "submodule", "deinit", "-f", submodulePath)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git submodule deinit failed: %w\n%s", err, string(output))
	}

	// Step 2: git rm -f
	cmd = exec.Command("git", "rm", "-f", submodulePath)
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git rm failed: %w\n%s", err, string(output))
	}

	// Step 3: Clean up .git/modules/<path>
	modulesPath := filepath.Join(repoRoot, ".git", "modules", submodulePath)
	cmd = exec.Command("rm", "-rf", modulesPath)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cleaning .git/modules failed: %w\n%s", err, string(output))
	}

	// Update project.yaml.
	if err := cfg.RemoveExtension(name); err != nil {
		return fmt.Errorf("updating config: %w", err)
	}
	if err := SaveConfig(configPath, cfg); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Update .gitignore.
	if err := RemoveFromGitignore(repoRoot, name); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}

	return nil
}

// List returns the status of all declared extensions.
func List(repoRoot string) ([]ExtensionStatus, error) {
	configPath := filepath.Join(repoRoot, ProjectConfigFile)
	cfg, err := LoadConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	// Get git submodule status.
	statusMap, err := submoduleStatusMap(repoRoot)
	if err != nil {
		// Non-fatal: we can still list extensions, just without git status.
		statusMap = make(map[string]string)
	}

	var result []ExtensionStatus
	for _, ext := range cfg.Extensions {
		submodulePath := ext.Path
		if submodulePath == "" {
			submodulePath = filepath.Join("extensions", ext.Name)
		}

		status := "ok"
		if gitStatus, ok := statusMap[submodulePath]; ok {
			status = gitStatus
		} else {
			status = "missing"
		}

		result = append(result, ExtensionStatus{
			Name:   ext.Name,
			Path:   submodulePath,
			Source: ext.Source,
			Branch: ext.Branch,
			Status: status,
		})
	}

	return result, nil
}

// Sync runs git submodule update --init --recursive to initialize and
// update all submodules.
func Sync(repoRoot string) error {
	cmd := exec.Command("git", "submodule", "update", "--init", "--recursive")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git submodule update failed: %w\n%s", err, string(output))
	}
	return nil
}

// submoduleStatusMap runs `git submodule status` and returns a map of
// submodule path -> status string. Status values:
//   - "ok" — submodule is initialized and clean
//   - "uninitialized" — submodule is not initialized (prefix -)
//   - "modified" — submodule has local modifications (prefix +)
func submoduleStatusMap(repoRoot string) (map[string]string, error) {
	cmd := exec.Command("git", "submodule", "status")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("git submodule status failed: %w\n%s", err, string(output))
	}

	result := make(map[string]string)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: " <sha> <path> (<desc>)" or "-<sha> <path>" or "+<sha> <path> (<desc>)"
		status := "ok"
		switch {
		case strings.HasPrefix(line, "-"):
			status = "uninitialized"
			line = line[1:]
		case strings.HasPrefix(line, "+"):
			status = "modified"
			line = line[1:]
		}

		// Split remaining: "<sha> <path> ..."
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			result[parts[1]] = status
		}
	}

	return result, nil
}
