package registry

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// BuildInstallPlan builds an install plan for the given type path.
// If noDeps is true, only the root type is included (no dependency resolution).
func BuildInstallPlan(typePath string, sources []Source, installedRoot string, noDeps bool) (*InstallPlan, error) {
	if noDeps {
		return buildNoDepsPlan(typePath, sources, installedRoot)
	}

	root, err := BuildDependencyTree(typePath, sources, installedRoot)
	if err != nil {
		return nil, err
	}

	allTypes := FlattenTree(root)
	counts := countByCategory(allTypes)
	skipCount := countInstalled(root)
	cliDeps := checkCLIDeps(allTypes)

	return &InstallPlan{
		Root:      root,
		AllTypes:  allTypes,
		Counts:    counts,
		CLIDeps:   cliDeps,
		SkipCount: skipCount,
	}, nil
}

func buildNoDepsPlan(typePath string, sources []Source, installedRoot string) (*InstallPlan, error) {
	resolved, err := ResolveType(typePath, sources)
	if err != nil {
		return nil, err
	}

	node := &DependencyNode{
		TypePath: typePath,
		Category: resolved.Category,
		Resolved: resolved,
	}

	allTypes := []*ResolvedType{resolved}
	counts := countByCategory(allTypes)
	cliDeps := checkCLIDeps(allTypes)

	return &InstallPlan{
		Root:     node,
		AllTypes: allTypes,
		Counts:   counts,
		CLIDeps:  cliDeps,
	}, nil
}

func countByCategory(types []*ResolvedType) map[string]int {
	counts := make(map[string]int)
	for _, t := range types {
		counts[t.Category]++
	}
	return counts
}

func countInstalled(node *DependencyNode) int {
	if node == nil {
		return 0
	}
	count := 0
	if node.Installed {
		count = 1
	}
	for _, child := range node.Children {
		count += countInstalled(child)
	}
	return count
}

// checkCLIDeps checks CLI dependencies for all skill types using exec.LookPath.
func checkCLIDeps(types []*ResolvedType) []CLIDepStatus {
	seen := make(map[string]bool)
	var result []CLIDepStatus

	for _, t := range types {
		if t.Category != "skill" {
			continue
		}

		parsed, err := manifest.ParseFile(t.ManifestPath)
		if err != nil {
			continue
		}

		skill, ok := parsed.(*manifest.SkillManifest)
		if !ok || len(skill.CLIDependencies) == 0 {
			continue
		}

		for _, dep := range skill.CLIDependencies {
			if seen[dep.Name] {
				continue
			}
			seen[dep.Name] = true

			_, err := exec.LookPath(dep.Name)
			result = append(result, CLIDepStatus{
				Name:      dep.Name,
				Available: err == nil,
			})
		}
	}

	return result
}

// PrintTree prints the dependency tree with box-drawing characters.
func PrintTree(w io.Writer, node *DependencyNode, prefix string, isLast bool) {
	if node == nil {
		return
	}

	// Determine the connector for this node.
	connector := "├── "
	if isLast {
		connector = "└── "
	}

	// Build the label.
	label := fmt.Sprintf("%s: %s", node.Category, NameFromPath(node.TypePath))
	if node.Deduped {
		label += " (deduped)"
	} else if node.Installed {
		label += " (already installed)"
	}

	// For the root node, don't print a connector.
	if prefix == "" {
		fmt.Fprintf(w, "  %s\n", label)
	} else {
		fmt.Fprintf(w, "  %s%s%s\n", prefix, connector, label)
	}

	// Print children.
	childPrefix := prefix
	if prefix != "" {
		if isLast {
			childPrefix += "    "
		} else {
			childPrefix += "│   "
		}
	}

	for i, child := range node.Children {
		isChildLast := i == len(node.Children)-1
		PrintTree(w, child, childPrefix, isChildLast)
	}
}

// PrintPlan prints the full install plan summary.
func PrintPlan(w io.Writer, plan *InstallPlan) {
	fmt.Fprintln(w, "Resolving dependencies...")
	fmt.Fprintln(w)

	// Print tree.
	PrintTree(w, plan.Root, "", true)
	fmt.Fprintln(w)

	// Print summary counts.
	var parts []string
	order := []string{"prompt", "persona", "context", "skill", "workflow", "template"}
	total := 0
	for _, cat := range order {
		if count, ok := plan.Counts[cat]; ok && count > 0 {
			noun := cat
			if count != 1 {
				noun = pluralize(cat)
			}
			parts = append(parts, fmt.Sprintf("%d %s", count, noun))
			total += count
		}
	}
	if len(parts) > 0 {
		fmt.Fprintf(w, "  Install: %s (%d types)\n", strings.Join(parts, ", "), total)
	}

	// Print CLI dep status.
	if len(plan.CLIDeps) > 0 {
		var depParts []string
		var missing []string
		for _, dep := range plan.CLIDeps {
			if dep.Available {
				depParts = append(depParts, fmt.Sprintf("%s \u2713", dep.Name))
			} else {
				depParts = append(depParts, fmt.Sprintf("%s \u2717", dep.Name))
				missing = append(missing, dep.Name)
			}
		}
		fmt.Fprintf(w, "  CLI deps required: %s\n", strings.Join(depParts, ", "))
		for _, m := range missing {
			fmt.Fprintf(w, "\n  Warning: Missing CLI: %s (run `agentx doctor --fix` after install)\n", m)
		}
	}

	if plan.SkipCount > 0 {
		fmt.Fprintf(w, "  (%d types already installed, will be skipped)\n", plan.SkipCount)
	}

	fmt.Fprintln(w)
}

// NameFromPath extracts the display name from a type path.
// "skills/scm/git/commit-analyzer" -> "scm/git/commit-analyzer"
// "personas/senior-java-dev" -> "senior-java-dev"
func NameFromPath(typePath string) string {
	parts := strings.SplitN(typePath, "/", 2)
	if len(parts) < 2 {
		return typePath
	}
	return parts[1]
}

// pluralize returns the plural form of a category name.
func pluralize(category string) string {
	switch category {
	case "context":
		return "contexts"
	case "persona":
		return "personas"
	case "skill":
		return "skills"
	case "workflow":
		return "workflows"
	case "prompt":
		return "prompts"
	case "template":
		return "templates"
	default:
		return category + "s"
	}
}
