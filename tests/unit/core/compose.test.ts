import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { compose, render } from '../../../src/core/compose.js';

describe('compose', () => {
  let installedDir: string;

  beforeEach(() => {
    installedDir = join(tmpdir(), `agentx-compose-test-${Date.now()}`);
    mkdirSync(installedDir, { recursive: true });
  });

  afterEach(() => {
    rmSync(installedDir, { recursive: true, force: true });
  });

  it('composes a prompt with persona and context', () => {
    // Create persona
    const personaDir = join(installedDir, 'personas/java-dev');
    mkdirSync(personaDir, { recursive: true });
    writeFileSync(
      join(personaDir, 'manifest.yaml'),
      `name: java-dev
type: persona
version: "1.0.0"
description: Java developer
expertise:
  - Spring Boot
  - AWS
tone: pragmatic
conventions:
  - Follow SOLID principles`,
    );

    // Create context
    const ctxDir = join(installedDir, 'context/spring-boot');
    mkdirSync(ctxDir, { recursive: true });
    writeFileSync(
      join(ctxDir, 'manifest.yaml'),
      `name: spring-boot
type: context
version: "1.0.0"
description: Spring Boot context
format: markdown
sources:
  - content.md`,
    );
    writeFileSync(join(ctxDir, 'content.md'), '# Spring Boot\nBest practices for Spring Boot.');

    // Create prompt
    const promptDir = join(installedDir, 'prompts/java-review');
    mkdirSync(promptDir, { recursive: true });
    writeFileSync(
      join(promptDir, 'manifest.yaml'),
      `name: java-review
type: prompt
version: "1.0.0"
description: Java code review prompt
persona: personas/java-dev
context:
  - context/spring-boot`,
    );

    const result = compose('prompts/java-review', installedDir);
    expect(result.promptName).toBe('java-review');
    expect(result.persona?.name).toBe('java-dev');
    expect(result.persona?.expertise).toContain('Spring Boot');
    expect(result.context.length).toBe(1);
    expect(result.context[0].content).toContain('Spring Boot');
    expect(result.warnings.length).toBe(0);
  });

  it('produces warnings for missing references', () => {
    const promptDir = join(installedDir, 'prompts/broken');
    mkdirSync(promptDir, { recursive: true });
    writeFileSync(
      join(promptDir, 'manifest.yaml'),
      `name: broken
type: prompt
version: "1.0.0"
description: Broken prompt
persona: personas/nonexistent
context:
  - context/nonexistent`,
    );

    const result = compose('prompts/broken', installedDir);
    expect(result.persona).toBeNull();
    expect(result.context.length).toBe(0);
    expect(result.warnings.length).toBe(2);
  });

  it('renders markdown output', () => {
    const personaDir = join(installedDir, 'personas/test');
    mkdirSync(personaDir, { recursive: true });
    writeFileSync(
      join(personaDir, 'manifest.yaml'),
      `name: test
type: persona
version: "1.0.0"
description: Test
expertise:
  - Testing
tone: direct`,
    );

    const promptDir = join(installedDir, 'prompts/test');
    mkdirSync(promptDir, { recursive: true });
    writeFileSync(
      join(promptDir, 'manifest.yaml'),
      `name: test
type: prompt
version: "1.0.0"
description: Test prompt
persona: personas/test`,
    );

    const composed = compose('prompts/test', installedDir);
    const md = render(composed);
    expect(md).toContain('# Persona: test');
    expect(md).toContain('**Expertise:** Testing');
    expect(md).toContain('**Tone:** direct');
  });
});
