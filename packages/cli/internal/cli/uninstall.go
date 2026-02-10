package cli

import (
	"fmt"

	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <type-path>",
	Short: "Remove an installed type",
	Long:  `Remove an installed type from ~/.agentx/installed/.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	typePath := args[0]

	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	if err := registry.RemoveType(typePath, installedRoot); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Removed %s\n", typePath)
	return nil
}
