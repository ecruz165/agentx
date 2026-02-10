package cli

import (
	"encoding/json"
	"fmt"

	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"
)

var (
	profileShowYAML bool
	profileShowJSON bool
)

func init() {
	profileShowCmd.Flags().BoolVar(&profileShowYAML, "yaml", false, "Output as YAML")
	profileShowCmd.Flags().BoolVar(&profileShowJSON, "json", false, "Output as JSON")

	profileCmd.AddCommand(profileListCmd)
	profileCmd.AddCommand(profileUseCmd)
	profileCmd.AddCommand(profileShowCmd)
	rootCmd.AddCommand(profileCmd)
}

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "Manage user configuration profiles",
	Long:  `Manage named configuration profiles for switching between environments.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all profiles",
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := userdata.ListProfiles()
		if err != nil {
			return fmt.Errorf("listing profiles: %w", err)
		}

		if len(profiles) == 0 {
			fmt.Println("No profiles found. Run 'agentx init --global' to create the default profile.")
			return nil
		}

		activeName, _ := userdata.ActiveProfileName()

		for _, name := range profiles {
			if name == activeName {
				fmt.Printf("  %s (active)\n", name)
			} else {
				fmt.Printf("  %s\n", name)
			}
		}
		return nil
	},
}

var profileUseCmd = &cobra.Command{
	Use:   "use <name>",
	Short: "Switch active profile",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := userdata.SwitchProfile(name); err != nil {
			return err
		}
		fmt.Printf("Switched to profile %q\n", name)
		return nil
	},
}

var profileShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active profile contents",
	RunE: func(cmd *cobra.Command, args []string) error {
		profile, err := userdata.LoadProfile()
		if err != nil {
			return fmt.Errorf("loading profile: %w", err)
		}

		if profileShowJSON {
			out, err := json.MarshalIndent(profile, "", "  ")
			if err != nil {
				return fmt.Errorf("marshaling profile as JSON: %w", err)
			}
			fmt.Println(string(out))
			return nil
		}

		if profileShowYAML {
			out, err := yaml.Marshal(profile)
			if err != nil {
				return fmt.Errorf("marshaling profile as YAML: %w", err)
			}
			fmt.Print(string(out))
			return nil
		}

		// Default: human-readable format.
		activeName, _ := userdata.ActiveProfileName()
		fmt.Printf("Profile: %s\n", activeName)
		fmt.Println("---")

		if profile.Name != "" {
			fmt.Printf("  name:           %s\n", profile.Name)
		}
		if profile.AwsProfile != "" {
			fmt.Printf("  aws_profile:    %s\n", profile.AwsProfile)
		}
		if profile.AwsRegion != "" {
			fmt.Printf("  aws_region:     %s\n", profile.AwsRegion)
		}
		if profile.GithubOrg != "" {
			fmt.Printf("  github_org:     %s\n", profile.GithubOrg)
		}
		if profile.SplunkHost != "" {
			fmt.Printf("  splunk_host:    %s\n", profile.SplunkHost)
		}
		if profile.DefaultBranch != "" {
			fmt.Printf("  default_branch: %s\n", profile.DefaultBranch)
		}
		for k, v := range profile.Extras {
			fmt.Printf("  %s: %v\n", k, v)
		}
		return nil
	},
}
