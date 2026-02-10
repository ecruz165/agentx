package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/agentx-labs/agentx/internal/branding"
	"github.com/agentx-labs/agentx/internal/extension"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var extensionBranch string

func init() {
	extensionAddCmd.Flags().StringVar(&extensionBranch, "branch", "main", "Git branch to track")

	extensionCmd.AddCommand(extensionAddCmd)
	extensionCmd.AddCommand(extensionRemoveCmd)
	extensionCmd.AddCommand(extensionListCmd)
	extensionCmd.AddCommand(extensionSyncCmd)
	rootCmd.AddCommand(extensionCmd)
}

var extensionCmd = &cobra.Command{
	Use:     "extension",
	Aliases: []string{"ext"},
	Short:   "Manage extension repositories",
	Long: `Manage extension repositories that provide additional AgentX types.

In platform-team mode (AGENTX_HOME set), extensions are git submodules.
In end-user mode, extensions are cloned to ~/.agentx/extensions/.`,
}

var extensionAddCmd = &cobra.Command{
	Use:   "add <name> <git-url>",
	Short: "Add an extension repository",
	Long: `Add a git repository as an extension.

In platform-team mode, the extension is added as a git submodule and
registered in project.yaml.

In end-user mode, the repository is cloned to ~/.agentx/extensions/<name>/.

Example:
  agentx extension add acme-corp https://github.com/acme/agentx-types.git
  agentx extension add my-types https://github.com/me/types.git --branch develop`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		gitURL := args[1]

		fmt.Fprintf(cmd.OutOrStdout(), "Adding extension %q from %s (branch: %s)...\n", name, gitURL, extensionBranch)

		if userdata.DetectMode() == userdata.ModePlatformTeam {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}
			if err := extension.Add(repoRoot, name, gitURL, extensionBranch); err != nil {
				return fmt.Errorf("adding extension: %w", err)
			}
		} else {
			extRoot, err := userdata.GetExtensionsRoot()
			if err != nil {
				return fmt.Errorf("resolving extensions directory: %w", err)
			}
			if err := extension.UserAdd(extRoot, name, gitURL, extensionBranch); err != nil {
				return fmt.Errorf("adding extension: %w", err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Extension %q added successfully.\n", name)
		return nil
	},
}

var extensionRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove an extension repository",
	Long: `Remove an extension.

In platform-team mode, the submodule is deinitialized, removed from git,
and deregistered from project.yaml.

In end-user mode, the extension directory is deleted from ~/.agentx/extensions/.

Example:
  agentx extension remove acme-corp`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		fmt.Fprintf(cmd.OutOrStdout(), "Removing extension %q...\n", name)

		if userdata.DetectMode() == userdata.ModePlatformTeam {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}
			if err := extension.Remove(repoRoot, name); err != nil {
				return fmt.Errorf("removing extension: %w", err)
			}
		} else {
			extRoot, err := userdata.GetExtensionsRoot()
			if err != nil {
				return fmt.Errorf("resolving extensions directory: %w", err)
			}
			if err := extension.UserRemove(extRoot, name); err != nil {
				return fmt.Errorf("removing extension: %w", err)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Extension %q removed successfully.\n", name)
		return nil
	},
}

var extensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all extensions and their status",
	Long: `List all extensions with their status.

In platform-team mode, lists extensions declared in project.yaml.
In end-user mode, scans ~/.agentx/extensions/ for cloned repos.

Status values:
  ok             — extension is initialized and clean
  dirty          — extension has local changes
  uninitialized  — submodule exists but is not initialized (platform-team only)
  modified       — submodule has local changes (platform-team only)
  missing        — submodule path not found by git (platform-team only)`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if userdata.DetectMode() == userdata.ModePlatformTeam {
			return listPlatformExtensions(cmd)
		}
		return listUserExtensions(cmd)
	},
}

var extensionSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Update all extensions",
	Long: `Update all extensions to the latest version.

In platform-team mode, runs git submodule update --init --recursive.
In end-user mode, runs git pull --rebase in each extension directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), "Syncing extensions...")

		if userdata.DetectMode() == userdata.ModePlatformTeam {
			repoRoot, err := findRepoRoot()
			if err != nil {
				return err
			}
			if err := extension.Sync(repoRoot); err != nil {
				return fmt.Errorf("syncing extensions: %w", err)
			}
		} else {
			extRoot, err := userdata.GetExtensionsRoot()
			if err != nil {
				return fmt.Errorf("resolving extensions directory: %w", err)
			}
			if err := extension.UserSync(extRoot); err != nil {
				return fmt.Errorf("syncing extensions: %w", err)
			}
		}

		fmt.Fprintln(cmd.OutOrStdout(), "Extensions synced successfully.")
		return nil
	},
}

func listPlatformExtensions(cmd *cobra.Command) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}

	statuses, err := extension.List(repoRoot)
	if err != nil {
		return fmt.Errorf("listing extensions: %w", err)
	}

	if len(statuses) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No extensions configured.")
		fmt.Fprintln(cmd.OutOrStdout(), "Use `agentx extension add <name> <git-url>` to add one.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tBRANCH\tSTATUS")
	for _, s := range statuses {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Path, s.Branch, s.Status)
	}
	return w.Flush()
}

func listUserExtensions(cmd *cobra.Command) error {
	extRoot, err := userdata.GetExtensionsRoot()
	if err != nil {
		return fmt.Errorf("resolving extensions directory: %w", err)
	}

	statuses, err := extension.UserList(extRoot)
	if err != nil {
		return fmt.Errorf("listing extensions: %w", err)
	}

	if len(statuses) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No extensions installed.")
		fmt.Fprintln(cmd.OutOrStdout(), "Use `agentx extension add <name> <git-url>` to add one.")
		return nil
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "NAME\tPATH\tBRANCH\tSTATUS")
	for _, s := range statuses {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", s.Name, s.Path, s.Branch, s.Status)
	}
	return w.Flush()
}

// findRepoRoot locates the AgentX repository root.
// It checks AGENTX_HOME first, then falls back to git rev-parse.
func findRepoRoot() (string, error) {
	if home := os.Getenv(branding.EnvVar("HOME")); home != "" {
		return home, nil
	}

	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("cannot determine repository root: set %s or run from within a git repository", branding.EnvVar("HOME"))
	}

	return strings.TrimSpace(string(output)), nil
}
