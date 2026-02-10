import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, readFileSync, existsSync, readlinkSync, rmSync, mkdtempSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { generate } from '../src/index.mjs';

let tempDir;
let installedPath;
let projectPath;

function setupMockInstalled() {
  installedPath = join(tempDir, 'installed');

  // Persona
  const personaDir = join(installedPath, 'personas', 'senior-java-dev');
  mkdirSync(personaDir, { recursive: true });
  writeFileSync(join(personaDir, 'manifest.yaml'), [
    'name: senior-java-dev',
    'type: persona',
    'version: "1.0.0"',
    'description: You are a senior Java developer with Spring Boot expertise.',
    'tone: direct, pragmatic, opinionated',
    'conventions:',
    '  - Prefers constructor injection over field injection',
    '  - Enforces test coverage for public methods',
  ].join('\n'));

  // Context
  const ctx1 = join(installedPath, 'context', 'spring-boot', 'error-handling');
  mkdirSync(ctx1, { recursive: true });
  writeFileSync(join(ctx1, 'patterns.md'), '# Error handling patterns\n');

  const ctx2 = join(installedPath, 'context', 'spring-boot', 'security');
  mkdirSync(ctx2, { recursive: true });
  writeFileSync(join(ctx2, 'patterns.md'), '# Security patterns\n');
}

function createProjectConfig() {
  return {
    tools: ['augment'],
    active: {
      personas: ['personas/senior-java-dev'],
      context: [
        'context/spring-boot/error-handling',
        'context/spring-boot/security',
      ],
      skills: [],
      workflows: [],
    },
  };
}

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-augment-'));
  setupMockInstalled();
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('generate', () => {
  it('creates augment-guidelines.md with persona description', async () => {
    const config = createProjectConfig();
    const result = await generate(config, installedPath, projectPath);

    const guidelinesPath = join(projectPath, '.augment', 'augment-guidelines.md');
    assert.ok(existsSync(guidelinesPath));
    assert.ok(result.created.includes(guidelinesPath));

    const content = readFileSync(guidelinesPath, 'utf8');
    assert.ok(content.includes('senior Java developer with Spring Boot expertise'));
    assert.ok(content.includes('direct, pragmatic, opinionated'));
    assert.ok(content.includes('Prefers constructor injection'));
    assert.ok(content.includes('See .augment/context/'));
  });

  it('creates context symlinks with flattened names', async () => {
    const config = createProjectConfig();
    const result = await generate(config, installedPath, projectPath);

    const link1 = join(projectPath, '.augment', 'context', 'spring-boot-error-handling');
    const link2 = join(projectPath, '.augment', 'context', 'spring-boot-security');

    assert.ok(existsSync(link1));
    assert.ok(existsSync(link2));
    assert.equal(
      readlinkSync(link1),
      join(installedPath, 'context', 'spring-boot', 'error-handling')
    );
    assert.equal(result.symlinked.length, 2);
  });

  it('handles missing persona with a warning', async () => {
    const config = {
      active: {
        personas: ['personas/nonexistent'],
        context: [],
      },
    };
    const result = await generate(config, installedPath, projectPath);
    assert.ok(result.warnings.some(w => w.includes('Persona not found')));
  });

  it('handles missing context with a warning', async () => {
    const config = {
      active: {
        personas: [],
        context: ['context/nonexistent'],
      },
    };
    const result = await generate(config, installedPath, projectPath);
    assert.ok(result.warnings.some(w => w.includes('Context not found')));
    assert.equal(result.symlinked.length, 0);
  });

  it('is idempotent', async () => {
    const config = createProjectConfig();
    const result1 = await generate(config, installedPath, projectPath);
    assert.ok(result1.created.length > 0);

    const result2 = await generate(config, installedPath, projectPath);
    const guidelinesPath = join(projectPath, '.augment', 'augment-guidelines.md');
    assert.ok(result2.updated.includes(guidelinesPath));
  });

  it('generates with empty config', async () => {
    const config = {};
    const result = await generate(config, installedPath, projectPath);
    assert.equal(result.warnings.length, 0);
    assert.ok(existsSync(join(projectPath, '.augment', 'augment-guidelines.md')));
  });
});
