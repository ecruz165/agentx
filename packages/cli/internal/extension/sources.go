package extension

import (
	"path/filepath"

	"github.com/agentx-labs/agentx/internal/registry"
)

// BuildSources expands the resolution order from project.yaml into a slice of
// registry.Source entries. The abstract token "extensions" is expanded into
// one Source per declared extension in their array order. "catalog" maps to
// <repoRoot>/catalog, and "local" maps to the current working directory.
func BuildSources(cfg *ExtensionConfig, repoRoot string) []registry.Source {
	var sources []registry.Source

	for _, entry := range cfg.Resolution.Order {
		switch entry {
		case "local":
			sources = append(sources, registry.Source{
				Name:     "local",
				BasePath: ".",
			})

		case "catalog":
			sources = append(sources, registry.Source{
				Name:     "catalog",
				BasePath: filepath.Join(repoRoot, "catalog"),
			})

		case "extensions":
			// Expand "extensions" into individual extension sources
			// in their declared order.
			for _, ext := range cfg.Extensions {
				basePath := ext.Path
				if basePath == "" {
					basePath = filepath.Join(repoRoot, "extensions", ext.Name)
				}
				if !filepath.IsAbs(basePath) {
					basePath = filepath.Join(repoRoot, basePath)
				}
				sources = append(sources, registry.Source{
					Name:     ext.Name,
					BasePath: basePath,
				})
			}

		default:
			// Unknown entries are passed through as-is; the caller
			// can decide what to do with them.
			sources = append(sources, registry.Source{
				Name:     entry,
				BasePath: filepath.Join(repoRoot, entry),
			})
		}
	}

	return sources
}
