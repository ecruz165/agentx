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
    '  - Follows 12-factor app principles',
    'context:',
    '  - context/spring-boot/error-handling',
  ].join('\n'));

  // Context
  const ctx1 = join(installedPath, 'context', 'spring-boot', 'error-handling');
  mkdirSync(ctx1, { recursive: true });
  writeFileSync(join(ctx1, 'manifest.yaml'), 'name: spring-boot-error-handling\ntype: context\nversion: "1.0.0"\ndescription: Error handling patterns\nformat: markdown\nsources:\n  - patterns.md\n');
  writeFileSync(join(ctx1, 'patterns.md'), '# Error handling patterns\n');

  const ctx2 = join(installedPath, 'context', 'spring-boot', 'security');
  mkdirSync(ctx2, { recursive: true });
  writeFileSync(join(ctx2, 'manifest.yaml'), 'name: spring-boot-security\ntype: context\nversion: "1.0.0"\ndescription: Security patterns\nformat: markdown\nsources:\n  - patterns.md\n');
  writeFileSync(join(ctx2, 'patterns.md'), '# Security patterns\n');

  // Skill
  const skillDir = join(installedPath, 'skills', 'scm', 'git', 'commit-analyzer');
  mkdirSync(skillDir, { recursive: true });
  writeFileSync(join(skillDir, 'manifest.yaml'), [
    'name: commit-analyzer',
    'type: skill',
    'version: "2.1.0"',
    'description: Analyzes git commit history for team metrics',
    'runtime: node',
    'topic: scm',
    'inputs:',
    '  - name: repoPath',
    '    type: string',
    '    required: true',
    '    description: Path to the git repository',
    '  - name: days',
    '    type: number',
    '    default: 30',
    '    description: Number of days to analyze',
  ].join('\n'));

  // Workflow
  const wfDir = join(installedPath, 'workflows', 'deploy-verify');
  mkdirSync(wfDir, { recursive: true });
  writeFileSync(join(wfDir, 'manifest.yaml'), [
    'name: deploy-verify',
    'type: workflow',
    'version: "1.0.0"',
    'description: Verifies deployment by checking commits, pipeline status, and config',
    'runtime: node',
    'inputs:',
    '  - name: repoPath',
    '    type: string',
    '    required: true',
    '  - name: pipelineId',
    '    type: string',
    '    required: true',
  ].join('\n'));
}

function createProjectConfig() {
  return {
    tools: ['claude-code', 'copilot', 'augment'],
    active: {
      personas: ['personas/senior-java-dev'],
      context: [
        'context/spring-boot/error-handling',
        'context/spring-boot/security',
      ],
      skills: ['skills/scm/git/commit-analyzer'],
      workflows: ['workflows/deploy-verify'],
    },
  };
}

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-claudecode-'));
  setupMockInstalled();
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('generate', () => {
  it('creates CLAUDE.md with persona, skills, workflows, and context reference', async () => {
    const config = createProjectConfig();
    const result = await generate(config, installedPath, projectPath);

    const claudeMd = join(projectPath, '.claude', 'CLAUDE.md');
    assert.ok(existsSync(claudeMd));
    assert.ok(result.created.includes(claudeMd));

    const content = readFileSync(claudeMd, 'utf8');
    assert.ok(content.includes('# Project Assistant Configuration'));
    assert.ok(content.includes('senior Java developer with Spring Boot expertise'));
    assert.ok(content.includes('direct, pragmatic, opinionated'));
    assert.ok(content.includes('Prefers constructor injection'));
    assert.ok(content.includes('commit-analyzer'));
    assert.ok(content.includes('deploy-verify'));
    assert.ok(content.includes('Refer to .claude/context/'));
  });

  it('creates command files for skills and workflows', async () => {
    const config = createProjectConfig();
    const result = await generate(config, installedPath, projectPath);

    const skillCmd = join(projectPath, '.claude', 'commands', 'commit-analyzer.md');
    assert.ok(existsSync(skillCmd));
    const skillContent = readFileSync(skillCmd, 'utf8');
    assert.ok(skillContent.includes('Analyzes git commit history'));
    assert.ok(skillContent.includes('agentx run skills/scm/git/commit-analyzer'));

    const wfCmd = join(projectPath, '.claude', 'commands', 'deploy-verify.md');
    assert.ok(existsSync(wfCmd));
    const wfContent = readFileSync(wfCmd, 'utf8');
    assert.ok(wfContent.includes('Verifies deployment'));
    assert.ok(wfContent.includes('agentx run workflows/deploy-verify'));
  });

  it('creates context symlinks with flattened names', async () => {
    const config = createProjectConfig();
    const result = await generate(config, installedPath, projectPath);

    const link1 = join(projectPath, '.claude', 'context', 'spring-boot-error-handling');
    const link2 = join(projectPath, '.claude', 'context', 'spring-boot-security');

    assert.ok(existsSync(link1));
    assert.ok(existsSync(link2));
    assert.equal(
      readlinkSync(link1),
      join(installedPath, 'context', 'spring-boot', 'error-handling')
    );
    assert.equal(
      readlinkSync(link2),
      join(installedPath, 'context', 'spring-boot', 'security')
    );
    assert.equal(result.symlinked.length, 2);
  });

  it('handles missing persona gracefully with a warning', async () => {
    const config = {
      active: {
        personas: ['personas/nonexistent'],
        context: [],
        skills: [],
        workflows: [],
      },
    };
    const result = await generate(config, installedPath, projectPath);

    assert.ok(result.warnings.some(w => w.includes('Persona not found: personas/nonexistent')));
    const claudeMd = join(projectPath, '.claude', 'CLAUDE.md');
    assert.ok(existsSync(claudeMd));
  });

  it('handles missing skill gracefully with a warning', async () => {
    const config = {
      active: {
        personas: [],
        context: [],
        skills: ['skills/nonexistent/tool'],
        workflows: [],
      },
    };
    const result = await generate(config, installedPath, projectPath);
    assert.ok(result.warnings.some(w => w.includes('Skill not found')));
  });

  it('handles missing context gracefully with a warning', async () => {
    const config = {
      active: {
        personas: [],
        context: ['context/nonexistent'],
        skills: [],
        workflows: [],
      },
    };
    const result = await generate(config, installedPath, projectPath);
    assert.ok(result.warnings.some(w => w.includes('Context not found')));
    assert.equal(result.symlinked.length, 0);
  });

  it('is idempotent — second run reports updated not created', async () => {
    const config = createProjectConfig();
    const result1 = await generate(config, installedPath, projectPath);
    assert.ok(result1.created.length > 0);

    const result2 = await generate(config, installedPath, projectPath);
    const claudeMd = join(projectPath, '.claude', 'CLAUDE.md');
    assert.ok(result2.updated.includes(claudeMd));
  });

  it('generates with empty active section', async () => {
    const config = { active: {} };
    const result = await generate(config, installedPath, projectPath);

    assert.equal(result.warnings.length, 0);
    const claudeMd = join(projectPath, '.claude', 'CLAUDE.md');
    assert.ok(existsSync(claudeMd));
  });

  it('generates with no active section at all', async () => {
    const config = {};
    const result = await generate(config, installedPath, projectPath);
    assert.equal(result.warnings.length, 0);
  });
});
