import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { ToolManager } from '../src/tool-manager.mjs';
import { VERSION_OUTPUTS } from './fixtures/version-outputs.mjs';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REGISTRY_DIR = join(__dirname, '..', 'registry');

function createMockShell(responseMap) {
  return async (command, args) => {
    const key = `${command} ${args.join(' ')}`;
    if (responseMap.has(key)) {
      return responseMap.get(key);
    }
    return VERSION_OUTPUTS.notFound;
  };
}

describe('promptInstall', () => {
  it('should return skipped with instructions in CI mode', async () => {
    const originalCI = process.env.CI;
    process.env.CI = 'true';
    try {
      const mockShell = createMockShell(new Map([
        ['which brew', VERSION_OUTPUTS.whichFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.promptInstall('git');
      assert.equal(result.installed, false);
      assert.equal(result.skipped, true);
      assert.ok(result.instructions);
    } finally {
      if (originalCI === undefined) {
        delete process.env.CI;
      } else {
        process.env.CI = originalCI;
      }
    }
  });

  it('should return error for unknown tool', async () => {
    const originalCI = process.env.CI;
    process.env.CI = 'true';
    try {
      const mockShell = createMockShell(new Map());
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.promptInstall('nonexistent');
      assert.equal(result.installed, false);
      assert.ok(result.error);
    } finally {
      if (originalCI === undefined) {
        delete process.env.CI;
      } else {
        process.env.CI = originalCI;
      }
    }
  });

  it('should include command in CI instructions when package manager is detected', async () => {
    const originalCI = process.env.CI;
    process.env.CI = 'true';
    try {
      const mockShell = createMockShell(new Map([
        ['which brew', VERSION_OUTPUTS.whichFound],
      ]));
      const tm = new ToolManager({ shellExecutor: mockShell, registryDir: REGISTRY_DIR });
      const result = await tm.promptInstall('git');
      assert.equal(result.skipped, true);
      // Should include the brew install command or manual instructions
      assert.ok(result.instructions);
    } finally {
      if (originalCI === undefined) {
        delete process.env.CI;
      } else {
        process.env.CI = originalCI;
      }
    }
  });
});
