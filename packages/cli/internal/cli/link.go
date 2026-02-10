package cli

import (
	"fmt"
	"os"

	"github.com/agentx-labs/agentx/internal/linker"
	"github.com/spf13/cobra"
)

func init() {
	linkCmd.AddCommand(linkAddCmd)
	linkCmd.AddCommand(linkRemoveCmd)
	linkCmd.AddCommand(linkSyncCmd)
	linkCmd.AddCommand(linkStatusCmd)
	rootCmd.AddCommand(linkCmd)
}

var linkCmd = &cobra.Command{
	Use:   "link",
	Short: "Manage linked types in this project",
	Long: `Manage which AgentX types (personas, context, skills, workflows) are
linked to this project and regenerate AI tool configurations.`,
}

var linkAddCmd = &cobra.Command{
	Use:   "add <type-path>",
	Short: "Link a type to this project",
	Long: `Add a type reference to this project's .agentx/project.yaml and regenerate
AI tool configurations.

Example:
  agentx link add personas/senior-java-dev
  agentx link add skills/scm/git/commit-analyzer
  agentx link add context/spring-boot/security`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		typeRef := args[0]
		fmt.Printf("Linking %s...\n", typeRef)

		if err := linker.AddType(cwd, typeRef); err != nil {
			return err
		}

		fmt.Printf("Linked %s successfully.\n", typeRef)
		return nil
	},
}

var linkRemoveCmd = &cobra.Command{
	Use:   "remove <type-path>",
	Short: "Unlink a type from this project",
	Long: `Remove a type reference from this project's .agentx/project.yaml and
regenerate AI tool configurations.

Example:
  agentx link remove personas/senior-java-dev`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		typeRef := args[0]
		fmt.Printf("Removing %s...\n", typeRef)

		if err := linker.RemoveType(cwd, typeRef); err != nil {
			return err
		}

		fmt.Printf("Removed %s successfully.\n", typeRef)
		return nil
	},
}

var linkSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate all AI tool configurations",
	Long: `Regenerate AI tool configuration files based on the current
.agentx/project.yaml. This re-runs all generators for the configured tools.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		fmt.Println("Syncing project configurations...")
		if err := linker.Sync(cwd); err != nil {
			return err
		}

		fmt.Println("Sync complete.")
		return nil
	},
}

var linkStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show status of linked configurations",
	Long:  `Show the current status of AI tool configurations for this project.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}

		results, err := linker.Status(cwd)
		if err != nil {
			return err
		}

		for _, r := range results {
			statusIcon := "?"
			switch r.Status {
			case "up-to-date":
				statusIcon = "OK"
			case "stale":
				statusIcon = "!!"
			case "not-generated":
				statusIcon = "--"
			}

			fmt.Printf("  [%s] %-12s %s", statusIcon, r.Tool+":", r.Status)
			if len(r.Files) > 0 {
				fmt.Printf(" (%s)", r.Files[0])
			}
			fmt.Println()

			if r.Symlinks.Total > 0 {
				fmt.Printf("       Symlinks: %d/%d valid\n", r.Symlinks.Valid, r.Symlinks.Total)
			}
		}

		return nil
	},
}
