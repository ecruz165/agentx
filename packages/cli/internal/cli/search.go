package cli

import (
	"encoding/json"
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/spf13/cobra"
)

var (
	searchTypeFilter   string
	searchTagFilter    string
	searchTopicFilter  string
	searchVendorFilter string
	searchCLIFilter    string
	searchJSON         bool
)

var searchCmd = &cobra.Command{
	Use:   "search [query]",
	Short: "Search for available types across all sources",
	Long: `Search for types (skills, workflows, prompts, personas, context, templates)
across all available sources (catalog, extensions, installed).

The query matches against type names and descriptions (case-insensitive substring).
Use --type to filter by category and --tag to filter by tags.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSearch,
}

func init() {
	searchCmd.Flags().StringVar(&searchTypeFilter, "type", "", "Filter by type (skill, workflow, prompt, persona, context, template)")
	searchCmd.Flags().StringVar(&searchTagFilter, "tag", "", "Filter by tags (comma-separated, matches any)")
	searchCmd.Flags().StringVar(&searchTopicFilter, "topic", "", "Filter by topic (e.g., scm, cicd, cloud)")
	searchCmd.Flags().StringVar(&searchVendorFilter, "vendor", "", "Filter by vendor (e.g., aws, github)")
	searchCmd.Flags().StringVar(&searchCLIFilter, "cli", "", "Filter by CLI dependency (e.g., git, aws)")
	searchCmd.Flags().BoolVar(&searchJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(searchCmd)
}

// searchEntry represents a discovered type for display.
type searchEntry struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Tags        []string `json:"tags,omitempty"`
	Source      string   `json:"source"`
	Topic       string   `json:"topic,omitempty"`
	Vendor      string   `json:"vendor,omitempty"`
	CLIDeps     []string `json:"cli_deps,omitempty"`
}

func runSearch(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = args[0]
	}

	sources, err := buildSources()
	if err != nil {
		return fmt.Errorf("building sources: %w", err)
	}

	cachePath, _ := registry.DefaultCachePath()
	discovered, err := registry.DiscoverAllCached(sources, cachePath)
	if err != nil {
		return fmt.Errorf("discovering types: %w", err)
	}

	// Parse tag filter into a set.
	var filterTags []string
	if searchTagFilter != "" {
		for _, t := range strings.Split(searchTagFilter, ",") {
			tag := strings.TrimSpace(t)
			if tag != "" {
				filterTags = append(filterTags, strings.ToLower(tag))
			}
		}
	}

	// Filter results.
	var entries []searchEntry
	for _, dt := range discovered {
		if !matchesSearch(dt, query, searchTypeFilter, filterTags, searchTopicFilter, searchVendorFilter, searchCLIFilter) {
			continue
		}

		entries = append(entries, searchEntry{
			Type:        dt.Category,
			Name:        dt.Name,
			Version:     dt.Version,
			Description: dt.Description,
			Tags:        dt.Tags,
			Source:      dt.Source,
			Topic:       dt.Topic,
			Vendor:      dt.Vendor,
			CLIDeps:     dt.CLIDeps,
		})
	}

	if len(entries) == 0 {
		msg := "No types found"
		if query != "" {
			msg += fmt.Sprintf(" matching %q", query)
		}
		if searchTypeFilter != "" {
			msg += fmt.Sprintf(" with --type=%s", searchTypeFilter)
		}
		if searchTagFilter != "" {
			msg += fmt.Sprintf(" with --tag=%s", searchTagFilter)
		}
		if searchTopicFilter != "" {
			msg += fmt.Sprintf(" with --topic=%s", searchTopicFilter)
		}
		if searchVendorFilter != "" {
			msg += fmt.Sprintf(" with --vendor=%s", searchVendorFilter)
		}
		if searchCLIFilter != "" {
			msg += fmt.Sprintf(" with --cli=%s", searchCLIFilter)
		}
		fmt.Fprintln(cmd.OutOrStdout(), msg)
		return nil
	}

	if searchJSON {
		return printSearchJSON(cmd, entries)
	}
	return printSearchTable(cmd, entries)
}

// matchesSearch returns true if the discovered type matches all provided filters.
// All filters are AND-combined: the type must match every non-empty filter.
func matchesSearch(dt registry.DiscoveredType, query, typeFilter string, filterTags []string, topicFilter, vendorFilter, cliFilter string) bool {
	// Filter by type category.
	if typeFilter != "" && dt.Category != typeFilter {
		return false
	}

	// Filter by tags (match any).
	if len(filterTags) > 0 {
		if !matchesAnyTag(dt.Tags, filterTags) {
			return false
		}
	}

	// Filter by topic (case-insensitive exact match).
	if topicFilter != "" {
		if !strings.EqualFold(dt.Topic, topicFilter) {
			return false
		}
	}

	// Filter by vendor (case-insensitive exact match).
	if vendorFilter != "" {
		if !strings.EqualFold(dt.Vendor, vendorFilter) {
			return false
		}
	}

	// Filter by CLI dependency (case-insensitive, matches if any dep matches).
	if cliFilter != "" {
		if !matchesCLIDep(dt.CLIDeps, cliFilter) {
			return false
		}
	}

	// Filter by query (substring match on name, description, or type path).
	if query != "" {
		q := strings.ToLower(query)
		nameLower := strings.ToLower(dt.Name)
		descLower := strings.ToLower(dt.Description)
		pathLower := strings.ToLower(dt.TypePath)
		if !strings.Contains(nameLower, q) &&
			!strings.Contains(descLower, q) &&
			!strings.Contains(pathLower, q) {
			return false
		}
	}

	return true
}

// matchesCLIDep returns true if any CLI dependency matches the filter (case-insensitive).
func matchesCLIDep(deps []string, filter string) bool {
	filterLower := strings.ToLower(filter)
	for _, dep := range deps {
		if strings.ToLower(dep) == filterLower {
			return true
		}
	}
	return false
}

// matchesAnyTag returns true if any of the type's tags match any of the filter tags.
// Comparison is case-insensitive.
func matchesAnyTag(typeTags []string, filterTags []string) bool {
	for _, ft := range filterTags {
		ftLower := strings.ToLower(ft)
		for _, tt := range typeTags {
			if strings.ToLower(tt) == ftLower {
				return true
			}
		}
	}
	return false
}

func printSearchTable(cmd *cobra.Command, entries []searchEntry) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TYPE\tNAME\tVERSION\tDESCRIPTION")
	for _, e := range entries {
		version := e.Version
		if version == "" {
			version = "-"
		}
		desc := e.Description
		if len(desc) > 60 {
			desc = desc[:57] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.Type, e.Name, version, desc)
	}
	return w.Flush()
}

func printSearchJSON(cmd *cobra.Command, entries []searchEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}
