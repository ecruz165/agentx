package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/agentx-labs/agentx/internal/catalog"
	"github.com/agentx-labs/agentx/internal/linker"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	initGlobal bool
	initTools  string
)

func init() {
	initCmd.Flags().BoolVar(&initGlobal, "global", false, "Initialize global userdata directory (~/.agentx/userdata/)")
	initCmd.Flags().StringVar(&initTools, "tools", "claude-code,copilot,augment,opencode", "Comma-separated list of AI tools to configure")
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize AgentX configuration",
	Long: `Initialize the AgentX configuration.

Without flags, creates a project-level .agentx/project.yaml in the current directory.
With --global, initializes the global userdata directory (~/.agentx/userdata/).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if initGlobal {
			return runGlobalInit()
		}
		return runProjectInit()
	},
}

func runGlobalInit() error {
	root, err := userdata.GetUserdataRoot()
	if err != nil {
		return err
	}
	fmt.Printf("Initializing userdata at %s\n", root)

	if err := userdata.InitGlobal(os.Stdout); err != nil {
		return fmt.Errorf("initializing userdata: %w", err)
	}

	fmt.Println("\nUserdata initialized successfully.")

	// Clone the catalog if not already present (end-user mode only).
	if userdata.DetectMode() == userdata.ModeEndUser {
		exists, _ := userdata.CatalogExists()
		if !exists {
			catalogRepoRoot, err := userdata.GetCatalogRepoRoot()
			if err != nil {
				fmt.Printf("\nWarning: could not determine catalog path: %v\n", err)
				fmt.Println("Run 'agentx catalog update' later to fetch the catalog.")
				return nil
			}

			fmt.Printf("\nCloning catalog to %s...\n", catalogRepoRoot)
			if err := catalog.Clone(catalogRepoRoot); err != nil {
				fmt.Printf("Warning: catalog clone failed: %v\n", err)
				fmt.Println("Run 'agentx catalog update' later to retry.")
				return nil // Non-fatal: userdata init still succeeded.
			}
			fmt.Println("Catalog cloned successfully.")
		}
	}

	return nil
}

func runProjectInit() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting current directory: %w", err)
	}

	// Check if project is already initialized
	configPath := linker.ProjectConfigPath(cwd)
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("project already initialized: %s exists", configPath)
	}

	// Parse tools list
	tools := parseToolsList(initTools)
	if len(tools) == 0 {
		return fmt.Errorf("at least one tool must be specified via --tools")
	}

	fmt.Printf("Initializing AgentX project in %s\n", cwd)
	fmt.Printf("Tools: %s\n", strings.Join(tools, ", "))

	if err := linker.InitProject(cwd, tools); err != nil {
		return fmt.Errorf("initializing project: %w", err)
	}

	fmt.Printf("\nProject initialized. Created %s\n", configPath)
	fmt.Println("Use 'agentx link add <type>' to link types to this project.")
	return nil
}

func parseToolsList(s string) []string {
	parts := strings.Split(s, ",")
	tools := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			tools = append(tools, trimmed)
		}
	}
	return tools
}
