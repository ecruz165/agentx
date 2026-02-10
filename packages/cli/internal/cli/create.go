package cli

import (
	"fmt"
	"path/filepath"
	"regexp"

	"github.com/agentx-labs/agentx/internal/scaffold"
	"github.com/spf13/cobra"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)

// Shared flag for all create subcommands.
var createOutputDir string

func init() {
	// Parent create command.
	createCmd.PersistentFlags().StringVar(&createOutputDir, "output-dir", "", "Output directory (default: ./<name>)")
	rootCmd.AddCommand(createCmd)

	// Subcommands.
	createCmd.AddCommand(createSkillCmd)
	createCmd.AddCommand(createWorkflowCmd)
	createCmd.AddCommand(createPromptCmd)
	createCmd.AddCommand(createPersonaCmd)
	createCmd.AddCommand(createContextCmd)
	createCmd.AddCommand(createTemplateCmd)
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Scaffold a new AgentX type from a template",
	Long:  `Create a new skill, workflow, prompt, persona, context, or template from built-in scaffolding templates.`,
}

// ─── create skill ──────────────────────────────────────────────────

var (
	skillTopic   string
	skillVendor  string
	skillRuntime string
)

var createSkillCmd = &cobra.Command{
	Use:   "skill <name>",
	Short: "Scaffold a new skill",
	Long: `Scaffold a new AgentX skill with the registry pattern boilerplate.

Examples:
  agentx create skill my-tool --topic cloud --vendor aws --runtime node
  agentx create skill token-counter --topic ai --runtime go`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		if skillTopic == "" {
			return fmt.Errorf("--topic is required for skills")
		}
		if err := validateName(skillTopic); err != nil {
			return fmt.Errorf("invalid topic: %w", err)
		}
		if skillVendor != "" {
			if err := validateName(skillVendor); err != nil {
				return fmt.Errorf("invalid vendor: %w", err)
			}
		}
		if skillRuntime != "node" && skillRuntime != "go" {
			return fmt.Errorf("--runtime must be 'node' or 'go', got %q", skillRuntime)
		}

		data := scaffold.NewScaffoldData(name, "skill", skillTopic, skillVendor, skillRuntime)
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("skill", data, outDir)
		if err != nil {
			return err
		}

		printResult("skill", result)

		// Next steps guidance.
		fmt.Println("\nNext steps:")
		if skillRuntime == "node" {
			fmt.Println("  1. Edit index.mjs to add your skill logic")
			fmt.Println("  2. Run 'npm install' to install dependencies")
		} else {
			fmt.Println("  1. Edit main.go to add your skill logic")
			fmt.Println("  2. Run 'go build' to verify compilation")
		}
		skillPath := data.SkillPath
		fmt.Printf("  3. Test with 'agentx run skills/%s'\n", skillPath)
		return nil
	},
}

func init() {
	createSkillCmd.Flags().StringVar(&skillTopic, "topic", "", "Skill topic (required)")
	createSkillCmd.Flags().StringVar(&skillVendor, "vendor", "", "Skill vendor (optional)")
	createSkillCmd.Flags().StringVar(&skillRuntime, "runtime", "node", "Skill runtime: node or go")
}

// ─── create workflow ───────────────────────────────────────────────

var createWorkflowCmd = &cobra.Command{
	Use:   "workflow <name>",
	Short: "Scaffold a new workflow",
	Long: `Scaffold a new AgentX workflow that composes multiple skills.

Example:
  agentx create workflow my-flow`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		data := scaffold.NewScaffoldData(name, "workflow", "", "", "node")
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("workflow", data, outDir)
		if err != nil {
			return err
		}

		printResult("workflow", result)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit workflow.yaml to define your steps")
		fmt.Println("  2. Edit index.mjs to implement step execution logic")
		fmt.Println("  3. Run 'npm install' to install dependencies")
		return nil
	},
}

// ─── create prompt ─────────────────────────────────────────────────

var createPromptCmd = &cobra.Command{
	Use:   "prompt <name>",
	Short: "Scaffold a new prompt",
	Long: `Scaffold a new AgentX prompt that composes persona, context, and skills.

Example:
  agentx create prompt my-prompt`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		data := scaffold.NewScaffoldData(name, "prompt", "", "", "")
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("prompt", data, outDir)
		if err != nil {
			return err
		}

		printResult("prompt", result)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit prompt.yaml to reference your persona, context, and skills")
		fmt.Println("  2. Edit prompt.hbs to customize the prompt template")
		return nil
	},
}

// ─── create persona ────────────────────────────────────────────────

var createPersonaCmd = &cobra.Command{
	Use:   "persona <name>",
	Short: "Scaffold a new persona",
	Long: `Scaffold a new AgentX persona that defines AI assistant behavior.

Example:
  agentx create persona senior-java-dev`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		data := scaffold.NewScaffoldData(name, "persona", "", "", "")
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("persona", data, outDir)
		if err != nil {
			return err
		}

		printResult("persona", result)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit persona.yaml to define expertise, tone, and conventions")
		return nil
	},
}

// ─── create context ────────────────────────────────────────────────

var createContextCmd = &cobra.Command{
	Use:   "context <name>",
	Short: "Scaffold a new context",
	Long: `Scaffold a new AgentX context that provides knowledge to AI assistants.

Example:
  agentx create context my-docs`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		data := scaffold.NewScaffoldData(name, "context", "", "", "")
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("context", data, outDir)
		if err != nil {
			return err
		}

		printResult("context", result)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit content.md with your context documentation")
		fmt.Println("  2. Edit context.yaml to adjust metadata if needed")
		return nil
	},
}

// ─── create template ───────────────────────────────────────────────

var createTemplateCmd = &cobra.Command{
	Use:   "template <name>",
	Short: "Scaffold a new template",
	Long: `Scaffold a new AgentX template for distributable report formats or output layouts.

Example:
  agentx create template my-template`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		if err := validateName(name); err != nil {
			return err
		}

		data := scaffold.NewScaffoldData(name, "template", "", "", "")
		outDir := resolveOutputDir(name)

		result, err := scaffold.Generate("template", data, outDir)
		if err != nil {
			return err
		}

		printResult("template", result)
		fmt.Println("\nNext steps:")
		fmt.Println("  1. Edit template.yaml to define your template variables")
		fmt.Println("  2. Edit template.hbs with your Handlebars template content")
		return nil
	},
}

// ─── Helpers ───────────────────────────────────────────────────────

func validateName(name string) error {
	if !namePattern.MatchString(name) {
		return fmt.Errorf("invalid name %q: must match pattern [a-z0-9][a-z0-9-]*", name)
	}
	return nil
}

func resolveOutputDir(name string) string {
	if createOutputDir != "" {
		return createOutputDir
	}
	return filepath.Join(".", name)
}

func printResult(typeName string, result *scaffold.Result) {
	fmt.Printf("Created %s at %s/\n", typeName, result.OutputDir)
	for _, f := range result.Files {
		fmt.Printf("  %s\n", f)
	}
	if len(result.Warnings) > 0 {
		fmt.Println("\nWarnings:")
		for _, w := range result.Warnings {
			fmt.Printf("  - %s\n", w)
		}
	}
}
