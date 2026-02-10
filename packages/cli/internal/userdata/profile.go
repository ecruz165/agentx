package userdata

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/platform"
	"go.yaml.in/yaml/v3"
)

// Profile represents a named user configuration profile.
type Profile struct {
	Name          string `yaml:"name"`
	AwsProfile    string `yaml:"aws_profile,omitempty"`
	AwsRegion     string `yaml:"aws_region,omitempty"`
	GithubOrg     string `yaml:"github_org,omitempty"`
	SplunkHost    string `yaml:"splunk_host,omitempty"`
	DefaultBranch string `yaml:"default_branch,omitempty"`

	// Extras holds arbitrary user-defined fields.
	Extras map[string]interface{} `yaml:",inline"`
}

// LoadProfile reads the active profile by following the active symlink.
func LoadProfile() (*Profile, error) {
	profilesDir, err := GetProfilesDir()
	if err != nil {
		return nil, err
	}
	activePath := filepath.Join(profilesDir, ActiveProfileLink)

	data, err := os.ReadFile(activePath)
	if err != nil {
		return nil, fmt.Errorf("reading active profile: %w", err)
	}

	var p Profile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parsing active profile: %w", err)
	}
	return &p, nil
}

// ListProfiles returns the names of all profiles in the profiles/ directory.
// Names are derived from .yaml filenames (without extension), excluding the
// active symlink.
func ListProfiles() ([]string, error) {
	profilesDir, err := GetProfilesDir()
	if err != nil {
		return nil, err
	}

	entries, err := os.ReadDir(profilesDir)
	if err != nil {
		return nil, fmt.Errorf("reading profiles directory: %w", err)
	}

	var names []string
	for _, e := range entries {
		name := e.Name()
		if name == ActiveProfileLink {
			continue
		}
		if strings.HasSuffix(name, ".yaml") {
			names = append(names, strings.TrimSuffix(name, ".yaml"))
		}
	}
	return names, nil
}

// ActiveProfileName returns the name of the currently active profile
// by reading the symlink target.
func ActiveProfileName() (string, error) {
	profilesDir, err := GetProfilesDir()
	if err != nil {
		return "", err
	}
	activePath := filepath.Join(profilesDir, ActiveProfileLink)

	target, err := platform.ReadSymlinkTarget(activePath)
	if err != nil {
		return "", fmt.Errorf("reading active profile symlink: %w", err)
	}
	return strings.TrimSuffix(filepath.Base(target), ".yaml"), nil
}

// SwitchProfile updates the active symlink to point to the named profile.
// It uses relative symlink targets for portability.
func SwitchProfile(name string) error {
	profilesDir, err := GetProfilesDir()
	if err != nil {
		return err
	}

	profileFile := name + ".yaml"
	profilePath := filepath.Join(profilesDir, profileFile)

	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return fmt.Errorf("profile %q not found. Use 'agentx profile list' to see available profiles", name)
	} else if err != nil {
		return fmt.Errorf("checking profile %q: %w", name, err)
	}

	activePath := filepath.Join(profilesDir, ActiveProfileLink)

	// Remove existing symlink (ignore error if it doesn't exist).
	platform.RemoveSymlink(activePath)

	if err := platform.CreateSymlink(profileFile, activePath); err != nil {
		return fmt.Errorf("creating active symlink: %w", err)
	}
	return nil
}
