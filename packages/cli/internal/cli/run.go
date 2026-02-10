package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/runtime"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var runInputs []string

var runCmd = &cobra.Command{
	Use:   "run <type-path>",
	Short: "Run a skill or workflow",
	Long: `Execute an installed skill or workflow.

For skills, this invokes the skill's runtime (node, go) with the provided inputs.
For workflows, this runs each step in sequence, passing outputs between steps.

Inputs are provided as key=value pairs via --input flags.`,
	Args: cobra.ExactArgs(1),
	RunE: runRun,
}

func init() {
	runCmd.Flags().StringArrayVarP(&runInputs, "input", "i", nil, "Input key=value pairs (can be specified multiple times)")
	rootCmd.AddCommand(runCmd)
}

func runRun(cmd *cobra.Command, args []string) error {
	typePath := args[0]

	// Parse --input flags into a map.
	inputArgs, err := parseInputArgs(runInputs)
	if err != nil {
		return err
	}

	// Resolve the installed type path.
	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	typeDir := filepath.Join(installedRoot, typePath)
	if _, err := os.Stat(typeDir); err != nil {
		return fmt.Errorf("type %q is not installed (run `agentx install %s` first)", typePath, typePath)
	}

	// Find and parse the manifest.
	manifestPath, err := findManifestInDir(typeDir)
	if err != nil {
		return fmt.Errorf("finding manifest for %s: %w", typePath, err)
	}

	parsed, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return fmt.Errorf("parsing manifest for %s: %w", typePath, err)
	}

	ctx := context.Background()

	switch m := parsed.(type) {
	case *manifest.SkillManifest:
		return runSkill(ctx, cmd, typePath, typeDir, m, inputArgs)
	case *manifest.WorkflowManifest:
		return runWorkflow(ctx, cmd, typePath, installedRoot, m, inputArgs)
	default:
		return fmt.Errorf("type %q is a %T — only skills and workflows can be run", typePath, parsed)
	}
}

// runSkill dispatches a single skill to its runtime.
func runSkill(ctx context.Context, cmd *cobra.Command, typePath, typeDir string, m *manifest.SkillManifest, args map[string]string) error {
	// Validate required inputs.
	if err := validateInputs(m.Inputs, args); err != nil {
		return fmt.Errorf("input validation for %s: %w", typePath, err)
	}

	// Ensure the skill registry exists.
	registryName := strings.TrimPrefix(typePath, "skills/")
	if err := userdata.EnsureSkillRegistry(registryName); err != nil {
		return fmt.Errorf("ensuring skill registry for %s: %w", typePath, err)
	}

	// Dispatch to the appropriate runtime.
	rt := runtime.DispatchRuntime(m.Runtime)

	fmt.Fprintf(cmd.OutOrStdout(), "Running %s (runtime: %s)...\n", m.Name, m.Runtime)

	output, err := rt.Run(ctx, typeDir, m, args)
	if err != nil {
		return fmt.Errorf("running skill %s: %w", typePath, err)
	}

	if output.ExitCode != 0 {
		return fmt.Errorf("skill %s exited with code %d", typePath, output.ExitCode)
	}

	return nil
}

// runWorkflow executes a workflow by running each step in sequence.
func runWorkflow(ctx context.Context, cmd *cobra.Command, typePath, installedRoot string, m *manifest.WorkflowManifest, args map[string]string) error {
	// Validate required inputs.
	if err := validateInputs(m.Inputs, args); err != nil {
		return fmt.Errorf("input validation for %s: %w", typePath, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Running workflow %s (%d steps)...\n", m.Name, len(m.Steps))

	for i, step := range m.Steps {
		fmt.Fprintf(cmd.OutOrStdout(), "\n--- Step %d/%d: %s (skill: %s) ---\n", i+1, len(m.Steps), step.ID, step.Skill)

		// Resolve the skill type path.
		skillTypePath := step.Skill
		if !strings.HasPrefix(skillTypePath, "skills/") {
			skillTypePath = "skills/" + skillTypePath
		}

		skillDir := filepath.Join(installedRoot, skillTypePath)
		if _, err := os.Stat(skillDir); err != nil {
			return fmt.Errorf("step %q: skill %q is not installed", step.ID, step.Skill)
		}

		// Parse the skill manifest.
		skillManifestPath, err := findManifestInDir(skillDir)
		if err != nil {
			return fmt.Errorf("step %q: finding manifest for skill %s: %w", step.ID, step.Skill, err)
		}

		parsed, err := manifest.ParseFile(skillManifestPath)
		if err != nil {
			return fmt.Errorf("step %q: parsing manifest for skill %s: %w", step.ID, step.Skill, err)
		}

		skillManifest, ok := parsed.(*manifest.SkillManifest)
		if !ok {
			return fmt.Errorf("step %q: %s is not a skill", step.ID, step.Skill)
		}

		// Merge workflow-level args with step-specific inputs.
		// Step inputs take precedence over workflow-level args.
		stepArgs := mergeArgs(args, step.Inputs)

		// Ensure the skill registry exists.
		registryName := strings.TrimPrefix(skillTypePath, "skills/")
		if err := userdata.EnsureSkillRegistry(registryName); err != nil {
			return fmt.Errorf("step %q: ensuring skill registry: %w", step.ID, err)
		}

		// Dispatch to runtime.
		rt := runtime.DispatchRuntime(skillManifest.Runtime)
		output, err := rt.Run(ctx, skillDir, skillManifest, stepArgs)
		if err != nil {
			return fmt.Errorf("step %q: running skill %s: %w", step.ID, step.Skill, err)
		}

		if output.ExitCode != 0 {
			return fmt.Errorf("step %q: skill %s exited with code %d", step.ID, step.Skill, output.ExitCode)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "--- Step %d/%d complete ---\n", i+1, len(m.Steps))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\nWorkflow %s completed successfully.\n", m.Name)
	return nil
}

// parseInputArgs parses --input key=value flags into a map.
func parseInputArgs(inputs []string) (map[string]string, error) {
	result := make(map[string]string)
	for _, input := range inputs {
		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid input format %q: expected key=value", input)
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "" {
			return nil, fmt.Errorf("invalid input format %q: key cannot be empty", input)
		}
		result[key] = value
	}
	return result, nil
}

// validateInputs checks that all required inputs are provided and applies defaults
// for optional inputs that are missing.
func validateInputs(fields []manifest.InputField, args map[string]string) error {
	for _, field := range fields {
		if _, ok := args[field.Name]; ok {
			continue
		}
		if field.Required {
			return fmt.Errorf("required input %q is missing", field.Name)
		}
		// Apply default value if available.
		if field.Default != nil {
			args[field.Name] = fmt.Sprintf("%v", field.Default)
		}
	}
	return nil
}

// mergeArgs merges workflow-level args with step-specific inputs.
// Step inputs (which may contain template references) take precedence.
func mergeArgs(workflowArgs map[string]string, stepInputs map[string]interface{}) map[string]string {
	merged := make(map[string]string, len(workflowArgs)+len(stepInputs))
	for k, v := range workflowArgs {
		merged[k] = v
	}
	for k, v := range stepInputs {
		merged[k] = fmt.Sprintf("%v", v)
	}
	return merged
}

// findManifestInDir searches for a manifest file in the given directory.
// It tries manifest.yaml first, then manifest.json, then category-specific names.
func findManifestInDir(dir string) (string, error) {
	candidates := []string{"manifest.yaml", "manifest.json"}

	// Also try type-specific manifest names.
	for _, typeName := range manifest.ValidTypes {
		candidates = append(candidates, typeName+".yaml")
	}

	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no manifest found in %s", dir)
}
