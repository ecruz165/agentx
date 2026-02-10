package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/platform"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var envShowNoRedact bool

func init() {
	envShowCmd.Flags().BoolVar(&envShowNoRedact, "no-redact", false, "Show values without redaction")

	envCmd.AddCommand(envListCmd)
	envCmd.AddCommand(envEditCmd)
	envCmd.AddCommand(envShowCmd)
	rootCmd.AddCommand(envCmd)
}

var envCmd = &cobra.Command{
	Use:   "env",
	Short: "Manage user environment/secret files",
	Long:  `Manage .env files for shared and skill-specific secrets.`,
}

var envListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all .env files (shared + per-skill)",
	RunE: func(cmd *cobra.Command, args []string) error {
		shared, skillSpecific, err := userdata.ListEnvFiles()
		if err != nil {
			return fmt.Errorf("listing env files: %w", err)
		}

		root, _ := userdata.GetUserdataRoot()

		if len(shared) > 0 {
			fmt.Println("Shared:")
			for _, path := range shared {
				rel, _ := filepath.Rel(root, path)
				fmt.Printf("  %s\n", rel)
			}
		}

		if len(skillSpecific) > 0 {
			fmt.Println("Skill-specific:")
			for _, path := range skillSpecific {
				rel, _ := filepath.Rel(root, path)
				fmt.Printf("  %s\n", rel)
			}
		}

		if len(shared) == 0 && len(skillSpecific) == 0 {
			fmt.Println("No .env files found. Run 'agentx init --global' to create the default env.")
		}

		return nil
	},
}

// Template comments for newly created env files.
const envFileTemplate = `# Environment variables for %s
# Add KEY=VALUE pairs below. Lines starting with # are comments.
`

var envEditCmd = &cobra.Command{
	Use:   "edit <target>",
	Short: "Open env file in editor",
	Long: `Open a .env file in your preferred editor ($EDITOR, defaults to vi).

  agentx env edit aws              # opens env/aws.env (shared vendor file)
  agentx env edit cloud/aws/ssm    # opens skills/cloud/aws/ssm/tokens.env`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		path, err := userdata.ResolveEnvTarget(target)
		if err != nil {
			return err
		}

		// Create file with template if it doesn't exist.
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			// Ensure parent directory exists.
			dir := filepath.Dir(path)
			if err := os.MkdirAll(dir, userdata.DirPermSecure); err != nil {
				return fmt.Errorf("creating directory %s: %w", dir, err)
			}
			content := fmt.Sprintf(envFileTemplate, target)
			if err := os.WriteFile(path, []byte(content), userdata.FilePermSecure); err != nil {
				return fmt.Errorf("creating env file %s: %w", path, err)
			}
			fmt.Printf("Created %s\n", path)
		}

		if err := userdata.OpenEditor(path); err != nil {
			return err
		}

		// Ensure secure permissions after editing.
		platform.Chmod(path, userdata.FilePermSecure)

		return nil
	},
}

var envShowCmd = &cobra.Command{
	Use:   "show <target>",
	Short: "Print env file contents (redacted by default)",
	Long: `Print the contents of a .env file with sensitive values redacted.

  agentx env show default          # shows env/default.env
  agentx env show aws              # shows env/aws.env
  agentx env show cloud/aws/ssm    # shows skills/cloud/aws/ssm/tokens.env

Use --no-redact to show actual values.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := args[0]
		path, err := userdata.ResolveEnvTarget(target)
		if err != nil {
			return err
		}

		entries, err := userdata.ParseEnvFile(path)
		if err != nil {
			return err
		}

		if len(entries) == 0 {
			fmt.Println("(empty)")
			return nil
		}

		fmt.Printf("# %s\n", path)
		for _, e := range entries {
			value := e.Value
			if !envShowNoRedact {
				value = userdata.RedactValue(e.Key, e.Value)
			}
			fmt.Printf("%s=%s\n", e.Key, value)
		}
		return nil
	},
}
