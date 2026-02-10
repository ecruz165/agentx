package registry

import "io"

// Source represents a location to search for types (e.g., catalog, extension).
type Source struct {
	Name     string // e.g., "catalog", "acme-corp"
	BasePath string // absolute path to the source root
}

// ResolvedType represents a type found in a source.
type ResolvedType struct {
	TypePath     string // e.g., "personas/senior-java-dev"
	ManifestPath string // absolute path to manifest file
	SourceDir    string // absolute path to the type directory
	SourceName   string // name of the source it was found in
	Category     string // "context", "persona", "skill", "workflow", "prompt", "template"
}

// DependencyNode represents a node in the dependency tree.
type DependencyNode struct {
	TypePath  string
	Category  string
	Resolved  *ResolvedType
	Children  []*DependencyNode
	Deduped   bool // true if this type was already seen earlier in the tree
	Installed bool // true if already in ~/.agentx/installed/
}

// CLIDepStatus represents whether a CLI dependency is available on the system.
type CLIDepStatus struct {
	Name      string
	Available bool
}

// InstallPlan summarizes what will be installed.
type InstallPlan struct {
	Root      *DependencyNode
	AllTypes  []*ResolvedType // flattened, deduplicated, topologically ordered
	Counts    map[string]int  // count per category
	CLIDeps   []CLIDepStatus  // CLI dependency check results
	SkipCount int             // already-installed count
}

// InstallResult captures the outcome of an install operation.
type InstallResult struct {
	Installed int
	Skipped   int
	Warnings  []string
}

// ConfirmFunc is called to confirm installation. Returns true to proceed.
type ConfirmFunc func() (bool, error)

// ProgressFunc is called to report progress during installation.
type ProgressFunc func(w io.Writer, category, name string, err error)
