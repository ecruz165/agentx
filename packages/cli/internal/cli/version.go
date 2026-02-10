package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

var (
	versionShort bool
	versionJSON  bool
)

func init() {
	versionCmd.Flags().BoolVar(&versionShort, "short", false, "Print version number only")
	versionCmd.Flags().BoolVar(&versionJSON, "json", false, "Print version info as JSON")
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if versionShort {
			fmt.Println(buildVersion)
			return nil
		}

		if versionJSON {
			info := map[string]string{
				"version": buildVersion,
				"commit":  buildCommit,
				"date":    buildDate,
			}
			out, err := json.MarshalIndent(info, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling version info: %w", err)
			}
			fmt.Println(string(out))
			return nil
		}

		fmt.Printf("agentx version %s (commit: %s, built: %s)\n", buildVersion, buildCommit, buildDate)
		return nil
	},
}
