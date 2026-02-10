import { describe, it, beforeEach, afterEach } from 'node:test';
import assert from 'node:assert/strict';
import { mkdirSync, writeFileSync, readlinkSync, existsSync, symlinkSync, utimesSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { mkdtempSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { loadManifest, createSymlink, flattenRef, isStale, ensureDir, validateSymlinks } from '../src/link-helpers.mjs';

let tempDir;

beforeEach(() => {
  tempDir = mkdtempSync(join(tmpdir(), 'agentx-link-helpers-'));
});

afterEach(() => {
  rmSync(tempDir, { recursive: true, force: true });
});

describe('loadManifest', () => {
  it('loads manifest.yaml when present', () => {
    const typeDir = join(tempDir, 'personas', 'senior-java-dev');
    mkdirSync(typeDir, { recursive: true });
    writeFileSync(join(typeDir, 'manifest.yaml'), 'name: senior-java-dev\ntype: persona\nversion: "1.0.0"\ndescription: A senior Java developer\ntone: direct\nconventions:\n  - use constructor injection\n');

    const result = loadManifest(tempDir, 'personas/senior-java-dev');
    assert.ok(result);
    assert.equal(result.manifest.name, 'senior-java-dev');
    assert.equal(result.manifest.type, 'persona');
    assert.equal(result.manifest.tone, 'direct');
    assert.deepEqual(result.manifest.conventions, ['use constructor injection']);
  });

  it('falls back to manifest.json when no yaml', () => {
    const typeDir = join(tempDir, 'context', 'react');
    mkdirSync(typeDir, { recursive: true });
    writeFileSync(join(typeDir, 'manifest.json'), JSON.stringify({
      name: 'react',
      type: 'context',
      version: '1.0.0',
      description: 'React patterns',
      format: 'markdown',
      sources: ['patterns.md'],
    }));

    const result = loadManifest(tempDir, 'context/react');
    assert.ok(result);
    assert.equal(result.manifest.name, 'react');
    assert.equal(result.manifest.format, 'markdown');
  });

  it('prefers manifest.yaml over manifest.json', () => {
    const typeDir = join(tempDir, 'personas', 'test');
    mkdirSync(typeDir, { recursive: true });
    writeFileSync(join(typeDir, 'manifest.yaml'), 'name: from-yaml\ntype: persona\nversion: "1.0.0"\ndescription: yaml version\n');
    writeFileSync(join(typeDir, 'manifest.json'), JSON.stringify({ name: 'from-json' }));

    const result = loadManifest(tempDir, 'personas/test');
    assert.equal(result.manifest.name, 'from-yaml');
  });

  it('returns null when no manifest exists', () => {
    const typeDir = join(tempDir, 'personas', 'nonexistent');
    mkdirSync(typeDir, { recursive: true });

    const result = loadManifest(tempDir, 'personas/nonexistent');
    assert.equal(result, null);
  });

  it('returns null when type directory does not exist', () => {
    const result = loadManifest(tempDir, 'personas/missing');
    assert.equal(result, null);
  });
});

describe('createSymlink', () => {
  it('creates a symlink to the target', () => {
    const target = join(tempDir, 'target-dir');
    mkdirSync(target);
    writeFileSync(join(target, 'file.txt'), 'hello');

    const linkPath = join(tempDir, 'output', 'my-link');
    createSymlink(target, linkPath);

    assert.ok(existsSync(linkPath));
    assert.equal(readlinkSync(linkPath), target);
  });

  it('replaces an existing symlink', () => {
    const target1 = join(tempDir, 'target1');
    const target2 = join(tempDir, 'target2');
    mkdirSync(target1);
    mkdirSync(target2);

    const linkPath = join(tempDir, 'link');
    symlinkSync(target1, linkPath);
    assert.equal(readlinkSync(linkPath), target1);

    createSymlink(target2, linkPath);
    assert.equal(readlinkSync(linkPath), target2);
  });

  it('creates parent directories as needed', () => {
    const target = join(tempDir, 'target');
    mkdirSync(target);

    const linkPath = join(tempDir, 'deep', 'nested', 'link');
    createSymlink(target, linkPath);

    assert.ok(existsSync(linkPath));
  });
});

describe('flattenRef', () => {
  it('strips type prefix and joins with hyphens', () => {
    assert.equal(flattenRef('context/spring-boot/error-handling'), 'spring-boot-error-handling');
  });

  it('handles single-level after prefix', () => {
    assert.equal(flattenRef('context/react'), 'react');
  });

  it('handles deep nesting', () => {
    assert.equal(flattenRef('skills/scm/git/commit-analyzer'), 'scm-git-commit-analyzer');
  });

  it('returns as-is when no slash', () => {
    assert.equal(flattenRef('something'), 'something');
  });
});

describe('isStale', () => {
  it('returns false when source does not exist', () => {
    assert.equal(isStale(join(tempDir, 'nonexistent'), []), false);
  });

  it('returns true when generated file does not exist', () => {
    const source = join(tempDir, 'source.yaml');
    writeFileSync(source, 'data: true');

    assert.equal(isStale(source, [join(tempDir, 'missing.md')]), true);
  });

  it('returns true when source is newer than generated file', () => {
    const generated = join(tempDir, 'output.md');
    writeFileSync(generated, 'old content');
    // Set generated file to the past
    const past = new Date(Date.now() - 10000);
    utimesSync(generated, past, past);

    const source = join(tempDir, 'source.yaml');
    writeFileSync(source, 'data: true');

    assert.equal(isStale(source, [generated]), true);
  });

  it('returns false when generated files are newer', () => {
    const source = join(tempDir, 'source.yaml');
    writeFileSync(source, 'data: true');
    // Set source to the past
    const past = new Date(Date.now() - 10000);
    utimesSync(source, past, past);

    const generated = join(tempDir, 'output.md');
    writeFileSync(generated, 'new content');

    assert.equal(isStale(source, [generated]), false);
  });
});

describe('ensureDir', () => {
  it('creates nested directories', () => {
    const dir = join(tempDir, 'a', 'b', 'c');
    ensureDir(dir);
    assert.ok(existsSync(dir));
  });

  it('does not throw on existing directory', () => {
    const dir = join(tempDir, 'exists');
    mkdirSync(dir);
    assert.doesNotThrow(() => ensureDir(dir));
  });
});

describe('validateSymlinks', () => {
  it('returns zeros for nonexistent directory', () => {
    const result = validateSymlinks(join(tempDir, 'nonexistent'));
    assert.deepEqual(result, { total: 0, valid: 0, broken: [] });
  });

  it('counts valid symlinks', () => {
    const dir = join(tempDir, 'links');
    mkdirSync(dir);
    const target = join(tempDir, 'real-target');
    mkdirSync(target);

    symlinkSync(target, join(dir, 'link1'));
    symlinkSync(target, join(dir, 'link2'));

    const result = validateSymlinks(dir);
    assert.equal(result.total, 2);
    assert.equal(result.valid, 2);
    assert.deepEqual(result.broken, []);
  });

  it('detects broken symlinks', () => {
    const dir = join(tempDir, 'links');
    mkdirSync(dir);

    // Create a symlink to a path that doesn't exist
    symlinkSync(join(tempDir, 'nonexistent-target'), join(dir, 'broken'));

    const target = join(tempDir, 'real-target');
    mkdirSync(target);
    symlinkSync(target, join(dir, 'valid'));

    const result = validateSymlinks(dir);
    assert.equal(result.total, 2);
    assert.equal(result.valid, 1);
    assert.deepEqual(result.broken, ['broken']);
  });

  it('ignores non-symlink entries', () => {
    const dir = join(tempDir, 'links');
    mkdirSync(dir);
    writeFileSync(join(dir, 'regular-file.txt'), 'not a symlink');

    const result = validateSymlinks(dir);
    assert.equal(result.total, 0);
    assert.equal(result.valid, 0);
  });
});
