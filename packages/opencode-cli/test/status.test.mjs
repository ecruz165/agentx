import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, existsSync, symlinkSync, rmSync, mkdtempSync, utimesSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { status } from '../src/index.mjs';

let tempDir;
let projectPath;

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-opencode-status-'));
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('status', () => {
  it('returns not-generated when AGENTS.md does not exist', async () => {
    const result = await status(projectPath);
    assert.equal(result.tool, 'opencode');
    assert.equal(result.status, 'not-generated');
    assert.equal(result.files.length, 0);
  });

  it('returns up-to-date when AGENTS.md is newer than project.yaml', async () => {
    // Create project.yaml in the past
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    const projectYaml = join(agentxDir, 'project.yaml');
    writeFileSync(projectYaml, 'tools:\n  - opencode\n');
    const past = new Date(Date.now() - 10000);
    utimesSync(projectYaml, past, past);

    // Create AGENTS.md now (newer) in project root
    writeFileSync(join(projectPath, 'AGENTS.md'), '# Generated');

    const result = await status(projectPath);
    assert.equal(result.status, 'up-to-date');
    assert.equal(result.files.length, 1);
  });

  it('returns stale when project.yaml is newer than AGENTS.md', async () => {
    // Create AGENTS.md in the past (project root)
    const agentsMd = join(projectPath, 'AGENTS.md');
    writeFileSync(agentsMd, '# Old');
    const past = new Date(Date.now() - 10000);
    utimesSync(agentsMd, past, past);

    // Create project.yaml now (newer)
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    writeFileSync(join(agentxDir, 'project.yaml'), 'tools:\n  - opencode\n');

    const result = await status(projectPath);
    assert.equal(result.status, 'stale');
  });

  it('counts valid and broken symlinks in .opencode/context/', async () => {
    // Create AGENTS.md in project root
    writeFileSync(join(projectPath, 'AGENTS.md'), '# Generated');

    const contextDir = join(projectPath, '.opencode', 'context');
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

  it('returns zero symlinks when .opencode/context/ does not exist', async () => {
    writeFileSync(join(projectPath, 'AGENTS.md'), '# Generated');

    const result = await status(projectPath);
    assert.equal(result.symlinks.total, 0);
    assert.equal(result.symlinks.valid, 0);
  });
});
