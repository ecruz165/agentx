package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/agentx-labs/agentx/internal/compose"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	promptCopy   bool
	promptOutput string
)

var promptCmd = &cobra.Command{
	Use:   "prompt [prompt-type-path]",
	Short: "Compose a prompt from installed types",
	Long: `Compose context guidance from an installed prompt type.

When called with an argument, the prompt command loads a prompt manifest and
resolves all referenced types (persona, context, skills, workflows) to produce
a unified markdown output.

When called with no arguments, an interactive mode guides you through selecting
a persona, topic, and intent to compose an ad-hoc prompt.

Output goes to stdout by default. Use --copy to copy to clipboard or
--output to write to a file.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runPrompt,
}

func init() {
	promptCmd.Flags().BoolVar(&promptCopy, "copy", false, "Copy output to clipboard instead of printing to stdout")
	promptCmd.Flags().StringVarP(&promptOutput, "output", "o", "", "Write output to a file")
	rootCmd.AddCommand(promptCmd)
}

func runPrompt(cmd *cobra.Command, args []string) error {
	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	var cp *compose.ComposedPrompt

	if len(args) == 0 {
		// Interactive mode: guide user through persona, topic, intent selection.
		result, interactiveErr := compose.RunInteractive(installedRoot, os.Stdin, cmd.ErrOrStderr())
		if interactiveErr != nil {
			return fmt.Errorf("interactive mode: %w", interactiveErr)
		}

		cp, err = compose.ComposeFromInteractive(result, installedRoot)
		if err != nil {
			return fmt.Errorf("composing prompt: %w", err)
		}
	} else {
		typePath := args[0]

		// Verify the prompt type is installed.
		if _, statErr := os.Stat(filepath.Join(installedRoot, typePath)); statErr != nil {
			return fmt.Errorf("prompt type %q is not installed (run `agentx install %s` first)", typePath, typePath)
		}

		// Compose the prompt from all referenced types.
		cp, err = compose.Compose(typePath, installedRoot)
		if err != nil {
			return fmt.Errorf("composing prompt: %w", err)
		}
	}

	// Print warnings to stderr so they don't pollute stdout output.
	for _, w := range cp.Warnings {
		fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s\n", w)
	}

	// Render the composed prompt to markdown text.
	output := compose.Render(cp)

	// Handle output modes.
	if promptOutput != "" {
		if err := os.WriteFile(promptOutput, []byte(output), 0644); err != nil {
			return fmt.Errorf("writing output to %s: %w", promptOutput, err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Wrote prompt to %s\n", promptOutput)
		return nil
	}

	if promptCopy {
		if err := copyToClipboard(output); err != nil {
			return fmt.Errorf("copying to clipboard: %w", err)
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "Prompt copied to clipboard.\n")
		return nil
	}

	// Default: print to stdout.
	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

// copyToClipboard writes text to the system clipboard.
// It uses pbcopy on macOS, clip.exe on Windows, and xclip/xsel on Linux.
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "windows":
		cmd = exec.Command("clip.exe")
	case "linux":
		// Try xclip first, then xsel.
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else if _, err := exec.LookPath("xsel"); err == nil {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		} else {
			return fmt.Errorf("no clipboard tool found: install xclip or xsel")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
