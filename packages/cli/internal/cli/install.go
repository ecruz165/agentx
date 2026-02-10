package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	installNoDeps bool
	installYes    bool
)

var installCmd = &cobra.Command{
	Use:   "install <type-path>",
	Short: "Install a type and its dependencies",
	Long: `Install a type (skill, workflow, prompt, persona, context) to ~/.agentx/installed/.
Dependencies are resolved and installed by default. Use --no-deps to skip.`,
	Args: cobra.ExactArgs(1),
	RunE: runInstall,
}

func init() {
	installCmd.Flags().BoolVar(&installNoDeps, "no-deps", false, "Install only the specified type, skip dependencies")
	installCmd.Flags().BoolVarP(&installYes, "yes", "y", false, "Skip confirmation prompt")
	rootCmd.AddCommand(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	typePath := args[0]

	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	sources, err := buildSources()
	if err != nil {
		return fmt.Errorf("building sources: %w", err)
	}

	// Build install plan.
	plan, err := registry.BuildInstallPlan(typePath, sources, installedRoot, installNoDeps)
	if err != nil {
		return err
	}

	if len(plan.AllTypes) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to install — all types are already installed.")
		return nil
	}

	// Print plan.
	registry.PrintPlan(cmd.OutOrStdout(), plan)

	// Prompt for confirmation unless -y is set.
	if !installYes {
		fmt.Fprint(cmd.OutOrStdout(), "? Proceed with installation? (Y/n) ")
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			answer := strings.TrimSpace(strings.ToLower(scanner.Text()))
			if answer != "" && answer != "y" && answer != "yes" {
				fmt.Fprintln(cmd.OutOrStdout(), "Installation cancelled.")
				return nil
			}
		}
	}

	// Install types in order.
	fmt.Fprintln(cmd.OutOrStdout(), "Installing...")

	var allWarnings []string
	installed := 0

	for _, resolved := range plan.AllTypes {
		name := registry.NameFromPath(resolved.TypePath)

		if err := registry.InstallType(resolved, installedRoot); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  \u2717 %s: %s (%v)\n", resolved.Category, name, err)
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  \u2713 %s: %s\n", resolved.Category, name)
		installed++

		// Install Node dependencies if applicable.
		installedDir := filepath.Join(installedRoot, resolved.TypePath)
		if warning, err := registry.InstallNodeDeps(installedDir); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "    \u26a0\ufe0f  npm install failed: %v\n", err)
		} else if warning != "" {
			allWarnings = append(allWarnings, warning)
		}

		// Initialize skill registry.
		if resolved.Category == "skill" {
			warnings, err := registry.InitSkillRegistry(resolved, installedRoot)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "    \u26a0\ufe0f  registry init failed: %v\n", err)
			}
			for _, w := range warnings {
				fmt.Fprintf(cmd.OutOrStdout(), "    \u26a0\ufe0f  %s\n", w)
				allWarnings = append(allWarnings, w)
			}
		}
	}

	// Print summary.
	fmt.Fprintln(cmd.OutOrStdout())
	if installed > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\u2713 Installed %d types.", installed)
		if plan.SkipCount > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), " %d already installed (skipped).", plan.SkipCount)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	tokenWarnings := 0
	for _, w := range allWarnings {
		if strings.Contains(w, "required") {
			tokenWarnings++
		}
	}
	if tokenWarnings > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "  %d skills need token configuration.\n", tokenWarnings)
		fmt.Fprintln(cmd.OutOrStdout(), "  Run `agentx doctor --check-registry` to see what's missing.")
	}

	return nil
}

// buildSources constructs the resolution sources from the current environment.
//
// Resolution order:
//  1. AGENTX_HOME catalog + extensions (platform-team mode)
//  2. Binary-relative ../catalog (bundled releases)
//  3. ~/.agentx/catalog-repo/catalog/ (end-user mode)
//  4. ~/.agentx/extensions/*/ (user-local extensions, end-user mode only)
func buildSources() ([]registry.Source, error) {
	var sources []registry.Source

	// 1. Check <PREFIX>_HOME for platform-team / development use.
	if home := os.Getenv(branding.EnvVar("HOME")); home != "" {
		catalogPath := filepath.Join(home, "catalog")
		if info, err := os.Stat(catalogPath); err == nil && info.IsDir() {
			sources = append(sources, registry.Source{Name: "catalog", BasePath: catalogPath})
		}

		// Platform-team extensions (submodules inside the repo).
		extDir := filepath.Join(home, "extensions")
		if info, err := os.Stat(extDir); err == nil && info.IsDir() {
			appendExtensionSources(&sources, extDir)
		}

		if len(sources) > 0 {
			return sources, nil
		}
	}

	// 2. Try to find catalog relative to the executable (bundled release).
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		catalogPath := filepath.Join(exeDir, "..", "catalog")
		if info, err := os.Stat(catalogPath); err == nil && info.IsDir() {
			sources = append(sources, registry.Source{Name: "catalog", BasePath: catalogPath})
		}
	}

	if len(sources) > 0 {
		return sources, nil
	}

	// 3. End-user mode: check ~/.agentx/catalog-repo/catalog/.
	catalogRoot, err := userdata.GetCatalogRoot()
	if err == nil {
		if info, statErr := os.Stat(catalogRoot); statErr == nil && info.IsDir() {
			sources = append(sources, registry.Source{Name: "catalog", BasePath: catalogRoot})
		}
	}

	// 4. End-user mode: scan ~/.agentx/extensions/*/.
	extRoot, err := userdata.GetExtensionsRoot()
	if err == nil {
		if info, statErr := os.Stat(extRoot); statErr == nil && info.IsDir() {
			appendExtensionSources(&sources, extRoot)
		}
	}

	if len(sources) == 0 {
		return nil, fmt.Errorf("no type sources found. Run '%s init --global' to set up the catalog", branding.CLIName())
	}

	return sources, nil
}

// appendExtensionSources scans a directory for subdirectories and appends
// each as a registry source.
func appendExtensionSources(sources *[]registry.Source, extDir string) {
	entries, err := os.ReadDir(extDir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			*sources = append(*sources, registry.Source{
				Name:     entry.Name(),
				BasePath: filepath.Join(extDir, entry.Name()),
			})
		}
	}
}
