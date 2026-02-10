import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, existsSync, symlinkSync, rmSync, mkdtempSync, utimesSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { status } from '../src/index.mjs';

let tempDir;
let projectPath;

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-claudecode-status-'));
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('status', () => {
  it('returns not-generated when CLAUDE.md does not exist', async () => {
    const result = await status(projectPath);
    assert.equal(result.tool, 'claude-code');
    assert.equal(result.status, 'not-generated');
    assert.equal(result.files.length, 0);
  });

  it('returns up-to-date when CLAUDE.md is newer than project.yaml', async () => {
    // Create project.yaml in the past
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    const projectYaml = join(agentxDir, 'project.yaml');
    writeFileSync(projectYaml, 'tools:\n  - claude-code\n');
    const past = new Date(Date.now() - 10000);
    utimesSync(projectYaml, past, past);

    // Create CLAUDE.md now (newer)
    const claudeDir = join(projectPath, '.claude');
    mkdirSync(claudeDir, { recursive: true });
    writeFileSync(join(claudeDir, 'CLAUDE.md'), '# Generated');

    const result = await status(projectPath);
    assert.equal(result.status, 'up-to-date');
    assert.equal(result.files.length, 1);
  });

  it('returns stale when project.yaml is newer than CLAUDE.md', async () => {
    // Create CLAUDE.md in the past
    const claudeDir = join(projectPath, '.claude');
    mkdirSync(claudeDir, { recursive: true });
    const claudeMd = join(claudeDir, 'CLAUDE.md');
    writeFileSync(claudeMd, '# Old');
    const past = new Date(Date.now() - 10000);
    utimesSync(claudeMd, past, past);

    // Create project.yaml now (newer)
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    writeFileSync(join(agentxDir, 'project.yaml'), 'tools:\n  - claude-code\n');

    const result = await status(projectPath);
    assert.equal(result.status, 'stale');
  });

  it('counts valid and broken symlinks', async () => {
    const claudeDir = join(projectPath, '.claude');
    mkdirSync(claudeDir, { recursive: true });
    writeFileSync(join(claudeDir, 'CLAUDE.md'), '# Generated');

    const contextDir = join(claudeDir, 'context');
    mkdirSync(contextDir, { recursive: true });

    // Valid symlink
    const validTarget = join(tempDir, 'installed', 'context', 'valid');
    mkdirSync(validTarget, { recursive: true });
    symlinkSync(validTarget, join(contextDir, 'valid-context'));

    // Broken symlink
    symlinkSync(join(tempDir, 'nonexistent'), join(contextDir, 'broken-context'));

    const result = await status(projectPath);
    assert.equal(result.symlinks.total, 2);
    assert.equal(result.symlinks.valid, 1);
  });
});
