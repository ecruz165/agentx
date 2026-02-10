package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/registry"
	"github.com/agentx-labs/agentx/internal/userdata"
	"github.com/spf13/cobra"
)

var (
	listTypeFilter string
	listJSON       bool
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed types",
	Long:  `List all types installed in ~/.agentx/installed/.`,
	RunE:  runList,
}

func init() {
	listCmd.Flags().StringVar(&listTypeFilter, "type", "", "Filter by type (skill, workflow, prompt, persona, context, template)")
	listCmd.Flags().BoolVar(&listJSON, "json", false, "Output in JSON format")
	rootCmd.AddCommand(listCmd)
}

// listEntry represents an installed type for display.
type listEntry struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	Version string `json:"version"`
}

func runList(cmd *cobra.Command, args []string) error {
	installedRoot, err := userdata.GetInstalledRoot()
	if err != nil {
		return fmt.Errorf("resolving installed root: %w", err)
	}

	if _, err := os.Stat(installedRoot); err != nil {
		fmt.Fprintln(cmd.OutOrStdout(), "No types installed yet.")
		return nil
	}

	sources := []registry.Source{{Name: "installed", BasePath: installedRoot}}
	types, err := registry.DiscoverTypes(sources)
	if err != nil {
		return fmt.Errorf("discovering installed types: %w", err)
	}

	if len(types) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No types installed yet.")
		return nil
	}

	// Build entries with manifest info.
	var entries []listEntry
	for _, t := range types {
		if listTypeFilter != "" && t.Category != listTypeFilter {
			continue
		}

		entry := listEntry{
			Type: t.Category,
			Name: registry.NameFromPath(t.TypePath),
			Path: t.TypePath,
		}

		// Try to read version from manifest.
		base, err := manifest.Parse(t.ManifestPath)
		if err == nil {
			entry.Version = base.Version
		}

		entries = append(entries, entry)
	}

	if len(entries) == 0 {
		if listTypeFilter != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "No installed types matching --type=%s\n", listTypeFilter)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "No types installed yet.")
		}
		return nil
	}

	if listJSON {
		return printListJSON(cmd, entries)
	}
	return printListTable(cmd, entries)
}

func printListTable(cmd *cobra.Command, entries []listEntry) error {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "TYPE\tNAME\tVERSION")
	for _, e := range entries {
		version := e.Version
		if version == "" {
			version = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", e.Type, e.Name, version)
	}
	return w.Flush()
}

func printListJSON(cmd *cobra.Command, entries []listEntry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return err
}
