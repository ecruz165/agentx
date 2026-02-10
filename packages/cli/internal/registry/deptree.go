package registry

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// BuildDependencyTree resolves a type and recursively builds its dependency tree.
// It marks nodes as Deduped if they appear more than once in the tree, and as
// Installed if they already exist in installedRoot.
func BuildDependencyTree(typePath string, sources []Source, installedRoot string) (*DependencyNode, error) {
	seen := make(map[string]bool)
	return buildNode(typePath, sources, installedRoot, seen)
}

func buildNode(typePath string, sources []Source, installedRoot string, seen map[string]bool) (*DependencyNode, error) {
	category := categoryFromPath(typePath)

	node := &DependencyNode{
		TypePath: typePath,
		Category: category,
	}

	// Check if already seen (dedup).
	if seen[typePath] {
		node.Deduped = true
		return node, nil
	}
	seen[typePath] = true

	// Check if already installed.
	if installedRoot != "" {
		installedDir := filepath.Join(installedRoot, typePath)
		if _, err := os.Stat(installedDir); err == nil {
			node.Installed = true
		}
	}

	// Resolve the type across sources.
	resolved, err := ResolveType(typePath, sources)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %w", typePath, err)
	}
	node.Resolved = resolved

	// Extract dependencies from the manifest.
	deps, err := extractDependencies(resolved.ManifestPath)
	if err != nil {
		return nil, fmt.Errorf("extracting dependencies from %s: %w", resolved.ManifestPath, err)
	}

	// Recursively build children.
	for _, depPath := range deps {
		child, err := buildNode(depPath, sources, installedRoot, seen)
		if err != nil {
			return nil, err
		}
		node.Children = append(node.Children, child)
	}

	return node, nil
}

// extractDependencies parses a manifest file and returns type paths of all dependencies.
// The dependency mapping is:
//   - Prompt: persona, context[], skills[], workflows[]
//   - Workflow: steps[].skill
//   - Persona: context[]
//   - Context, Skill, Template: no type dependencies
func extractDependencies(manifestPath string) ([]string, error) {
	parsed, err := manifest.ParseFile(manifestPath)
	if err != nil {
		return nil, err
	}

	var deps []string

	switch m := parsed.(type) {
	case *manifest.PromptManifest:
		if m.Persona != "" {
			deps = append(deps, m.Persona)
		}
		deps = append(deps, m.Context...)
		deps = append(deps, m.Skills...)
		deps = append(deps, m.Workflows...)

	case *manifest.WorkflowManifest:
		for _, step := range m.Steps {
			if step.Skill != "" {
				deps = append(deps, step.Skill)
			}
		}

	case *manifest.PersonaManifest:
		deps = append(deps, m.Context...)

	case *manifest.ContextManifest:
		// No type dependencies.

	case *manifest.SkillManifest:
		// No type dependencies (only CLI deps).

	case *manifest.TemplateManifest:
		// No type dependencies.
	}

	return deps, nil
}

// FlattenTree returns all resolved types in topological order (dependencies first),
// with duplicates and already-installed types removed.
func FlattenTree(root *DependencyNode) []*ResolvedType {
	seen := make(map[string]bool)
	var result []*ResolvedType
	flattenRecursive(root, seen, &result)
	return result
}

func flattenRecursive(node *DependencyNode, seen map[string]bool, result *[]*ResolvedType) {
	if node == nil || node.Deduped || node.Installed || seen[node.TypePath] {
		return
	}

	// Process children first (dependencies before dependents).
	for _, child := range node.Children {
		flattenRecursive(child, seen, result)
	}

	if !seen[node.TypePath] && node.Resolved != nil {
		seen[node.TypePath] = true
		*result = append(*result, node.Resolved)
	}
}
