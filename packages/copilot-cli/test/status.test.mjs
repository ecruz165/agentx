import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, symlinkSync, rmSync, mkdtempSync, utimesSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { status } from '../src/index.mjs';

let tempDir;
let projectPath;

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-copilot-status-'));
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('status', () => {
  it('returns not-generated when copilot-instructions.md does not exist', async () => {
    const result = await status(projectPath);
    assert.equal(result.tool, 'copilot');
    assert.equal(result.status, 'not-generated');
  });

  it('returns up-to-date when instructions are newer than project.yaml', async () => {
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    const projectYaml = join(agentxDir, 'project.yaml');
    writeFileSync(projectYaml, 'tools:\n  - copilot\n');
    const past = new Date(Date.now() - 10000);
    utimesSync(projectYaml, past, past);

    const githubDir = join(projectPath, '.github');
    mkdirSync(githubDir, { recursive: true });
    writeFileSync(join(githubDir, 'copilot-instructions.md'), '# Generated');

    const result = await status(projectPath);
    assert.equal(result.status, 'up-to-date');
  });

  it('returns stale when project.yaml is newer', async () => {
    const githubDir = join(projectPath, '.github');
    mkdirSync(githubDir, { recursive: true });
    const instrPath = join(githubDir, 'copilot-instructions.md');
    writeFileSync(instrPath, '# Old');
    const past = new Date(Date.now() - 10000);
    utimesSync(instrPath, past, past);

    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    writeFileSync(join(agentxDir, 'project.yaml'), 'tools:\n  - copilot\n');

    const result = await status(projectPath);
    assert.equal(result.status, 'stale');
  });

  it('counts symlinks correctly', async () => {
    const githubDir = join(projectPath, '.github');
    mkdirSync(githubDir, { recursive: true });
    writeFileSync(join(githubDir, 'copilot-instructions.md'), '# Generated');

    const contextDir = join(githubDir, 'copilot-context');
    mkdirSync(contextDir);

    const validTarget = join(tempDir, 'target');
    mkdirSync(validTarget);
    symlinkSync(validTarget, join(contextDir, 'valid'));
    symlinkSync(join(tempDir, 'nope'), join(contextDir, 'broken'));

    const result = await status(projectPath);
    assert.equal(result.symlinks.total, 2);
    assert.equal(result.symlinks.valid, 1);
  });
});
