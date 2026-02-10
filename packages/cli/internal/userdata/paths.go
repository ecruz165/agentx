package userdata

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/branding"
)

// Directory and file name constants for the userdata convention.
const (
	UserdataDir        = "userdata"
	InstalledDir       = "installed"
	EnvDir             = "env"
	ProfilesDir        = "profiles"
	SkillsDir          = "skills"
	PreferencesFile    = "preferences.yaml"
	DefaultEnvFile     = "default.env"
	ActiveProfileLink  = "active"
	DefaultProfileFile = "default.yaml"

	// Catalog and extension directories for end-user mode.
	CatalogRepoDir = "catalog-repo"
	CatalogDir     = "catalog"
	ExtensionsDir  = "extensions"
)

// Permission constants.
const (
	DirPermSecure  os.FileMode = 0700
	FilePermSecure os.FileMode = 0600
	DirPermNormal  os.FileMode = 0755
)

// GetInstalledRoot returns the path to the installed types directory.
// It checks the AGENTX_INSTALLED environment variable first,
// then falls back to ~/.agentx/installed.
func GetInstalledRoot() (string, error) {
	if v := os.Getenv(branding.EnvVar("INSTALLED")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, branding.HomeDir(), InstalledDir), nil
}

// GetUserdataRoot returns the path to the userdata directory.
// It checks the AGENTX_USERDATA environment variable first,
// then falls back to ~/.agentx/userdata.
func GetUserdataRoot() (string, error) {
	if v := os.Getenv(branding.EnvVar("USERDATA")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, branding.HomeDir(), UserdataDir), nil
}

// GetEnvDir returns the path to the env/ directory within userdata.
func GetEnvDir() (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, EnvDir), nil
}

// GetProfilesDir returns the path to the profiles/ directory within userdata.
func GetProfilesDir() (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ProfilesDir), nil
}

// GetPreferencesPath returns the path to preferences.yaml within userdata.
func GetPreferencesPath() (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, PreferencesFile), nil
}

// GetSkillsDir returns the path to the skills/ directory within userdata.
func GetSkillsDir() (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, SkillsDir), nil
}

// GetVendorEnvPath returns the path to a vendor-specific .env file.
// For example, GetVendorEnvPath("aws") returns "<userdata>/env/aws.env".
func GetVendorEnvPath(vendor string) (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, EnvDir, vendor+".env"), nil
}

// GetSkillRegistryPath returns the path to a skill's registry directory.
// For example, GetSkillRegistryPath("cloud/aws/ssm-lookup") returns
// "<userdata>/skills/cloud/aws/ssm-lookup/".
func GetSkillRegistryPath(skillPath string) (string, error) {
	root, err := GetUserdataRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, SkillsDir, skillPath), nil
}

// GetCatalogRepoRoot returns the path to the catalog git repo directory.
// Checks AGENTX_CATALOG env override first, then falls back to ~/.agentx/catalog-repo/.
func GetCatalogRepoRoot() (string, error) {
	if v := os.Getenv(branding.EnvVar("CATALOG")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, branding.HomeDir(), CatalogRepoDir), nil
}

// GetCatalogRoot returns the path to the catalog/ subdirectory within the catalog repo.
// This is where type categories (skills/, workflows/, etc.) live.
func GetCatalogRoot() (string, error) {
	repoRoot, err := GetCatalogRepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(repoRoot, CatalogDir), nil
}

// GetExtensionsRoot returns the path to the user-local extensions directory.
// Checks AGENTX_EXTENSIONS env override first, then falls back to ~/.agentx/extensions/.
func GetExtensionsRoot() (string, error) {
	if v := os.Getenv(branding.EnvVar("EXTENSIONS")); v != "" {
		return v, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, branding.HomeDir(), ExtensionsDir), nil
}

// CatalogExists checks if the catalog directory has at least one category subdirectory.
func CatalogExists() (bool, error) {
	catalogRoot, err := GetCatalogRoot()
	if err != nil {
		return false, err
	}
	entries, err := os.ReadDir(catalogRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, fmt.Errorf("reading catalog directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return true, nil
		}
	}
	return false, nil
}
