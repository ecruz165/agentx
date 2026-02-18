import { describe, it, expect } from 'vitest';
import {
  ManifestSchema,
  ContextManifestSchema,
  PersonaManifestSchema,
  SkillManifestSchema,
  WorkflowManifestSchema,
  PromptManifestSchema,
  TemplateManifestSchema,
} from '../../../src/config/schema.js';

describe('ManifestSchema', () => {
  describe('BaseManifest validation', () => {
    it('rejects missing name', () => {
      const result = ContextManifestSchema.safeParse({
        type: 'context',
        version: '1.0',
        description: 'test',
        format: 'markdown',
        sources: ['file.md'],
      });
      expect(result.success).toBe(false);
    });

    it('rejects invalid name pattern', () => {
      const result = ContextManifestSchema.safeParse({
        name: 'Invalid_Name',
        type: 'context',
        version: '1.0',
        description: 'test',
        format: 'markdown',
        sources: ['file.md'],
      });
      expect(result.success).toBe(false);
    });

    it('rejects invalid version pattern', () => {
      const result = ContextManifestSchema.safeParse({
        name: 'test',
        type: 'context',
        version: 'abc',
        description: 'test',
        format: 'markdown',
        sources: ['file.md'],
      });
      expect(result.success).toBe(false);
    });

    it('accepts valid version formats', () => {
      for (const version of ['1.0', '1.2.3', 'v1.2.3', '1.2.3-beta.1']) {
        const result = ContextManifestSchema.safeParse({
          name: 'test',
          type: 'context',
          version,
          description: 'test',
          format: 'markdown',
          sources: ['file.md'],
        });
        expect(result.success).toBe(true);
      }
    });
  });

  describe('ContextManifest', () => {
    it('validates a valid context manifest', () => {
      const result = ContextManifestSchema.safeParse({
        name: 'spring-boot',
        type: 'context',
        version: '1.0.0',
        description: 'Spring Boot context',
        format: 'markdown',
        sources: ['content.md'],
        tokens: 1500,
      });
      expect(result.success).toBe(true);
    });

    it('requires sources with at least one entry', () => {
      const result = ContextManifestSchema.safeParse({
        name: 'test',
        type: 'context',
        version: '1.0',
        description: 'test',
        format: 'markdown',
        sources: [],
      });
      expect(result.success).toBe(false);
    });
  });

  describe('PersonaManifest', () => {
    it('validates a valid persona manifest', () => {
      const result = PersonaManifestSchema.safeParse({
        name: 'senior-java-dev',
        type: 'persona',
        version: '1.0.0',
        description: 'Senior Java developer',
        expertise: ['Spring Boot', 'AWS'],
        tone: 'pragmatic',
        conventions: ['Follow SOLID principles'],
        context: ['context/spring-boot'],
      });
      expect(result.success).toBe(true);
    });

    it('validates context references pattern', () => {
      const result = PersonaManifestSchema.safeParse({
        name: 'test',
        type: 'persona',
        version: '1.0',
        description: 'test',
        context: ['invalid-path'],
      });
      expect(result.success).toBe(false);
    });
  });

  describe('SkillManifest', () => {
    it('validates a valid skill manifest', () => {
      const result = SkillManifestSchema.safeParse({
        name: 'commit-analyzer',
        type: 'skill',
        version: '1.0.0',
        description: 'Analyzes git commits',
        runtime: 'node',
        topic: 'scm',
        cli_dependencies: [{ name: 'git', min_version: '2.0.0' }],
        inputs: [{ name: 'repo', type: 'string', required: true }],
      });
      expect(result.success).toBe(true);
    });

    it('requires runtime and topic', () => {
      const result = SkillManifestSchema.safeParse({
        name: 'test',
        type: 'skill',
        version: '1.0',
        description: 'test',
      });
      expect(result.success).toBe(false);
    });
  });

  describe('WorkflowManifest', () => {
    it('validates a valid workflow manifest', () => {
      const result = WorkflowManifestSchema.safeParse({
        name: 'code-review',
        type: 'workflow',
        version: '1.0.0',
        description: 'Code review workflow',
        runtime: 'node',
        steps: [
          { id: 'analyze', skill: 'skills/scm/git/commit-analyzer' },
          { id: 'report', skill: 'skills/ai/openai/summarizer' },
        ],
      });
      expect(result.success).toBe(true);
    });

    it('requires at least one step', () => {
      const result = WorkflowManifestSchema.safeParse({
        name: 'test',
        type: 'workflow',
        version: '1.0',
        description: 'test',
        runtime: 'node',
        steps: [],
      });
      expect(result.success).toBe(false);
    });
  });

  describe('PromptManifest', () => {
    it('validates a valid prompt manifest', () => {
      const result = PromptManifestSchema.safeParse({
        name: 'java-pr-review',
        type: 'prompt',
        version: '1.0.0',
        description: 'Java PR review prompt',
        persona: 'personas/senior-java-dev',
        context: ['context/spring-boot'],
        skills: ['skills/scm/git/commit-analyzer'],
        workflows: ['workflows/code-review'],
      });
      expect(result.success).toBe(true);
    });
  });

  describe('TemplateManifest', () => {
    it('validates a valid template manifest', () => {
      const result = TemplateManifestSchema.safeParse({
        name: 'skill-readme',
        type: 'template',
        version: '1.0.0',
        description: 'README template for skills',
        format: 'hbs',
        variables: [
          { name: 'skillName', description: 'Name of the skill', required: true },
        ],
      });
      expect(result.success).toBe(true);
    });
  });

  describe('Discriminated union', () => {
    it('parses context type correctly', () => {
      const data = {
        name: 'test',
        type: 'context',
        version: '1.0',
        description: 'test',
        format: 'markdown',
        sources: ['file.md'],
      };
      const result = ManifestSchema.parse(data);
      expect(result.type).toBe('context');
    });

    it('rejects unknown type', () => {
      const result = ManifestSchema.safeParse({
        name: 'test',
        type: 'unknown',
        version: '1.0',
        description: 'test',
      });
      expect(result.success).toBe(false);
    });
  });
});
