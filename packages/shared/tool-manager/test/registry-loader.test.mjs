import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { loadRegistry } from '../src/registry-loader.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REGISTRY_DIR = join(__dirname, '..', 'registry');

describe('loadRegistry', () => {
  it('should load all YAML files from the default registry', async () => {
    const registry = await loadRegistry(REGISTRY_DIR);
    assert.ok(registry instanceof Map);
    assert.ok(registry.size >= 6, `Expected at least 6 tools, got ${registry.size}`);
  });

  it('should load git tool definition correctly', async () => {
    const registry = await loadRegistry(REGISTRY_DIR);
    const git = registry.get('git');
    assert.ok(git, 'git should be in registry');
    assert.equal(git.name, 'git');
    assert.equal(git.display_name, 'Git');
    assert.ok(git.check.command);
    assert.ok(git.check.version_regex);
    assert.ok(git.check.min_version);
    assert.ok(git.install.manual);
    assert.ok(git.install.manual.url);
  });

  it('should load aws-cli tool definition correctly', async () => {
    const registry = await loadRegistry(REGISTRY_DIR);
    const aws = registry.get('aws');
    assert.ok(aws, 'aws should be in registry');
    assert.equal(aws.display_name, 'AWS CLI');
    assert.ok(aws.install.homebrew?.package);
    assert.ok(aws.install.apt?.commands, 'aws should have apt commands (multi-step)');
  });

  it('should load all core tools', async () => {
    const registry = await loadRegistry(REGISTRY_DIR);
    const expected = ['git', 'aws', 'gh', 'mvn', 'kubectl', 'docker'];
    for (const name of expected) {
      assert.ok(registry.has(name), `Registry should contain '${name}'`);
    }
  });

  it('should throw for invalid tool definitions', async () => {
    const tempDir = await mkdtemp(join(tmpdir(), 'tm-test-'));
    try {
      await writeFile(join(tempDir, 'bad.yaml'), 'name: bad\n');
      await assert.rejects(
        () => loadRegistry(tempDir),
        /missing required field/
      );
    } finally {
      await rm(tempDir, { recursive: true });
    }
  });

  it('should throw for YAML missing check.command', async () => {
    const tempDir = await mkdtemp(join(tmpdir(), 'tm-test-'));
    try {
      const yaml = [
        'name: test-tool',
        'check:',
        '  version_regex: "(\\\\d+)"',
        'install:',
        '  manual:',
        '    url: https://example.com',
        '    instructions: Install manually',
      ].join('\n');
      await writeFile(join(tempDir, 'test.yaml'), yaml);
      await assert.rejects(
        () => loadRegistry(tempDir),
        /check\.command/
      );
    } finally {
      await rm(tempDir, { recursive: true });
    }
  });

  it('should handle empty registry directory', async () => {
    const tempDir = await mkdtemp(join(tmpdir(), 'tm-test-'));
    try {
      const registry = await loadRegistry(tempDir);
      assert.equal(registry.size, 0);
    } finally {
      await rm(tempDir, { recursive: true });
    }
  });
});
