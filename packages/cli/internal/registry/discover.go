package registry

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// knownCategories are the top-level directories that contain types.
var knownCategories = []string{
	"context",
	"personas",
	"skills",
	"workflows",
	"prompts",
	"templates",
}

// DiscoveredType represents a type found in a source, enriched with manifest metadata.
type DiscoveredType struct {
	TypePath    string   // e.g., "skills/scm/git/commit-analyzer"
	Category    string   // e.g., "skill", "persona", "context"
	Name        string   // display name from manifest
	Version     string   // version from manifest
	Description string   // description from manifest
	Tags        []string // tags from manifest
	Source      string   // which source it was found in
	Topic       string   // topic from skill manifest (empty for non-skills)
	Vendor      string   // vendor from manifest (empty if not set)
	CLIDeps     []string // CLI dependency names from skill manifest
}

// DiscoverAll walks all sources and returns all available types enriched with
// manifest metadata. Each returned DiscoveredType includes the type path,
// manifest metadata (name, version, description, tags), and source name.
// Types found in earlier sources take priority (later duplicates are skipped).
func DiscoverAll(sources []Source) ([]DiscoveredType, error) {
	resolved, err := DiscoverTypes(sources)
	if err != nil {
		return nil, err
	}

	var result []DiscoveredType
	for _, r := range resolved {
		dt := DiscoveredType{
			TypePath: r.TypePath,
			Category: r.Category,
			Name:     NameFromPath(r.TypePath),
			Source:   r.SourceName,
		}

		// Enrich with manifest metadata if parseable.
		base, err := manifest.Parse(r.ManifestPath)
		if err == nil {
			if base.Name != "" {
				dt.Name = base.Name
			}
			dt.Version = base.Version
			dt.Description = base.Description
			dt.Tags = base.Tags
			if base.Vendor != nil {
				dt.Vendor = *base.Vendor
			}
		}

		// Enrich skill-specific fields.
		if r.Category == "skill" {
			parsed, parseErr := manifest.ParseFile(r.ManifestPath)
			if parseErr == nil {
				if skill, ok := parsed.(*manifest.SkillManifest); ok {
					dt.Topic = skill.Topic
					for _, dep := range skill.CLIDependencies {
						dt.CLIDeps = append(dt.CLIDeps, dep.Name)
					}
				}
			}
		}

		result = append(result, dt)
	}

	return result, nil
}

// DiscoverTypes walks all sources and returns all types with manifests.
// Types found in earlier sources take priority (later duplicates are skipped).
func DiscoverTypes(sources []Source) ([]*ResolvedType, error) {
	seen := make(map[string]bool)
	var result []*ResolvedType

	for _, src := range sources {
		types, err := walkSource(src)
		if err != nil {
			continue // skip inaccessible sources
		}
		for _, t := range types {
			if !seen[t.TypePath] {
				seen[t.TypePath] = true
				result = append(result, t)
			}
		}
	}

	return result, nil
}

// DiscoverByCategory returns types filtered by category (e.g., "skill", "persona").
func DiscoverByCategory(sources []Source, category string) ([]*ResolvedType, error) {
	all, err := DiscoverTypes(sources)
	if err != nil {
		return nil, err
	}

	var filtered []*ResolvedType
	for _, t := range all {
		if t.Category == category {
			filtered = append(filtered, t)
		}
	}
	return filtered, nil
}

// walkSource walks a single source directory and finds all types with manifests.
// It looks for manifest files inside known category directories at any nesting depth.
func walkSource(source Source) ([]*ResolvedType, error) {
	var result []*ResolvedType

	for _, cat := range knownCategories {
		catDir := filepath.Join(source.BasePath, cat)
		if _, err := os.Stat(catDir); err != nil {
			continue
		}

		err := filepath.WalkDir(catDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil // skip inaccessible entries
			}
			if d.IsDir() {
				return nil
			}

			// Check if this is a manifest file.
			name := d.Name()
			if !isManifestFile(name) {
				return nil
			}

			// Build the type path from the directory relative to source root.
			dir := filepath.Dir(path)
			relDir, err := filepath.Rel(source.BasePath, dir)
			if err != nil {
				return nil
			}

			typePath := typePathFromManifest(relDir)

			// Only use the highest-priority manifest in each directory.
			// If we already found a manifest for this type path, skip.
			for _, existing := range result {
				if existing.TypePath == typePath {
					return nil
				}
			}

			category := categoryFromPath(typePath)
			if category == "" {
				return nil
			}

			result = append(result, &ResolvedType{
				TypePath:     typePath,
				ManifestPath: path,
				SourceDir:    dir,
				SourceName:   source.Name,
				Category:     category,
			})

			return nil
		})
		if err != nil {
			continue
		}
	}

	return result, nil
}

// isManifestFile returns true if the filename is a recognized manifest file.
func isManifestFile(name string) bool {
	if name == "manifest.yaml" || name == "manifest.json" {
		return true
	}
	// Type-specific names: skill.yaml, persona.yaml, etc.
	if strings.HasSuffix(name, ".yaml") {
		base := strings.TrimSuffix(name, ".yaml")
		switch base {
		case "context", "persona", "skill", "workflow", "prompt", "template":
			return true
		}
	}
	return false
}
