package userdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/platform"
)

// Default content for env/default.env.
const defaultEnvContent = `# Shared environment variables loaded by all skills.
LOG_LEVEL=info
OUTPUT_FORMAT=json
`

// Default content for profiles/default.yaml.
const defaultProfileContent = `name: default
# aws_profile: my-profile
# aws_region: us-east-1
# github_org: my-org
# splunk_host: splunk.example.com
# default_branch: main
`

// Default content for preferences.yaml.
const defaultPreferencesContent = `output_format: json
color: true
verbose: false
# default_persona: senior-java-dev
# default_branch: main
# editor: vim
`

// InitGlobal creates the full userdata directory structure with proper permissions.
// It prints progress messages to w. Existing items are skipped with a message.
func InitGlobal(w io.Writer) error {
	root, err := GetUserdataRoot()
	if err != nil {
		return err
	}

	// Create root userdata directory.
	if err := ensureDir(w, root, DirPermNormal); err != nil {
		return err
	}

	// Create env/ directory with secure permissions.
	envDir := filepath.Join(root, EnvDir)
	if err := ensureDir(w, envDir, DirPermSecure); err != nil {
		return err
	}

	// Create env/default.env.
	defaultEnv := filepath.Join(envDir, DefaultEnvFile)
	if err := ensureFile(w, defaultEnv, defaultEnvContent, FilePermSecure); err != nil {
		return err
	}

	// Create profiles/ directory with secure permissions.
	profilesDir := filepath.Join(root, ProfilesDir)
	if err := ensureDir(w, profilesDir, DirPermSecure); err != nil {
		return err
	}

	// Create profiles/default.yaml.
	defaultProfile := filepath.Join(profilesDir, DefaultProfileFile)
	if err := ensureFile(w, defaultProfile, defaultProfileContent, DirPermNormal); err != nil {
		return err
	}

	// Create active symlink -> default.yaml (relative).
	activePath := filepath.Join(profilesDir, ActiveProfileLink)
	if err := ensureSymlink(w, activePath, DefaultProfileFile); err != nil {
		return err
	}

	// Create preferences.yaml.
	prefsPath := filepath.Join(root, PreferencesFile)
	if err := ensureFile(w, prefsPath, defaultPreferencesContent, DirPermNormal); err != nil {
		return err
	}

	// Create skills/ directory.
	skillsDir := filepath.Join(root, SkillsDir)
	if err := ensureDir(w, skillsDir, DirPermNormal); err != nil {
		return err
	}

	return nil
}

// EnsureSkillRegistry creates the registry directory for a skill at
// <userdata>/skills/<skillPath>/.
func EnsureSkillRegistry(skillPath string) error {
	p, err := GetSkillRegistryPath(skillPath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(p, DirPermNormal); err != nil {
		return fmt.Errorf("creating skill registry %s: %w", p, err)
	}
	return nil
}

// ensureDir creates a directory if it doesn't exist.
func ensureDir(w io.Writer, path string, perm os.FileMode) error {
	if info, err := os.Stat(path); err == nil {
		if info.IsDir() {
			fmt.Fprintf(w, "  [SKIP] %s already exists\n", path)
			return nil
		}
		return fmt.Errorf("%s exists but is not a directory", path)
	}

	if err := os.MkdirAll(path, perm); err != nil {
		return fmt.Errorf("creating directory %s: %w", path, err)
	}
	// MkdirAll may not apply exact perms if parent dirs needed creation.
	if err := platform.Chmod(path, perm); err != nil {
		return fmt.Errorf("setting permissions on %s: %w", path, err)
	}
	fmt.Fprintf(w, "  [ OK ] Created %s\n", path)
	return nil
}

// ensureFile creates a file with content if it doesn't exist.
func ensureFile(w io.Writer, path, content string, perm os.FileMode) error {
	if _, err := os.Stat(path); err == nil {
		fmt.Fprintf(w, "  [SKIP] %s already exists\n", path)
		return nil
	}

	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		return fmt.Errorf("creating file %s: %w", path, err)
	}
	fmt.Fprintf(w, "  [ OK ] Created %s\n", path)
	return nil
}

// ensureSymlink creates a symlink if it doesn't exist.
func ensureSymlink(w io.Writer, linkPath, target string) error {
	if _, err := os.Lstat(linkPath); err == nil {
		fmt.Fprintf(w, "  [SKIP] %s already exists\n", linkPath)
		return nil
	}

	if err := platform.CreateSymlink(target, linkPath); err != nil {
		return fmt.Errorf("creating symlink %s -> %s: %w", linkPath, target, err)
	}
	fmt.Fprintf(w, "  [ OK ] Created %s -> %s\n", linkPath, target)
	return nil
}
