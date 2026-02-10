package registry

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// manifestNames is the fallback order for finding manifest files.
var manifestNames = []string{"manifest.yaml", "manifest.json"}

// ResolveType searches for a type across sources in priority order.
// It returns the first match found. Sources are searched in slice order
// (first source = highest priority).
func ResolveType(typePath string, sources []Source) (*ResolvedType, error) {
	category := categoryFromPath(typePath)
	if category == "" {
		return nil, fmt.Errorf("cannot determine category from type path %q", typePath)
	}

	for _, src := range sources {
		dir := filepath.Join(src.BasePath, typePath)
		manifestPath, err := findManifest(dir, typePath)
		if err != nil {
			continue // not found in this source
		}
		return &ResolvedType{
			TypePath:     typePath,
			ManifestPath: manifestPath,
			SourceDir:    dir,
			SourceName:   src.Name,
			Category:     category,
		}, nil
	}

	return nil, fmt.Errorf("type %q not found in any source", typePath)
}

// findManifest searches for a manifest file in the given directory.
// Fallback order: manifest.yaml > manifest.json > <type>.yaml
// where <type> is derived from the type path category.
func findManifest(dir, typePath string) (string, error) {
	// Try standard manifest names first.
	for _, name := range manifestNames {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	// Try type-specific name as fallback (e.g., skill.yaml, persona.yaml).
	category := categoryFromPath(typePath)
	if category != "" {
		p := filepath.Join(dir, category+".yaml")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no manifest found in %s", dir)
}

// categoryFromPath extracts the singular category name from a type path.
// "personas/senior-java-dev" -> "persona"
// "skills/scm/git/commit-analyzer" -> "skill"
// "context/spring-boot/security" -> "context"
func categoryFromPath(typePath string) string {
	parts := strings.SplitN(typePath, "/", 2)
	if len(parts) == 0 {
		return ""
	}

	plural := parts[0]
	switch plural {
	case "personas":
		return "persona"
	case "skills":
		return "skill"
	case "workflows":
		return "workflow"
	case "prompts":
		return "prompt"
	case "templates":
		return "template"
	case "context":
		return "context"
	default:
		return ""
	}
}

// typePathFromManifest constructs a type path from a source-relative directory.
// The directory should be relative to the source root (e.g., "personas/senior-java-dev").
func typePathFromManifest(relDir string) string {
	return filepath.ToSlash(relDir)
}
