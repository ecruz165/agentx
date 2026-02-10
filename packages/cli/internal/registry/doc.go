// Package registry handles type discovery, indexing, dependency resolution,
// and installation for AgentX types. It scans source directories (catalog and
// extensions) for manifests, builds dependency trees, plans installations, and
// copies resolved types into the user's installed directory.
package registry