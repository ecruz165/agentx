import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { ToolManager } from '../src/tool-manager.mjs';
import { VERSION_OUTPUTS } from './fixtures/version-outputs.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REGISTRY_DIR = join(__dirname, '..', 'registry');

/**
 * Create a mock shell that returns results based on command+args key.
 */
function createMockShell(responseMap) {
  return async (command, args) => {
    const key = `${command} ${args.join(' ')}`;
    // Check for exact match first
    if (responseMap.has(key)) {
      return responseMap.get(key);
    }
    // Check for partial matches (e.g., 'which brew')
    for (const [pattern, result] of responseMap) {
      if (key.startsWith(pattern)) return result;
    }
    return VERSION_OUTPUTS.notFound;
  };
}

describe('ToolManager', () => {
  describe('check()', () => {
    it('should detect an installed tool with version', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.git],
        ['which brew', VERSION_OUTPUTS.whichFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.check('git');
      assert.equal(result.installed, true);
      assert.equal(result.version, '2.41.0');
      assert.equal(result.meetsMinimum, true);
    });

    it('should detect a missing tool', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.check('git');
      assert.equal(result.installed, false);
    });

    it('should detect an outdated tool', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.gitOld],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.check('git', { minVersion: '2.30.0' });
      assert.equal(result.installed, true);
      assert.equal(result.version, '2.20.0');
      assert.equal(result.meetsMinimum, false);
    });

    it('should return error for unknown tool', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.check('nonexistent-tool');
      assert.equal(result.installed, false);
      assert.ok(result.error);
    });

    it('should cache results', async () => {
      let callCount = 0;
      const mockShell = async (command, args) => {
        callCount++;
        if (command === 'git') return VERSION_OUTPUTS.git;
        return VERSION_OUTPUTS.notFound;
      };
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      await tm.check('git');
      await tm.check('git');
      // Should only call shell once (second call from cache)
      // Registry load also calls shell zero times, so count = 1
      assert.equal(callCount, 1);
    });

    it('should respect clearCache', async () => {
      let callCount = 0;
      const mockShell = async (command, args) => {
        callCount++;
        if (command === 'git') return VERSION_OUTPUTS.git;
        return VERSION_OUTPUTS.notFound;
      };
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      await tm.check('git');
      tm.clearCache();
      await tm.check('git');
      assert.equal(callCount, 2);
    });
  });

  describe('checkAll()', () => {
    it('should return satisfied when all deps are met', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.git],
        ['gh --version', VERSION_OUTPUTS.gh],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.checkAll([
        { name: 'git', minVersion: '2.30.0' },
        { name: 'gh', minVersion: '2.0.0' },
      ]);
      assert.equal(result.satisfied, true);
      assert.equal(result.missing.length, 0);
      assert.equal(result.outdated.length, 0);
    });

    it('should report missing tools', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.git],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.checkAll([
        { name: 'git' },
        { name: 'aws' },
      ]);
      assert.equal(result.satisfied, false);
      assert.ok(result.missing.includes('aws'));
    });

    it('should report outdated tools', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.gitOld],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.checkAll([
        { name: 'git', minVersion: '2.30.0' },
      ]);
      assert.equal(result.satisfied, false);
      assert.ok(result.outdated.includes('git'));
    });

    it('should handle mixed missing and outdated', async () => {
      const mockShell = createMockShell(new Map([
        ['git --version', VERSION_OUTPUTS.gitOld],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.checkAll([
        { name: 'git', minVersion: '2.30.0' },
        { name: 'aws' },
      ]);
      assert.equal(result.satisfied, false);
      assert.ok(result.outdated.includes('git'));
      assert.ok(result.missing.includes('aws'));
    });
  });

  describe('detectPackageManager()', () => {
    it('should detect homebrew on macOS when brew is available', async () => {
      const mockShell = createMockShell(new Map([
        ['which brew', VERSION_OUTPUTS.whichFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      // On macOS (darwin), this test will work directly
      if (process.platform === 'darwin') {
        const pm = await tm.detectPackageManager();
        assert.equal(pm.name, 'homebrew');
        assert.equal(pm.available, true);
      }
    });

    it('should fall back to manual when no package manager is found', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const pm = await tm.detectPackageManager();
      // On macOS, homebrew detection will fail, fallback to manual
      // On other platforms, equivalent behavior
      assert.ok(pm.name);
      assert.equal(pm.available, true);
    });
  });

  describe('getInstallCommand()', () => {
    it('should return install command for known tool', async () => {
      const mockShell = createMockShell(new Map([
        ['which brew', VERSION_OUTPUTS.whichFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const cmd = await tm.getInstallCommand('git');
      assert.ok(cmd.method);
      // If homebrew detected, should return brew command
      if (cmd.method === 'homebrew') {
        assert.equal(cmd.command, 'brew install git');
      }
    });

    it('should return manual instructions for unknown tool', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const cmd = await tm.getInstallCommand('nonexistent');
      assert.equal(cmd.method, 'manual');
    });

    it('should fall back to manual when pm has no config for tool', async () => {
      const mockShell = createMockShell(new Map([
        ['which brew', VERSION_OUTPUTS.whichNotFound],
        ['which apt-get', VERSION_OUTPUTS.whichNotFound],
        ['where winget', VERSION_OUTPUTS.whichNotFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const cmd = await tm.getInstallCommand('git');
      assert.equal(cmd.method, 'manual');
      assert.ok(cmd.url);
    });
  });

  describe('getToolDefinition()', () => {
    it('should return tool definition', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const def = await tm.getToolDefinition('git');
      assert.ok(def);
      assert.equal(def.name, 'git');
    });

    it('should return undefined for unknown tool', async () => {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const def = await tm.getToolDefinition('nonexistent');
      assert.equal(def, undefined);
    });
  });
});
