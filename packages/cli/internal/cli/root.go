package cli

import (
	"fmt"
	"os"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/catalog"
	"github.com/agentx-labs/agentx/internal/config"
	"github.com/agentx-labs/agentx/internal/updater"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	buildVersion string
	buildCommit  string
	buildDate    string
)

var rootCmd = &cobra.Command{
	Use:   branding.CLIName(),
	Short: branding.Description(),
	Long: branding.DisplayName() + ` manages the installation, linking, and discovery of reusable types
(skills, workflows, prompts, personas, context) that power AI coding assistants.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip banners for commands that manage their own state.
		name := cmd.Name()
		if name == "update" || name == "self-update" || name == "catalog" || name == "init" {
			return
		}

		// Non-blocking banner from cached version check.
		u := updater.New(buildVersion)
		u.CheckAndPrintBanner(os.Stderr, config.Dir())

		// Catalog freshness check (end-user mode only, no network).
		if userdata.DetectMode() == userdata.ModeEndUser {
			catalogRepoRoot, err := userdata.GetCatalogRepoRoot()
			if err == nil && catalog.IsStale(catalogRepoRoot, catalog.DefaultMaxAge) {
				exists, _ := userdata.CatalogExists()
				if exists {
					fmt.Fprintf(os.Stderr, "Catalog is more than 7 days old. Run '%s catalog update'.\n", branding.CLIName())
				}
			}
		}
	},
}

// Execute runs the root command with build info injected via ldflags.
func Execute(version, commit, date string) error {
	buildVersion = version
	buildCommit = commit
	buildDate = date
	return rootCmd.Execute()
}
