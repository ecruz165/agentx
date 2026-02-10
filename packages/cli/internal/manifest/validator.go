package manifest

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"go.yaml.in/yaml/v3"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

//go:embed schema/manifest.schema.json
var schemaBytes []byte

var (
	compiledSchema *jsonschema.Schema
	compileOnce    sync.Once
	compileErr     error
	printer        = message.NewPrinter(language.English)
)

// ValidationResult contains the outcome of a schema validation.
type ValidationResult struct {
	Valid  bool
	Issues []ValidationIssue
}

// ValidationIssue represents a single validation error from the schema.
type ValidationIssue struct {
	Path    string // Instance location (e.g., "/name", "/steps/0/skill")
	Message string // Human-readable error message
	Keyword string // Schema keyword location that failed
}

// getSchema compiles the embedded JSON schema once and returns it.
func getSchema() (*jsonschema.Schema, error) {
	compileOnce.Do(func() {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(schemaBytes))
		if err != nil {
			compileErr = fmt.Errorf("unmarshaling schema JSON: %w", err)
			return
		}

		c := jsonschema.NewCompiler()
		if err := c.AddResource("manifest.schema.json", doc); err != nil {
			compileErr = fmt.Errorf("adding schema resource: %w", err)
			return
		}
		compiledSchema, compileErr = c.Compile("manifest.schema.json")
		if compileErr != nil {
			compileErr = fmt.Errorf("compiling schema: %w", compileErr)
		}
	})
	return compiledSchema, compileErr
}

// Validate validates raw YAML bytes against the manifest JSON schema.
// The error return is for I/O or schema compilation failures.
// Validation issues are returned in the ValidationResult.
func Validate(data []byte) (*ValidationResult, error) {
	schema, err := getSchema()
	if err != nil {
		return nil, fmt.Errorf("loading schema: %w", err)
	}

	// Unmarshal YAML to a generic structure.
	var raw interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parsing YAML: %w", err)
	}

	// Convert YAML maps to JSON-compatible types and marshal to JSON,
	// then unmarshal with json.Number support for the schema validator.
	raw = normalizeYAML(raw)
	jsonData, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("converting to JSON: %w", err)
	}

	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("preparing JSON for validation: %w", err)
	}

	// Validate against the schema.
	err = schema.Validate(inst)
	if err == nil {
		return &ValidationResult{Valid: true}, nil
	}

	// Extract validation issues using the BasicOutput format.
	validationErr, ok := err.(*jsonschema.ValidationError)
	if !ok {
		return nil, fmt.Errorf("unexpected validation error type: %w", err)
	}

	issues := extractIssues(validationErr)
	return &ValidationResult{
		Valid:  false,
		Issues: issues,
	}, nil
}

// ValidateFile reads a file and validates it against the manifest schema.
func ValidateFile(path string) (*ValidationResult, error) {
	data, err := readFile(path)
	if err != nil {
		return nil, err
	}
	return Validate(data)
}

// extractIssues walks the ValidationError tree and returns leaf-level issues.
// For oneOf schemas (discriminated unions), we walk all branches to collect
// specific property-level errors rather than just "oneOf failed".
func extractIssues(ve *jsonschema.ValidationError) []ValidationIssue {
	var issues []ValidationIssue
	collectValidationIssues(ve, &issues)

	// Deduplicate: oneOf branches produce many overlapping errors.
	if len(issues) == 0 {
		return []ValidationIssue{{
			Message: ve.Error(),
		}}
	}
	return deduplicateIssues(issues)
}

// collectValidationIssues recursively walks the error tree to find leaf errors
// with specific property information.
func collectValidationIssues(ve *jsonschema.ValidationError, issues *[]ValidationIssue) {
	if len(ve.Causes) == 0 {
		// Leaf error.
		path := "/" + strings.Join(ve.InstanceLocation, "/")
		if len(ve.InstanceLocation) == 0 {
			path = ""
		}

		keyword := ""
		if ve.ErrorKind != nil {
			kwPath := ve.ErrorKind.KeywordPath()
			if len(kwPath) > 0 {
				keyword = kwPath[len(kwPath)-1]
			}
		}

		msg := ""
		if ve.ErrorKind != nil {
			msg = ve.ErrorKind.LocalizedString(printer)
		}

		// Skip generic container errors that aren't informative.
		if keyword == "oneOf" || keyword == "allOf" || keyword == "$ref" || keyword == "" {
			return
		}

		*issues = append(*issues, ValidationIssue{
			Path:    path,
			Message: msg,
			Keyword: keyword,
		})
		return
	}

	for _, cause := range ve.Causes {
		collectValidationIssues(cause, issues)
	}
}

// deduplicateIssues removes duplicate issues (same path + keyword + message).
func deduplicateIssues(issues []ValidationIssue) []ValidationIssue {
	seen := make(map[string]bool)
	var result []ValidationIssue
	for _, issue := range issues {
		key := issue.Path + "|" + issue.Keyword + "|" + issue.Message
		if !seen[key] {
			seen[key] = true
			result = append(result, issue)
		}
	}
	return result
}

// normalizeYAML recursively converts YAML-decoded values to JSON-compatible types.
// YAML v3 may produce map[string]interface{} but also int/int64 that JSON Schema
// validators may not handle consistently — this normalizes them.
func normalizeYAML(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		m := make(map[string]interface{}, len(val))
		for k, v := range val {
			m[k] = normalizeYAML(v)
		}
		return m
	case []interface{}:
		a := make([]interface{}, len(val))
		for i, v := range val {
			a[i] = normalizeYAML(v)
		}
		return a
	case int:
		return val
	case int64:
		return val
	case float64:
		return val
	default:
		return val
	}
}
