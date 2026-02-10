package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/extension"
	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	checkCLI        bool
	checkRuntime    bool
	checkLinks      bool
	checkExtensions bool
	checkUserdata   bool
	checkRegistry   bool
	checkManifest   string
	doctorFix       bool
	traceEnv        string
)

func init() {
	doctorCmd.Flags().BoolVar(&checkCLI, "check-cli", false, "Verify all CLI dependencies")
	doctorCmd.Flags().BoolVar(&checkRuntime, "check-runtime", false, "Verify Node/Go available")
	doctorCmd.Flags().BoolVar(&checkLinks, "check-links", false, "Verify symlinks intact")
	doctorCmd.Flags().BoolVar(&checkExtensions, "check-extensions", false, "Verify submodules initialized and synced")
	doctorCmd.Flags().BoolVar(&checkUserdata, "check-userdata", false, "Verify userdata directory")
	doctorCmd.Flags().BoolVar(&checkRegistry, "check-registry", false, "Validate skill registries")
	doctorCmd.Flags().StringVar(&checkManifest, "check-manifest", "", "Validate a manifest file at the given path")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Interactively install missing/outdated CLIs")
	doctorCmd.Flags().StringVar(&traceEnv, "trace-env", "", "Show env resolution order for a skill")
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Health check for AgentX installation",
	Long:  `Run diagnostic checks on your AgentX installation and environment.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		anyFlag := checkCLI || checkRuntime || checkLinks || checkExtensions ||
			checkUserdata || checkRegistry || checkManifest != "" || doctorFix || traceEnv != ""

		// If no specific flag, run all checks.
		if !anyFlag {
			runAllChecks()
			return nil
		}

		if checkCLI {
			installedRoot, err := userdata.GetInstalledRoot()
			if err != nil {
				return fmt.Errorf("resolving installed root: %w", err)
			}
			if err := userdata.CheckCLIDeps(os.Stdout, installedRoot); err != nil {
				return err
			}
		}
		if checkRuntime {
			runRuntimeCheck()
		}
		if checkLinks {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return fmt.Errorf("resolving project root: %w", err)
			}
			if err := userdata.CheckLinks(os.Stdout, repoRoot); err != nil {
				return err
			}
		}
		if checkExtensions {
			runExtensionsCheck()
		}
		if checkUserdata {
			userdata.CheckUserdata(os.Stdout, doctorFix)
		}
		if checkRegistry {
			installedRoot, err := userdata.GetInstalledRoot()
			if err != nil {
				return fmt.Errorf("resolving installed root: %w", err)
			}
			userdataRoot, err := userdata.GetUserdataRoot()
			if err != nil {
				return fmt.Errorf("resolving userdata root: %w", err)
			}
			if err := userdata.CheckRegistry(os.Stdout, installedRoot, userdataRoot); err != nil {
				return err
			}
		}
		if checkManifest != "" {
			if err := runManifestCheck(checkManifest); err != nil {
				return err
			}
		}
		if doctorFix {
			installedRoot, err := userdata.GetInstalledRoot()
			if err != nil {
				return fmt.Errorf("resolving installed root: %w", err)
			}
			userdataRoot, err := userdata.GetUserdataRoot()
			if err != nil {
				return fmt.Errorf("resolving userdata root: %w", err)
			}
			if err := userdata.FixRegistry(os.Stdout, installedRoot, userdataRoot); err != nil {
				return err
			}
		}
		if traceEnv != "" {
			installedRoot, err := userdata.GetInstalledRoot()
			if err != nil {
				return fmt.Errorf("resolving installed root: %w", err)
			}
			userdataRoot, err := userdata.GetUserdataRoot()
			if err != nil {
				return fmt.Errorf("resolving userdata root: %w", err)
			}
			if err := userdata.TraceEnv(os.Stdout, traceEnv, installedRoot, userdataRoot); err != nil {
				return err
			}
		}

		return nil
	},
}

func runAllChecks() {
	runRuntimeCheck()
	runExtensionsCheck()
	userdata.CheckUserdata(os.Stdout, false)

	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		fmt.Printf("[WARN] Could not resolve installed root: %v\n", err)
		return
	}
	userdataRoot, err := userdata.GetUserdataRoot()
	if err != nil {
		fmt.Printf("[WARN] Could not resolve userdata root: %v\n", err)
		return
	}
	if err := userdata.CheckCLIDeps(os.Stdout, installedRoot); err != nil {
		fmt.Printf("[WARN] CLI check failed: %v\n", err)
	}
	if err := userdata.CheckRegistry(os.Stdout, installedRoot, userdataRoot); err != nil {
		fmt.Printf("[WARN] Registry check failed: %v\n", err)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Printf("[WARN] Could not resolve project root: %v\n", err)
		return
	}
	if err := userdata.CheckLinks(os.Stdout, repoRoot); err != nil {
		fmt.Printf("[WARN] Link check failed: %v\n", err)
	}
}

func runRuntimeCheck() {
	fmt.Println("Runtime check:")
	checkBinary("go")
	checkBinary("node")
	checkBinary("git")
}

func checkBinary(name string) {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Printf("  [MISS] %s not found\n", name)
		return
	}
	fmt.Printf("  [ OK ] %s found at %s\n", name, path)
}

func runExtensionsCheck() {
	fmt.Println("Extensions check:")

	repoRoot, err := findRepoRoot()
	if err != nil {
		fmt.Printf("  [WARN] Cannot determine repo root: %v\n", err)
		return
	}

	// Check that project.yaml is parseable.
	configPath := filepath.Join(repoRoot, extension.ProjectConfigFile)
	cfg, err := extension.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("  [FAIL] Cannot parse %s: %v\n", extension.ProjectConfigFile, err)
		return
	}
	fmt.Printf("  [ OK ] %s is valid\n", extension.ProjectConfigFile)

	if len(cfg.Extensions) == 0 {
		fmt.Printf("  [INFO] No extensions declared\n")
		return
	}

	// Check each declared extension.
	statuses, err := extension.List(repoRoot)
	if err != nil {
		fmt.Printf("  [WARN] Cannot list extension status: %v\n", err)
		return
	}

	for _, s := range statuses {
		// Check that the directory exists.
		extDir := filepath.Join(repoRoot, s.Path)
		if _, statErr := os.Stat(extDir); statErr != nil {
			fmt.Printf("  [FAIL] %s: directory missing (%s)\n", s.Name, s.Path)
			continue
		}

		switch s.Status {
		case "ok":
			fmt.Printf("  [ OK ] %s: clean\n", s.Name)
		case "uninitialized":
			fmt.Printf("  [WARN] %s: submodule not initialized (run `agentx extension sync`)\n", s.Name)
		case "modified":
			fmt.Printf("  [WARN] %s: submodule has local modifications\n", s.Name)
		case "missing":
			fmt.Printf("  [FAIL] %s: not tracked as a git submodule\n", s.Name)
		default:
			fmt.Printf("  [WARN] %s: unknown status %q\n", s.Name, s.Status)
		}
	}
}

func runManifestCheck(path string) error {
	fmt.Printf("Manifest validation: %s\n", path)

	// Validate against JSON Schema.
	result, err := manifest.ValidateFile(path)
	if err != nil {
		fmt.Printf("  [FAIL] %v\n", err)
		return fmt.Errorf("manifest validation failed: %w", err)
	}

	if result.Valid {
		// Parse to get type and name for the success message.
		base, err := manifest.Parse(path)
		if err != nil {
			fmt.Printf("  [ OK ] Valid manifest\n")
			return nil
		}
		fmt.Printf("  [ OK ] Valid %s manifest: %s (v%s)\n", base.Type, base.Name, base.Version)
		return nil
	}

	// Report validation issues.
	fmt.Printf("  [FAIL] %d validation issue(s):\n", len(result.Issues))
	for _, issue := range result.Issues {
		if issue.Path != "" {
			fmt.Printf("    - %s: %s\n", issue.Path, issue.Message)
		} else {
			fmt.Printf("    - %s\n", issue.Message)
		}
	}
	return fmt.Errorf("manifest %s has %d validation issue(s)", path, len(result.Issues))
}
