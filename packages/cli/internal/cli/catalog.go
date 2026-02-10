package cli

import (
	"fmt"
	"time"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/catalog"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

func init() {
	catalogCmd.AddCommand(catalogUpdateCmd)
	catalogCmd.AddCommand(catalogStatusCmd)
	rootCmd.AddCommand(catalogCmd)
}

var catalogCmd = &cobra.Command{
	Use:   "catalog",
	Short: "Manage the AgentX type catalog",
	Long: `Manage the catalog of available AgentX types (skills, workflows, etc.).

In end-user mode, the catalog is a shallow clone of the AgentX repository's
catalog/ directory, stored at ~/.agentx/catalog-repo/.

In platform-team mode (AGENTX_HOME set), the catalog is part of the
repository and should be updated via git pull.`,
}

var catalogUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update the catalog to the latest version",
	Long: `Pull the latest catalog types from the remote repository.

In end-user mode, this runs git pull in ~/.agentx/catalog-repo/.
If the catalog hasn't been cloned yet, it will be cloned first.

In platform-team mode, this prints a message directing you to use
git pull in the repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if userdata.DetectMode() == userdata.ModePlatformTeam {
			fmt.Fprintln(cmd.OutOrStdout(), "Catalog is managed by the repository.")
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'git pull' in your AgentX repo to update.")
			return nil
		}

		catalogRepoRoot, err := userdata.GetCatalogRepoRoot()
		if err != nil {
			return fmt.Errorf("resolving catalog path: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Updating catalog at %s...\n", catalogRepoRoot)

		if err := catalog.Update(catalogRepoRoot); err != nil {
			return fmt.Errorf("updating catalog: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Catalog updated successfully.")
		return nil
	},
}

var catalogStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show catalog status and location",
	RunE: func(cmd *cobra.Command, args []string) error {
		mode := userdata.DetectMode()
		fmt.Fprintf(cmd.OutOrStdout(), "Mode:         %s\n", mode)

		if mode == userdata.ModePlatformTeam {
			home := fmt.Sprintf("$%s/catalog/", branding.EnvVar("HOME"))
			fmt.Fprintf(cmd.OutOrStdout(), "Catalog path: %s\n", home)
			fmt.Fprintln(cmd.OutOrStdout(), "Managed by:   git repository")
			return nil
		}

		catalogRoot, err := userdata.GetCatalogRoot()
		if err != nil {
			return fmt.Errorf("resolving catalog path: %w", err)
		}
		catalogRepoRoot, err := userdata.GetCatalogRepoRoot()
		if err != nil {
			return fmt.Errorf("resolving catalog repo path: %w", err)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Catalog path: %s\n", catalogRoot)
		fmt.Fprintf(cmd.OutOrStdout(), "Repo URL:     %s\n", catalog.RepoURL())

		exists, _ := userdata.CatalogExists()
		if !exists {
			fmt.Fprintln(cmd.OutOrStdout(), "Status:       not installed")
			fmt.Fprintln(cmd.OutOrStdout(), "\nRun 'agentx catalog update' or 'agentx init --global' to install.")
			return nil
		}

		lastUpdated := catalog.ReadFreshnessMarker(catalogRepoRoot)
		if lastUpdated.IsZero() {
			fmt.Fprintln(cmd.OutOrStdout(), "Last updated: unknown")
		} else {
			age := time.Since(lastUpdated).Truncate(time.Minute)
			fmt.Fprintf(cmd.OutOrStdout(), "Last updated: %s (%s ago)\n", lastUpdated.Format(time.RFC3339), age)
		}

		if catalog.IsStale(catalogRepoRoot, catalog.DefaultMaxAge) {
			fmt.Fprintln(cmd.OutOrStdout(), "Status:       stale (run 'agentx catalog update')")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "Status:       up to date")
		}

		return nil
	},
}
