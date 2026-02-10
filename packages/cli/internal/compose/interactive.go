package compose

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
	"github.com/agentx-labs/agentx/internal/registry"
)

// InteractiveResult holds the selections made during interactive prompt composition.
type InteractiveResult struct {
	PersonaPath string
	Topic       string
	Intent      string
}

// RunInteractive walks the user through persona, topic, and intent selection
// using numbered menus on stdin/stdout. It returns the selections, or an error
// if input is invalid or the installed root has no types.
func RunInteractive(installedRoot string, r io.Reader, w io.Writer) (*InteractiveResult, error) {
	reader := bufio.NewReader(r)

	// Discover installed types.
	sources := []registry.Source{
		{Name: "installed", BasePath: installedRoot},
	}

	// Step 1: Select persona.
	personas, err := registry.DiscoverByCategory(sources, "persona")
	if err != nil {
		return nil, fmt.Errorf("discovering personas: %w", err)
	}
	if len(personas) == 0 {
		return nil, fmt.Errorf("no installed personas found; install some first with `agentx install`")
	}

	personaIdx, err := selectFromList(reader, w, "Select persona:", resolvedTypeNames(personas))
	if err != nil {
		return nil, err
	}
	selectedPersona := personas[personaIdx]

	// Step 2: Select topic from installed skills.
	skills, err := registry.DiscoverByCategory(sources, "skill")
	if err != nil {
		return nil, fmt.Errorf("discovering skills: %w", err)
	}

	topics := uniqueTopics(skills, installedRoot)
	if len(topics) == 0 {
		topics = []string{"general"}
	}

	topicIdx, err := selectFromList(reader, w, "Select topic:", topics)
	if err != nil {
		return nil, err
	}
	selectedTopic := topics[topicIdx]

	// Step 3: Enter intent.
	fmt.Fprintf(w, "\nEnter intent (e.g., code-review, migration, incident-triage): ")
	intent, err := reader.ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("reading intent: %w", err)
	}
	intent = strings.TrimSpace(intent)
	if intent == "" {
		intent = "general-guidance"
	}

	return &InteractiveResult{
		PersonaPath: selectedPersona.TypePath,
		Topic:       selectedTopic,
		Intent:      intent,
	}, nil
}

// ComposeFromInteractive builds a ComposedPrompt from interactive selections
// instead of from a prompt manifest.
func ComposeFromInteractive(result *InteractiveResult, installedRoot string) (*ComposedPrompt, error) {
	cp := &ComposedPrompt{
		PromptName: fmt.Sprintf("%s %s", result.Topic, result.Intent),
	}

	// Load persona.
	persona, warnings := loadPersona(result.PersonaPath, installedRoot)
	cp.Persona = persona
	cp.Warnings = append(cp.Warnings, warnings...)

	// Discover installed types.
	sources := []registry.Source{
		{Name: "installed", BasePath: installedRoot},
	}

	// Find context matching the topic.
	contexts, _ := registry.DiscoverByCategory(sources, "context")
	for _, ctx := range contexts {
		if matchesTopic(ctx, result.Topic, installedRoot) {
			sections, w := loadContext(ctx.TypePath, installedRoot)
			cp.Context = append(cp.Context, sections...)
			cp.Warnings = append(cp.Warnings, w...)
		}
	}

	// Find skills matching the topic.
	skills, _ := registry.DiscoverByCategory(sources, "skill")
	for _, s := range skills {
		if matchesTopic(s, result.Topic, installedRoot) {
			ref, w := loadSkillRef(s.TypePath, installedRoot)
			if ref != nil {
				cp.Skills = append(cp.Skills, *ref)
			}
			cp.Warnings = append(cp.Warnings, w...)
		}
	}

	return cp, nil
}

// selectFromList presents a numbered list and returns the selected index.
func selectFromList(reader *bufio.Reader, w io.Writer, prompt string, items []string) (int, error) {
	fmt.Fprintf(w, "\n%s\n", prompt)
	for i, item := range items {
		fmt.Fprintf(w, "  %d) %s\n", i+1, item)
	}
	fmt.Fprintf(w, "Enter number [1-%d]: ", len(items))

	line, err := reader.ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("reading selection: %w", err)
	}

	num, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || num < 1 || num > len(items) {
		return 0, fmt.Errorf("invalid selection %q: choose 1-%d", strings.TrimSpace(line), len(items))
	}

	return num - 1, nil
}

// resolvedTypeNames extracts display names from resolved types.
func resolvedTypeNames(types []*registry.ResolvedType) []string {
	names := make([]string, len(types))
	for i, t := range types {
		names[i] = registry.NameFromPath(t.TypePath)
	}
	return names
}

// uniqueTopics extracts unique topic values from skill manifests.
func uniqueTopics(skills []*registry.ResolvedType, installedRoot string) []string {
	seen := make(map[string]bool)
	var topics []string

	for _, s := range skills {
		parsed, err := manifest.ParseFile(s.ManifestPath)
		if err != nil {
			continue
		}
		skill, ok := parsed.(*manifest.SkillManifest)
		if !ok || skill.Topic == "" {
			continue
		}
		if !seen[skill.Topic] {
			seen[skill.Topic] = true
			topics = append(topics, skill.Topic)
		}
	}

	return topics
}

// matchesTopic checks if a resolved type's path or manifest topic matches the given topic.
func matchesTopic(rt *registry.ResolvedType, topic string, installedRoot string) bool {
	// Check if the type path contains the topic.
	if strings.Contains(rt.TypePath, topic) {
		return true
	}

	// For skills, check the manifest topic field.
	if rt.Category == "skill" {
		parsed, err := manifest.ParseFile(rt.ManifestPath)
		if err != nil {
			return false
		}
		skill, ok := parsed.(*manifest.SkillManifest)
		if ok && skill.Topic == topic {
			return true
		}
	}

	// For context, check if the type path segment matches.
	if rt.Category == "context" {
		parts := strings.Split(rt.TypePath, "/")
		for _, p := range parts {
			if p == topic {
				return true
			}
		}
	}

	return false
}

// ParseManifest reads and parses a manifest from a file path for use in interactive mode.
// It is exported for testing.
func ParseManifest(path string) (*manifest.BaseManifest, error) {
	return manifest.Parse(path)
}

// ListInstalledPrompts returns the type paths of all installed prompt types.
func ListInstalledPrompts(installedRoot string) ([]string, error) {
	sources := []registry.Source{
		{Name: "installed", BasePath: installedRoot},
	}

	prompts, err := registry.DiscoverByCategory(sources, "prompt")
	if err != nil {
		return nil, err
	}

	paths := make([]string, len(prompts))
	for i, p := range prompts {
		paths[i] = p.TypePath
	}
	return paths, nil
}

// isTerminal checks if the given file is a terminal (for auto-detecting interactive mode).
func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}

// FindInstalledPromptDir checks if a prompt manifest directory exists under installedRoot.
func FindInstalledPromptDir(installedRoot, promptPath string) (string, error) {
	dir := filepath.Join(installedRoot, promptPath)
	if _, err := os.Stat(dir); err != nil {
		return "", fmt.Errorf("prompt type %q is not installed", promptPath)
	}
	return dir, nil
}
