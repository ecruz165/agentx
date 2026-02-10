// Package compose loads a prompt manifest and all its referenced types
// (persona, context, skills, workflows) to produce a unified text output.
package compose

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// ComposedPrompt holds the fully resolved content assembled from a prompt
// manifest and all of its referenced types.
type ComposedPrompt struct {
	// PromptName is the display name from the prompt manifest.
	PromptName string

	// Persona holds the resolved persona information, or nil if no persona
	// was referenced (or the persona could not be loaded).
	Persona *PersonaSection

	// Context holds the resolved context sections with their file content.
	Context []ContextSection

	// Skills holds references to the skills declared by the prompt.
	Skills []SkillRef

	// Workflows holds references to the workflows declared by the prompt.
	Workflows []WorkflowRef

	// Warnings collects non-fatal issues encountered during composition
	// (e.g. a missing persona or unreadable context file).
	Warnings []string
}

// PersonaSection contains the key fields from a resolved persona manifest.
type PersonaSection struct {
	Name        string
	Expertise   []string
	Tone        string
	Conventions []string
}

// ContextSection holds one block of context content, identified by name.
type ContextSection struct {
	Name    string
	Content string
}

// SkillRef is a lightweight reference to a skill used in display output.
type SkillRef struct {
	Name        string
	Description string
}

// WorkflowRef is a lightweight reference to a workflow used in display output.
type WorkflowRef struct {
	Name        string
	Description string
}

// Compose loads the prompt manifest at promptPath (relative type path such as
// "prompts/java-pr-review") from the installed root directory, resolves all
// referenced types, and assembles a ComposedPrompt.
func Compose(promptPath, installedRoot string) (*ComposedPrompt, error) {
	promptDir := filepath.Join(installedRoot, promptPath)

	manifestPath, err := findManifest(promptDir)
	if err != nil {
		return nil, fmt.Errorf("finding prompt manifest in %s: %w", promptDir, err)
	}

	pm, err := manifest.ParsePrompt(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("parsing prompt manifest: %w", err)
	}

	cp := &ComposedPrompt{
		PromptName: pm.Name,
	}

	// Resolve persona.
	if pm.Persona != "" {
		persona, warnings := loadPersona(pm.Persona, installedRoot)
		cp.Persona = persona
		cp.Warnings = append(cp.Warnings, warnings...)
	}

	// Resolve context blocks.
	for _, ctxPath := range pm.Context {
		sections, warnings := loadContext(ctxPath, installedRoot)
		cp.Context = append(cp.Context, sections...)
		cp.Warnings = append(cp.Warnings, warnings...)
	}

	// Resolve skills.
	for _, skillPath := range pm.Skills {
		ref, warnings := loadSkillRef(skillPath, installedRoot)
		if ref != nil {
			cp.Skills = append(cp.Skills, *ref)
		}
		cp.Warnings = append(cp.Warnings, warnings...)
	}

	// Resolve workflows.
	for _, wfPath := range pm.Workflows {
		ref, warnings := loadWorkflowRef(wfPath, installedRoot)
		if ref != nil {
			cp.Workflows = append(cp.Workflows, *ref)
		}
		cp.Warnings = append(cp.Warnings, warnings...)
	}

	return cp, nil
}

// Render formats a ComposedPrompt as a Markdown string suitable for piping
// to stdout or copying to the clipboard.
func Render(cp *ComposedPrompt) string {
	var b strings.Builder

	// Persona section.
	if cp.Persona != nil {
		b.WriteString("# Persona: ")
		b.WriteString(cp.Persona.Name)
		b.WriteString("\n")

		if len(cp.Persona.Expertise) > 0 {
			b.WriteString("Expertise: ")
			b.WriteString(strings.Join(cp.Persona.Expertise, ", "))
			b.WriteString("\n")
		}

		if cp.Persona.Tone != "" {
			b.WriteString("Tone: ")
			b.WriteString(cp.Persona.Tone)
			b.WriteString("\n")
		}

		if len(cp.Persona.Conventions) > 0 {
			b.WriteString("\n## Conventions\n")
			for _, conv := range cp.Persona.Conventions {
				b.WriteString("- ")
				b.WriteString(conv)
				b.WriteString("\n")
			}
		}
	}

	// Context section.
	if len(cp.Context) > 0 {
		b.WriteString("\n## Context\n")
		for _, ctx := range cp.Context {
			b.WriteString("\n### ")
			b.WriteString(ctx.Name)
			b.WriteString("\n")
			b.WriteString(ctx.Content)
			// Ensure trailing newline.
			if !strings.HasSuffix(ctx.Content, "\n") {
				b.WriteString("\n")
			}
		}
	}

	// Skills section.
	if len(cp.Skills) > 0 {
		b.WriteString("\n## Available Skills\n")
		for _, s := range cp.Skills {
			b.WriteString("- ")
			b.WriteString(s.Name)
			if s.Description != "" {
				b.WriteString(": ")
				b.WriteString(s.Description)
			}
			b.WriteString("\n")
		}
	}

	// Workflows section.
	if len(cp.Workflows) > 0 {
		b.WriteString("\n## Available Workflows\n")
		for _, w := range cp.Workflows {
			b.WriteString("- ")
			b.WriteString(w.Name)
			if w.Description != "" {
				b.WriteString(": ")
				b.WriteString(w.Description)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

// loadPersona reads the persona manifest from the installed root and returns
// a PersonaSection. Non-fatal errors produce warnings instead of failing.
func loadPersona(personaPath, installedRoot string) (*PersonaSection, []string) {
	var warnings []string

	personaDir := filepath.Join(installedRoot, personaPath)
	manifestPath, err := findManifest(personaDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("persona %q: %v", personaPath, err))
		return nil, warnings
	}

	pm, err := manifest.ParsePersona(manifestPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("persona %q: %v", personaPath, err))
		return nil, warnings
	}

	return &PersonaSection{
		Name:        pm.Name,
		Expertise:   pm.Expertise,
		Tone:        pm.Tone,
		Conventions: pm.Conventions,
	}, warnings
}

// loadContext reads the context manifest and all of its source files from the
// installed root. Each source file produces one ContextSection.
func loadContext(ctxPath, installedRoot string) ([]ContextSection, []string) {
	var sections []ContextSection
	var warnings []string

	ctxDir := filepath.Join(installedRoot, ctxPath)
	manifestPath, err := findManifest(ctxDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("context %q: %v", ctxPath, err))
		return nil, warnings
	}

	cm, err := manifest.ParseContext(manifestPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("context %q: %v", ctxPath, err))
		return nil, warnings
	}

	for _, src := range cm.Sources {
		srcPath := filepath.Join(ctxDir, src)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("context %q source %q: %v", ctxPath, src, err))
			continue
		}

		// Derive a human-readable name from the context manifest name.
		name := formatContextName(cm.Name)

		sections = append(sections, ContextSection{
			Name:    name,
			Content: string(data),
		})
	}

	return sections, warnings
}

// loadSkillRef reads just enough of the skill manifest to produce a SkillRef.
func loadSkillRef(skillPath, installedRoot string) (*SkillRef, []string) {
	var warnings []string

	skillDir := filepath.Join(installedRoot, skillPath)
	manifestPath, err := findManifest(skillDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("skill %q: %v", skillPath, err))
		return nil, warnings
	}

	base, err := manifest.Parse(manifestPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("skill %q: %v", skillPath, err))
		return nil, warnings
	}

	return &SkillRef{
		Name:        base.Name,
		Description: base.Description,
	}, warnings
}

// loadWorkflowRef reads just enough of the workflow manifest to produce a WorkflowRef.
func loadWorkflowRef(wfPath, installedRoot string) (*WorkflowRef, []string) {
	var warnings []string

	wfDir := filepath.Join(installedRoot, wfPath)
	manifestPath, err := findManifest(wfDir)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("workflow %q: %v", wfPath, err))
		return nil, warnings
	}

	base, err := manifest.Parse(manifestPath)
	if err != nil {
		warnings = append(warnings, fmt.Sprintf("workflow %q: %v", wfPath, err))
		return nil, warnings
	}

	return &WorkflowRef{
		Name:        base.Name,
		Description: base.Description,
	}, warnings
}

// findManifest searches a directory for a manifest file and returns its path.
// It tries manifest.yaml first, then manifest.json, then type-specific names.
func findManifest(dir string) (string, error) {
	candidates := []string{"manifest.yaml", "manifest.json"}
	for _, typeName := range manifest.ValidTypes {
		candidates = append(candidates, typeName+".yaml")
	}

	for _, name := range candidates {
		p := filepath.Join(dir, name)
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}

	return "", fmt.Errorf("no manifest found in %s", dir)
}

// formatContextName converts a manifest name like "error-handling" to a
// title-case display name like "Error Handling".
func formatContextName(name string) string {
	words := strings.Split(name, "-")
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + w[1:]
		}
	}
	return strings.Join(words, " ")
}
