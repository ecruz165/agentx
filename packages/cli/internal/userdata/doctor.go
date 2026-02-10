package userdata

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/platform"
)

// CheckUserdata validates the userdata directory structure and permissions.
// When fix is true, it attempts to repair issues.
func CheckUserdata(w io.Writer, fix bool) error {
	root, err := GetUserdataRoot()
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "Userdata check:")

	// Check root exists.
	if _, statErr := os.Stat(root); os.IsNotExist(statErr) {
		fmt.Fprintf(w, "  [MISS] %s does not exist\n", root)
		if fix {
			fmt.Fprintln(w, "  [FIX ] Running init --global...")
			if initErr := InitGlobal(w); initErr != nil {
				return fmt.Errorf("auto-fix init: %w", initErr)
			}
		} else {
			fmt.Fprintln(w, "         Run 'agentx init --global' to create")
		}
		return nil
	}
	fmt.Fprintf(w, "  [ OK ] %s exists\n", root)

	// Check env/ directory.
	envDir := filepath.Join(root, EnvDir)
	checkDirWithPerm(w, envDir, DirPermSecure, fix)

	// Check profiles/ directory.
	profilesDir := filepath.Join(root, ProfilesDir)
	checkDirWithPerm(w, profilesDir, DirPermSecure, fix)

	// Check active symlink.
	activePath := filepath.Join(profilesDir, ActiveProfileLink)
	checkActiveSymlink(w, activePath, profilesDir)

	// Check preferences.yaml.
	prefsPath := filepath.Join(root, PreferencesFile)
	checkFileExists(w, prefsPath, fix)

	// Check skills/ directory.
	skillsDir := filepath.Join(root, SkillsDir)
	checkDirExists(w, skillsDir, fix)

	// Check env file permissions.
	checkEnvFilePerms(w, envDir, fix)

	return nil
}

func checkDirWithPerm(w io.Writer, path string, expectedPerm os.FileMode, fix bool) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(w, "  [MISS] %s does not exist\n", path)
		if fix {
			if mkErr := os.MkdirAll(path, expectedPerm); mkErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not create %s: %v\n", path, mkErr)
				return
			}
			platform.Chmod(path, expectedPerm)
			fmt.Fprintf(w, "  [FIX ] Created %s with %o\n", path, expectedPerm)
		}
		return
	}
	if err != nil {
		fmt.Fprintf(w, "  [FAIL] %s: %v\n", path, err)
		return
	}

	actualPerm := info.Mode().Perm()
	if actualPerm != expectedPerm {
		fmt.Fprintf(w, "  [WARN] %s has permissions %o (expected %o)\n", path, actualPerm, expectedPerm)
		if fix {
			if chErr := platform.Chmod(path, expectedPerm); chErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not fix permissions on %s: %v\n", path, chErr)
				return
			}
			fmt.Fprintf(w, "  [FIX ] Fixed permissions on %s to %o\n", path, expectedPerm)
		}
		return
	}
	fmt.Fprintf(w, "  [ OK ] %s (permissions %o)\n", path, actualPerm)
}

func checkActiveSymlink(w io.Writer, activePath, profilesDir string) {
	target, err := platform.ReadSymlinkTarget(activePath)
	if err != nil {
		fmt.Fprintf(w, "  [MISS] %s symlink not found or invalid\n", activePath)
		return
	}

	// Resolve relative target.
	resolvedTarget := target
	if !filepath.IsAbs(target) {
		resolvedTarget = filepath.Join(profilesDir, target)
	}

	if _, err := os.Stat(resolvedTarget); os.IsNotExist(err) {
		fmt.Fprintf(w, "  [WARN] %s -> %s (target does not exist)\n", activePath, target)
		return
	}
	fmt.Fprintf(w, "  [ OK ] %s -> %s\n", activePath, target)
}

func checkFileExists(w io.Writer, path string, fix bool) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Fprintf(w, "  [MISS] %s does not exist\n", path)
		return
	}
	fmt.Fprintf(w, "  [ OK ] %s exists\n", path)
}

func checkDirExists(w io.Writer, path string, fix bool) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		fmt.Fprintf(w, "  [MISS] %s does not exist\n", path)
		if fix {
			if mkErr := os.MkdirAll(path, DirPermNormal); mkErr != nil {
				fmt.Fprintf(w, "  [FAIL] Could not create %s: %v\n", path, mkErr)
				return
			}
			fmt.Fprintf(w, "  [FIX ] Created %s\n", path)
		}
		return
	}
	if err != nil {
		fmt.Fprintf(w, "  [FAIL] %s: %v\n", path, err)
		return
	}
	if !info.IsDir() {
		fmt.Fprintf(w, "  [WARN] %s exists but is not a directory\n", path)
		return
	}
	fmt.Fprintf(w, "  [ OK ] %s exists\n", path)
}

func checkEnvFilePerms(w io.Writer, envDir string, fix bool) {
	entries, err := os.ReadDir(envDir)
	if err != nil {
		return // env dir may not exist, already reported
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".env") {
			continue
		}
		path := filepath.Join(envDir, e.Name())
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		perm := info.Mode().Perm()
		if perm != FilePermSecure {
			fmt.Fprintf(w, "  [WARN] %s has permissions %o (expected %o)\n", path, perm, FilePermSecure)
			if fix {
				if chErr := platform.Chmod(path, FilePermSecure); chErr != nil {
					fmt.Fprintf(w, "  [FAIL] Could not fix permissions on %s: %v\n", path, chErr)
					continue
				}
				fmt.Fprintf(w, "  [FIX ] Fixed permissions on %s to %o\n", path, FilePermSecure)
			}
		}
	}
}
