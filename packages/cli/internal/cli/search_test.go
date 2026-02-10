package cli

import (
	"testing"

	"github.com/agentx-labs/agentx/internal/registry"
)

func TestMatchesSearchByQuery(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath:    "skills/scm/git/commit-analyzer",
		Category:    "skill",
		Name:        "commit-analyzer",
		Version:     "1.0.0",
		Description: "Analyzes git commit history",
		Tags:        []string{"git", "scm"},
		Source:      "catalog",
	}

	tests := []struct {
		name     string
		query    string
		expected bool
	}{
		{"empty query matches all", "", true},
		{"exact name match", "commit-analyzer", true},
		{"partial name match", "commit", true},
		{"case insensitive name", "COMMIT", true},
		{"description match", "git commit", true},
		{"description partial", "analyzes", true},
		{"type path match", "scm/git", true},
		{"no match", "nonexistent-thing", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, tt.query, "", nil, "", "", "")
			if got != tt.expected {
				t.Errorf("matchesSearch(query=%q) = %v, want %v", tt.query, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByType(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "skills/test/basic-skill",
		Category: "skill",
		Name:     "basic-skill",
	}

	tests := []struct {
		name       string
		typeFilter string
		expected   bool
	}{
		{"no type filter", "", true},
		{"matching type", "skill", true},
		{"non-matching type", "persona", false},
		{"non-matching type workflow", "workflow", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, "", tt.typeFilter, nil, "", "", "")
			if got != tt.expected {
				t.Errorf("matchesSearch(type=%q) = %v, want %v", tt.typeFilter, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByTag(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "skills/scm/git/commit-analyzer",
		Category: "skill",
		Name:     "commit-analyzer",
		Tags:     []string{"git", "scm", "analysis"},
	}

	tests := []struct {
		name       string
		filterTags []string
		expected   bool
	}{
		{"no tag filter", nil, true},
		{"empty tag filter", []string{}, true},
		{"matching single tag", []string{"git"}, true},
		{"matching second tag", []string{"scm"}, true},
		{"case insensitive tag", []string{"GIT"}, true},
		{"one of multiple tags matches", []string{"nonexistent", "git"}, true},
		{"no matching tag", []string{"java", "spring"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, "", "", tt.filterTags, "", "", "")
			if got != tt.expected {
				t.Errorf("matchesSearch(tags=%v) = %v, want %v", tt.filterTags, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchNoTags(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "personas/senior-java-dev",
		Category: "persona",
		Name:     "senior-java-dev",
		Tags:     nil,
	}

	// A type with no tags should not match a tag filter.
	got := matchesSearch(dt, "", "", []string{"java"}, "", "", "")
	if got {
		t.Error("type with no tags should not match a tag filter")
	}

	// But it should match when there's no tag filter.
	got = matchesSearch(dt, "", "", nil, "", "", "")
	if !got {
		t.Error("type with no tags should match when no tag filter is set")
	}
}

func TestMatchesSearchCombined(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath:    "skills/scm/git/commit-analyzer",
		Category:    "skill",
		Name:        "commit-analyzer",
		Description: "Analyzes git commit history",
		Tags:        []string{"git", "scm"},
	}

	// All filters match.
	got := matchesSearch(dt, "commit", "skill", []string{"git"}, "", "", "")
	if !got {
		t.Error("expected match when all filters match")
	}

	// Query matches but type doesn't.
	got = matchesSearch(dt, "commit", "persona", []string{"git"}, "", "", "")
	if got {
		t.Error("expected no match when type filter doesn't match")
	}

	// Query matches, type matches, but tag doesn't.
	got = matchesSearch(dt, "commit", "skill", []string{"java"}, "", "", "")
	if got {
		t.Error("expected no match when tag filter doesn't match")
	}

	// Type and tag match, but query doesn't.
	got = matchesSearch(dt, "nonexistent", "skill", []string{"git"}, "", "", "")
	if got {
		t.Error("expected no match when query doesn't match")
	}
}

func TestMatchesAnyTag(t *testing.T) {
	tests := []struct {
		name       string
		typeTags   []string
		filterTags []string
		expected   bool
	}{
		{"both empty", nil, nil, false},
		{"no type tags", nil, []string{"git"}, false},
		{"no filter tags", []string{"git"}, nil, false},
		{"single match", []string{"git"}, []string{"git"}, true},
		{"case insensitive", []string{"Git"}, []string{"git"}, true},
		{"partial overlap", []string{"git", "scm"}, []string{"java", "git"}, true},
		{"no overlap", []string{"git", "scm"}, []string{"java", "spring"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesAnyTag(tt.typeTags, tt.filterTags)
			if got != tt.expected {
				t.Errorf("matchesAnyTag(%v, %v) = %v, want %v", tt.typeTags, tt.filterTags, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByTopic(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "skills/scm/git/commit-analyzer",
		Category: "skill",
		Name:     "commit-analyzer",
		Topic:    "scm",
	}

	tests := []struct {
		name        string
		topicFilter string
		expected    bool
	}{
		{"no topic filter", "", true},
		{"matching topic", "scm", true},
		{"case insensitive topic", "SCM", true},
		{"non-matching topic", "cloud", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, "", "", nil, tt.topicFilter, "", "")
			if got != tt.expected {
				t.Errorf("matchesSearch(topic=%q) = %v, want %v", tt.topicFilter, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByVendor(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "skills/cloud/aws/ssm-lookup",
		Category: "skill",
		Name:     "ssm-lookup",
		Vendor:   "aws",
	}

	tests := []struct {
		name         string
		vendorFilter string
		expected     bool
	}{
		{"no vendor filter", "", true},
		{"matching vendor", "aws", true},
		{"case insensitive vendor", "AWS", true},
		{"non-matching vendor", "gcp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, "", "", nil, "", tt.vendorFilter, "")
			if got != tt.expected {
				t.Errorf("matchesSearch(vendor=%q) = %v, want %v", tt.vendorFilter, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByCLI(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "skills/scm/git/commit-analyzer",
		Category: "skill",
		Name:     "commit-analyzer",
		CLIDeps:  []string{"git", "gh"},
	}

	tests := []struct {
		name      string
		cliFilter string
		expected  bool
	}{
		{"no cli filter", "", true},
		{"matching cli", "git", true},
		{"matching second cli", "gh", true},
		{"case insensitive cli", "GIT", true},
		{"non-matching cli", "aws", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSearch(dt, "", "", nil, "", "", tt.cliFilter)
			if got != tt.expected {
				t.Errorf("matchesSearch(cli=%q) = %v, want %v", tt.cliFilter, got, tt.expected)
			}
		})
	}
}

func TestMatchesSearchByVendor_NoVendor(t *testing.T) {
	dt := registry.DiscoveredType{
		TypePath: "personas/senior-java-dev",
		Category: "persona",
		Name:     "senior-java-dev",
		Vendor:   "",
	}

	// A type with no vendor should not match a vendor filter.
	got := matchesSearch(dt, "", "", nil, "", "aws", "")
	if got {
		t.Error("type with no vendor should not match a vendor filter")
	}

	// But it should match when there's no vendor filter.
	got = matchesSearch(dt, "", "", nil, "", "", "")
	if !got {
		t.Error("type with no vendor should match when no vendor filter is set")
	}
}

func TestMatchesCLIDep(t *testing.T) {
	tests := []struct {
		name     string
		deps     []string
		filter   string
		expected bool
	}{
		{"no deps", nil, "git", false},
		{"matching dep", []string{"git"}, "git", true},
		{"case insensitive", []string{"Git"}, "git", true},
		{"no match", []string{"git"}, "aws", false},
		{"multiple deps match", []string{"git", "gh", "aws"}, "aws", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesCLIDep(tt.deps, tt.filter)
			if got != tt.expected {
				t.Errorf("matchesCLIDep(%v, %q) = %v, want %v", tt.deps, tt.filter, got, tt.expected)
			}
		})
	}
}
