import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { HomebrewDetector } from '../src/detectors/homebrew.mjs';
import { WingetDetector } from '../src/detectors/winget.mjs';
import { AptDetector } from '../src/detectors/apt.mjs';
import { ManualDetector } from '../src/detectors/manual.mjs';
import { BaseDetector } from '../src/detectors/base.mjs';
import { VERSION_OUTPUTS } from './fixtures/version-outputs.mjs';

/**
 * Create a mock shell executor that returns predefined results.
 * @param {Map<string, import('../src/shell.mjs').ShellResult>} responseMap
 * @returns {import('../src/shell.mjs').ShellExecutor}
 */
function createMockShell(responseMap) {
  return async (command, args) => {
    const key = `${command} ${args.join(' ')}`;
    return responseMap.get(key) || VERSION_OUTPUTS.whichNotFound;
  };
}

describe('BaseDetector', () => {
  it('should throw if shellExecutor is not provided', () => {
    assert.throws(() => new BaseDetector(null), /shellExecutor is required/);
  });

  it('should throw on unimplemented isAvailable', async () => {
    const mockShell = createMockShell(new Map());
    const detector = new BaseDetector(mockShell);
    await assert.rejects(() => detector.isAvailable(), /must be implemented/);
  });

  it('should throw on unimplemented name', () => {
    const mockShell = createMockShell(new Map());
    const detector = new BaseDetector(mockShell);
    assert.throws(() => detector.name, /must be implemented/);
  });
});

describe('HomebrewDetector', () => {
  it('should detect Homebrew when brew is available', async () => {
    const mockShell = createMockShell(new Map([
      ['which brew', VERSION_OUTPUTS.whichFound],
    ]));
    const detector = new HomebrewDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, true);
  });

  it('should return failure when brew is not available', async () => {
    const mockShell = createMockShell(new Map());
    const detector = new HomebrewDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, false);
  });

  it('should have name "homebrew"', () => {
    const mockShell = createMockShell(new Map());
    const detector = new HomebrewDetector(mockShell);
    assert.equal(detector.name, 'homebrew');
  });
});

describe('WingetDetector', () => {
  it('should detect winget when available', async () => {
    const mockShell = createMockShell(new Map([
      ['where winget', VERSION_OUTPUTS.whichFound],
    ]));
    const detector = new WingetDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, true);
  });

  it('should return failure when winget is not available', async () => {
    const mockShell = createMockShell(new Map());
    const detector = new WingetDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, false);
  });

  it('should have name "winget"', () => {
    const mockShell = createMockShell(new Map());
    const detector = new WingetDetector(mockShell);
    assert.equal(detector.name, 'winget');
  });
});

describe('AptDetector', () => {
  it('should detect apt when apt-get is available', async () => {
    const mockShell = createMockShell(new Map([
      ['which apt-get', VERSION_OUTPUTS.whichFound],
    ]));
    const detector = new AptDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, true);
  });

  it('should return failure when apt-get is not available', async () => {
    const mockShell = createMockShell(new Map());
    const detector = new AptDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, false);
  });

  it('should have name "apt"', () => {
    const mockShell = createMockShell(new Map());
    const detector = new AptDetector(mockShell);
    assert.equal(detector.name, 'apt');
  });
});

describe('ManualDetector', () => {
  it('should always be available', async () => {
    const mockShell = createMockShell(new Map());
    const detector = new ManualDetector(mockShell);
    const result = await detector.isAvailable();
    assert.equal(result.success, true);
  });

  it('should have name "manual"', () => {
    const mockShell = createMockShell(new Map());
    const detector = new ManualDetector(mockShell);
    assert.equal(detector.name, 'manual');
  });
});
