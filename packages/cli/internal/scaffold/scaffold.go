package scaffold

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/agentx-labs/agentx/internal/manifest"
)

// ScaffoldData holds all template variables available to scaffold templates.
type ScaffoldData struct {
	Name        string // e.g., "ssm-lookup"
	Topic       string // e.g., "cloud" (skills only)
	Vendor      string // e.g., "aws" (skills only, may be empty)
	Runtime     string // "node" or "go" (skills only)
	Description string // Human-readable description
	Version     string // Semver, e.g., "0.1.0"
	PackageName string // Derived: @agentx/skill-<topic>-<name>
	SkillPath   string // Derived: <topic>/<vendor>/<name> or <topic>/<name>
	ModuleName  string // Derived: github.com/agentx-labs/agentx-skill-<name>
	Year        int    // Current year
}

// Result holds the outcome of a scaffold generation.
type Result struct {
	OutputDir string
	Files     []string
	Warnings  []string
}

// NewScaffoldData creates a ScaffoldData with derived fields populated.
func NewScaffoldData(name, typeName, topic, vendor, runtime string) *ScaffoldData {
	d := &ScaffoldData{
		Name:    name,
		Topic:   topic,
		Vendor:  vendor,
		Runtime: runtime,
		Version: "0.1.0",
		Year:    time.Now().Year(),
	}

	d.Description = fmt.Sprintf("AgentX %s: %s", typeName, name)

	// Derive skill-specific fields.
	if topic != "" {
		if vendor != "" {
			d.SkillPath = fmt.Sprintf("%s/%s/%s", topic, vendor, name)
			d.PackageName = fmt.Sprintf("@agentx/skill-%s-%s", topic, name)
		} else {
			d.SkillPath = fmt.Sprintf("%s/%s", topic, name)
			d.PackageName = fmt.Sprintf("@agentx/skill-%s-%s", topic, name)
		}
	}

	d.ModuleName = fmt.Sprintf("github.com/agentx-labs/agentx-skill-%s", name)

	return d
}

// templateSetName returns the embedded directory name for a given type+runtime.
func templateSetName(typeName, runtime string) string {
	if typeName == "skill" {
		return "skill-" + runtime
	}
	return typeName
}

// manifestFileName returns the expected manifest file name for a type.
func manifestFileName(typeName string) string {
	return typeName + ".yaml"
}

// Generate creates a new type from scaffolding templates.
func Generate(typeName string, data *ScaffoldData, outputDir string) (*Result, error) {
	setName := templateSetName(typeName, data.Runtime)
	templatesDir := filepath.Join("scaffolds", setName)

	// Verify template set exists in embedded FS.
	entries, err := fs.ReadDir(scaffoldFS, templatesDir)
	if err != nil {
		return nil, fmt.Errorf("template set %q not found: %w", setName, err)
	}

	// Create output directory.
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	// Check for existing files to prevent accidental overwrites.
	existingEntries, err := os.ReadDir(outputDir)
	if err == nil && len(existingEntries) > 0 {
		return nil, fmt.Errorf("output directory %s is not empty; remove existing files first", outputDir)
	}

	result := &Result{
		OutputDir: outputDir,
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		tmplPath := filepath.Join(templatesDir, entry.Name())
		tmplBytes, err := fs.ReadFile(scaffoldFS, tmplPath)
		if err != nil {
			return nil, fmt.Errorf("reading template %s: %w", tmplPath, err)
		}

		// Strip .tmpl extension for the output filename.
		outName := strings.TrimSuffix(entry.Name(), ".tmpl")
		outPath := filepath.Join(outputDir, outName)

		// Handlebars (.hbs) files use {{ }} syntax that conflicts with
		// Go's text/template. Copy them verbatim without processing.
		if strings.HasSuffix(outName, ".hbs") {
			if err := os.WriteFile(outPath, tmplBytes, 0644); err != nil {
				return nil, fmt.Errorf("writing %s: %w", outPath, err)
			}
			result.Files = append(result.Files, outName)
			continue
		}

		// Parse and execute the Go template.
		tmpl, err := template.New(entry.Name()).Parse(string(tmplBytes))
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", entry.Name(), err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return nil, fmt.Errorf("executing template %s: %w", entry.Name(), err)
		}

		if err := os.WriteFile(outPath, buf.Bytes(), 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", outPath, err)
		}

		result.Files = append(result.Files, outName)
	}

	// Validate the generated manifest against JSON Schema.
	manifestFile := filepath.Join(outputDir, manifestFileName(typeName))
	if _, err := os.Stat(manifestFile); err == nil {
		valResult, valErr := manifest.ValidateFile(manifestFile)
		if valErr != nil {
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Could not validate manifest: %v", valErr))
		} else if !valResult.Valid {
			for _, issue := range valResult.Issues {
				msg := issue.Message
				if issue.Path != "" {
					msg = issue.Path + ": " + msg
				}
				result.Warnings = append(result.Warnings, msg)
			}
		}
	}

	return result, nil
}
