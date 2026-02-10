import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, symlinkSync, rmSync, mkdtempSync, utimesSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import { status } from '../src/index.mjs';

let tempDir;
let projectPath;

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-augment-status-'));
  projectPath = join(tempDir, 'my-project');
  mkdirSync(projectPath, { recursive: true });
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('status', () => {
  it('returns not-generated when augment-guidelines.md does not exist', async () => {
    const result = await status(projectPath);
    assert.equal(result.tool, 'augment');
    assert.equal(result.status, 'not-generated');
  });

  it('returns up-to-date when guidelines are newer than project.yaml', async () => {
    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    const projectYaml = join(agentxDir, 'project.yaml');
    writeFileSync(projectYaml, 'tools:\n  - augment\n');
    const past = new Date(Date.now() - 10000);
    utimesSync(projectYaml, past, past);

    const augmentDir = join(projectPath, '.augment');
    mkdirSync(augmentDir, { recursive: true });
    writeFileSync(join(augmentDir, 'augment-guidelines.md'), '# Generated');

    const result = await status(projectPath);
    assert.equal(result.status, 'up-to-date');
  });

  it('returns stale when project.yaml is newer', async () => {
    const augmentDir = join(projectPath, '.augment');
    mkdirSync(augmentDir, { recursive: true });
    const guidelinesPath = join(augmentDir, 'augment-guidelines.md');
    writeFileSync(guidelinesPath, '# Old');
    const past = new Date(Date.now() - 10000);
    utimesSync(guidelinesPath, past, past);

    const agentxDir = join(projectPath, '.agentx');
    mkdirSync(agentxDir, { recursive: true });
    writeFileSync(join(agentxDir, 'project.yaml'), 'tools:\n  - augment\n');

    const result = await status(projectPath);
    assert.equal(result.status, 'stale');
  });

  it('counts symlinks correctly', async () => {
    const augmentDir = join(projectPath, '.augment');
    mkdirSync(augmentDir, { recursive: true });
    writeFileSync(join(augmentDir, 'augment-guidelines.md'), '# Generated');

    const contextDir = join(augmentDir, 'context');
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
