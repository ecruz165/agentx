package extension

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// UserAdd clones a git repository as a user-local extension into extRoot/<name>/.
func UserAdd(extRoot, name, gitURL, branch string) error {
	if branch == "" {
		branch = "main"
	}

	targetDir := filepath.Join(extRoot, name)

	// Check if extension directory already exists.
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("extension %q already exists at %s", name, targetDir)
	}

	if err := os.MkdirAll(extRoot, 0755); err != nil {
		return fmt.Errorf("creating extensions directory: %w", err)
	}

	cmd := exec.Command("git", "clone", "--depth=1", "-b", branch, gitURL, targetDir)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Clean up partial clone.
		_ = os.RemoveAll(targetDir)
		return fmt.Errorf("git clone failed: %w\n%s", err, strings.TrimSpace(string(output)))
	}

	return nil
}

// UserRemove removes a user-local extension directory.
func UserRemove(extRoot, name string) error {
	targetDir := filepath.Join(extRoot, name)

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		return fmt.Errorf("extension %q not found at %s", name, targetDir)
	}

	if err := os.RemoveAll(targetDir); err != nil {
		return fmt.Errorf("removing extension directory: %w", err)
	}

	return nil
}

// UserExtensionStatus represents the status of a user-local extension.
type UserExtensionStatus struct {
	Name   string
	Path   string
	Branch string
	Status string // "ok", "dirty", "error"
}

// UserList scans the extensions root and returns status for each extension directory.
func UserList(extRoot string) ([]UserExtensionStatus, error) {
	entries, err := os.ReadDir(extRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading extensions directory: %w", err)
	}

	var result []UserExtensionStatus
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		extPath := filepath.Join(extRoot, entry.Name())
		branch, status := userExtStatus(extPath)

		result = append(result, UserExtensionStatus{
			Name:   entry.Name(),
			Path:   extPath,
			Branch: branch,
			Status: status,
		})
	}

	return result, nil
}

// UserSync pulls latest changes for all user-local extensions.
func UserSync(extRoot string) error {
	entries, err := os.ReadDir(extRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading extensions directory: %w", err)
	}

	var errs []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		extPath := filepath.Join(extRoot, entry.Name())
		cmd := exec.Command("git", "pull", "--rebase")
		cmd.Dir = extPath
		if output, err := cmd.CombinedOutput(); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v (%s)", entry.Name(), err, strings.TrimSpace(string(output))))
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("some extensions failed to sync:\n  %s", strings.Join(errs, "\n  "))
	}

	return nil
}

// userExtStatus returns the branch name and dirty/clean status of a git repo.
func userExtStatus(repoPath string) (branch string, status string) {
	// Get current branch.
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = repoPath
	out, err := cmd.Output()
	if err != nil {
		return "unknown", "error"
	}
	branch = strings.TrimSpace(string(out))

	// Check for uncommitted changes.
	cmd = exec.Command("git", "status", "--porcelain")
	cmd.Dir = repoPath
	out, err = cmd.Output()
	if err != nil {
		return branch, "error"
	}

	if len(strings.TrimSpace(string(out))) > 0 {
		return branch, "dirty"
	}
	return branch, "ok"
}
