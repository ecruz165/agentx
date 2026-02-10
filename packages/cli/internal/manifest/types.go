package manifest

// BaseManifest contains fields shared by all manifest types.
type BaseManifest struct {
	Name        string   `yaml:"name" json:"name"`
	Type        string   `yaml:"type" json:"type"`
	Version     string   `yaml:"version" json:"version"`
	Description string   `yaml:"description" json:"description"`
	Tags        []string `yaml:"tags,omitempty" json:"tags,omitempty"`
	Author      string   `yaml:"author,omitempty" json:"author,omitempty"`
	Vendor      *string  `yaml:"vendor,omitempty" json:"vendor,omitempty"`
}

// ContextManifest represents a context type manifest.
type ContextManifest struct {
	BaseManifest `yaml:",inline"`
	Format       string   `yaml:"format" json:"format"`
	Tokens       int      `yaml:"tokens,omitempty" json:"tokens,omitempty"`
	Sources      []string `yaml:"sources" json:"sources"`
}

// PersonaManifest represents a persona type manifest.
type PersonaManifest struct {
	BaseManifest `yaml:",inline"`
	Expertise    []string `yaml:"expertise,omitempty" json:"expertise,omitempty"`
	Tone         string   `yaml:"tone,omitempty" json:"tone,omitempty"`
	Conventions  []string `yaml:"conventions,omitempty" json:"conventions,omitempty"`
	Context      []string `yaml:"context,omitempty" json:"context,omitempty"`
	Template     string   `yaml:"template,omitempty" json:"template,omitempty"`
}

// SkillManifest represents a skill type manifest.
type SkillManifest struct {
	BaseManifest    `yaml:",inline"`
	Runtime         string             `yaml:"runtime" json:"runtime"`
	Topic           string             `yaml:"topic" json:"topic"`
	CLIDependencies []CLIDependency    `yaml:"cli_dependencies,omitempty" json:"cli_dependencies,omitempty"`
	Inputs          []InputField       `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs         *OutputDeclaration `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Registry        *RegistryBlock     `yaml:"registry,omitempty" json:"registry,omitempty"`
}

// WorkflowManifest represents a workflow type manifest.
type WorkflowManifest struct {
	BaseManifest `yaml:",inline"`
	Runtime      string             `yaml:"runtime" json:"runtime"`
	Steps        []WorkflowStep     `yaml:"steps" json:"steps"`
	Inputs       []InputField       `yaml:"inputs,omitempty" json:"inputs,omitempty"`
	Outputs      *OutputDeclaration `yaml:"outputs,omitempty" json:"outputs,omitempty"`
}

// PromptManifest represents a prompt type manifest.
type PromptManifest struct {
	BaseManifest `yaml:",inline"`
	Persona      string   `yaml:"persona,omitempty" json:"persona,omitempty"`
	Context      []string `yaml:"context,omitempty" json:"context,omitempty"`
	Skills       []string `yaml:"skills,omitempty" json:"skills,omitempty"`
	Workflows    []string `yaml:"workflows,omitempty" json:"workflows,omitempty"`
	Template     string   `yaml:"template,omitempty" json:"template,omitempty"`
}

// TemplateManifest represents a template type manifest.
type TemplateManifest struct {
	BaseManifest `yaml:",inline"`
	Format       string             `yaml:"format" json:"format"`
	Variables    []TemplateVariable `yaml:"variables,omitempty" json:"variables,omitempty"`
}

// CLIDependency represents an external CLI tool dependency.
type CLIDependency struct {
	Name       string `yaml:"name" json:"name"`
	MinVersion string `yaml:"min_version,omitempty" json:"min_version,omitempty"`
}

// InputField represents an input parameter for a skill or workflow.
type InputField struct {
	Name        string      `yaml:"name" json:"name"`
	Type        string      `yaml:"type" json:"type"`
	Required    bool        `yaml:"required,omitempty" json:"required,omitempty"`
	Default     interface{} `yaml:"default,omitempty" json:"default,omitempty"`
	Description string      `yaml:"description,omitempty" json:"description,omitempty"`
}

// OutputDeclaration describes the output format of a skill or workflow.
type OutputDeclaration struct {
	Format string `yaml:"format" json:"format"`
	Schema string `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// RegistryBlock declares userdata folder structure for a skill.
type RegistryBlock struct {
	Tokens    []RegistryToken        `yaml:"tokens,omitempty" json:"tokens,omitempty"`
	Config    map[string]interface{} `yaml:"config,omitempty" json:"config,omitempty"`
	State     []string               `yaml:"state,omitempty" json:"state,omitempty"`
	Output    *RegistryOutput        `yaml:"output,omitempty" json:"output,omitempty"`
	Templates *RegistryTemplates     `yaml:"templates,omitempty" json:"templates,omitempty"`
}

// RegistryToken represents an environment variable or secret required by a skill.
type RegistryToken struct {
	Name        string `yaml:"name" json:"name"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// RegistryOutput describes the output schema for a skill's output/latest.json.
type RegistryOutput struct {
	Schema string `yaml:"schema,omitempty" json:"schema,omitempty"`
}

// RegistryTemplates describes template graduation configuration for a skill.
type RegistryTemplates struct {
	Format      string `yaml:"format" json:"format"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

// WorkflowStep represents a single step in a workflow.
type WorkflowStep struct {
	ID     string                 `yaml:"id" json:"id"`
	Skill  string                 `yaml:"skill" json:"skill"`
	Inputs map[string]interface{} `yaml:"inputs,omitempty" json:"inputs,omitempty"`
}

// TemplateVariable represents a variable available for substitution in a template.
type TemplateVariable struct {
	Name        string `yaml:"name" json:"name"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
	Default     string `yaml:"default,omitempty" json:"default,omitempty"`
	Required    bool   `yaml:"required,omitempty" json:"required,omitempty"`
}

// ManifestType constants for the type discriminator field.
const (
	TypeContext  = "context"
	TypePersona  = "persona"
	TypeSkill    = "skill"
	TypeWorkflow = "workflow"
	TypePrompt   = "prompt"
	TypeTemplate = "template"
)

// ValidTypes contains all valid manifest type values.
var ValidTypes = []string{
	TypeContext,
	TypePersona,
	TypeSkill,
	TypeWorkflow,
	TypePrompt,
	TypeTemplate,
}
