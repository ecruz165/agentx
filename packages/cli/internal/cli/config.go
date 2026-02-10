package cli

import (
	"fmt"

	"github.com/agentx-labs/agentx/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configGetCmd)
	rootCmd.AddCommand(configCmd)
}

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage user settings",
	Long:  `Read and write AgentX configuration stored at ~/.agentx/config.yaml.`,
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		config.Load()
		key, value := args[0], args[1]
		if err := config.Set(key, value); err != nil {
			return fmt.Errorf("setting config key %q: %w", key, err)
		}
		fmt.Printf("Set %s = %s\n", key, value)
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Get a configuration value",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		config.Load()
		value := config.Get(args[0])
		fmt.Println(value)
		return nil
	},
}
