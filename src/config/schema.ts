import { z } from 'zod';

// ── Shared sub-schemas ──────────────────────────────────────────────

export const CLIDependencySchema = z.object({
  name: z.string(),
  min_version: z.string().optional(),
});

export const InputFieldSchema = z.object({
  name: z.string(),
  type: z.enum(['string', 'number', 'boolean', 'array', 'object']),
  required: z.boolean().optional(),
  default: z.unknown().optional(),
  description: z.string().optional(),
});

export const OutputDeclarationSchema = z.object({
  format: z.string(),
  schema: z.string().optional(),
});

export const RegistryTokenSchema = z.object({
  name: z.string(),
  required: z.boolean().optional(),
  default: z.string().optional(),
  description: z.string().optional(),
});

export const RegistryTemplatesSchema = z.object({
  format: z.string(),
  description: z.string().optional(),
});

export const RegistryBlockSchema = z.object({
  tokens: z.array(RegistryTokenSchema).optional(),
  config: z.record(z.string(), z.unknown()).optional(),
  state: z.array(z.string()).optional(),
  output: z.object({ schema: z.string().optional() }).optional(),
  templates: RegistryTemplatesSchema.nullable().optional(),
});

export const WorkflowStepSchema = z.object({
  id: z.string(),
  skill: z.string().regex(/^skills\/[a-z0-9-]+(\/[a-z0-9-]+)*$/),
  inputs: z.record(z.string(), z.unknown()).optional(),
});

export const TemplateVariableSchema = z.object({
  name: z.string(),
  description: z.string().optional(),
  default: z.string().optional(),
  required: z.boolean().optional(),
});

// ── Base fields (shared by all manifest types) ──────────────────────

const namePattern = /^[a-z0-9][a-z0-9-]*$/;
const versionPattern = /^v?[0-9]+(\.[0-9]+)*(-[a-zA-Z0-9.-]+)?$/;

const BaseFields = {
  name: z.string().regex(namePattern, 'Lowercase alphanumeric with hyphens'),
  version: z.string().regex(versionPattern, 'Relaxed semver: 1.0, 1.2.3, v1.2.3'),
  description: z.string().min(1),
  tags: z.array(z.string()).optional(),
  author: z.string().optional(),
  vendor: z.string().nullable().optional(),
};

// ── Manifest type schemas ───────────────────────────────────────────

export const MANIFEST_TYPES = [
  'context',
  'persona',
  'skill',
  'workflow',
  'prompt',
  'template',
] as const;

export type ManifestType = (typeof MANIFEST_TYPES)[number];

export const ContextManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('context'),
  format: z.string(),
  tokens: z.number().int().nonnegative().optional(),
  sources: z.array(z.string()).min(1),
});

export const PersonaManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('persona'),
  expertise: z.array(z.string()).optional(),
  tone: z.string().optional(),
  conventions: z.array(z.string()).optional(),
  context: z
    .array(z.string().regex(/^context\/[a-z0-9-]+(\/[a-z0-9-]+)*$/))
    .optional(),
  template: z.string().optional(),
});

export const SkillManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('skill'),
  runtime: z.enum(['node', 'go']),
  topic: z.string(),
  cli_dependencies: z.array(CLIDependencySchema).optional(),
  inputs: z.array(InputFieldSchema).optional(),
  outputs: OutputDeclarationSchema.optional(),
  registry: RegistryBlockSchema.optional(),
});

export const WorkflowManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('workflow'),
  runtime: z.enum(['node', 'go']),
  steps: z.array(WorkflowStepSchema).min(1),
  inputs: z.array(InputFieldSchema).optional(),
  outputs: OutputDeclarationSchema.optional(),
});

export const PromptManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('prompt'),
  persona: z.string().regex(/^personas\/[a-z0-9-]+$/).optional(),
  context: z
    .array(z.string().regex(/^context\/[a-z0-9-]+(\/[a-z0-9-]+)*$/))
    .optional(),
  skills: z
    .array(z.string().regex(/^skills\/[a-z0-9-]+(\/[a-z0-9-]+)*$/))
    .optional(),
  workflows: z
    .array(z.string().regex(/^workflows\/[a-z0-9-]+(\/[a-z0-9-]+)*$/))
    .optional(),
  template: z.string().optional(),
});

export const TemplateManifestSchema = z.object({
  ...BaseFields,
  type: z.literal('template'),
  format: z.string(),
  variables: z.array(TemplateVariableSchema).optional(),
});

// ── Discriminated union ─────────────────────────────────────────────

export const ManifestSchema = z.discriminatedUnion('type', [
  ContextManifestSchema,
  PersonaManifestSchema,
  SkillManifestSchema,
  WorkflowManifestSchema,
  PromptManifestSchema,
  TemplateManifestSchema,
]);
