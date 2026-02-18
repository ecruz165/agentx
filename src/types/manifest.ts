import type { z } from 'zod';
import type {
  ContextManifestSchema,
  PersonaManifestSchema,
  SkillManifestSchema,
  WorkflowManifestSchema,
  PromptManifestSchema,
  TemplateManifestSchema,
  ManifestSchema,
  CLIDependencySchema,
  InputFieldSchema,
  OutputDeclarationSchema,
  RegistryBlockSchema,
  WorkflowStepSchema,
  TemplateVariableSchema,
} from '../config/schema.js';

export type ContextManifest = z.infer<typeof ContextManifestSchema>;
export type PersonaManifest = z.infer<typeof PersonaManifestSchema>;
export type SkillManifest = z.infer<typeof SkillManifestSchema>;
export type WorkflowManifest = z.infer<typeof WorkflowManifestSchema>;
export type PromptManifest = z.infer<typeof PromptManifestSchema>;
export type TemplateManifest = z.infer<typeof TemplateManifestSchema>;
export type Manifest = z.infer<typeof ManifestSchema>;

export type CLIDependency = z.infer<typeof CLIDependencySchema>;
export type InputField = z.infer<typeof InputFieldSchema>;
export type OutputDeclaration = z.infer<typeof OutputDeclarationSchema>;
export type RegistryBlock = z.infer<typeof RegistryBlockSchema>;
export type WorkflowStep = z.infer<typeof WorkflowStepSchema>;
export type TemplateVariable = z.infer<typeof TemplateVariableSchema>;

export type BaseManifest = {
  name: string;
  type: string;
  version: string;
  description: string;
  tags?: string[];
  author?: string;
  vendor?: string | null;
};
