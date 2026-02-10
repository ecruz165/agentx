import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { HomebrewInstaller } from '../src/installers/homebrew.mjs';
import { WingetInstaller } from '../src/installers/winget.mjs';
import { AptInstaller } from '../src/installers/apt.mjs';
import { ManualInstaller } from '../src/installers/manual.mjs';
import { BaseInstaller } from '../src/installers/base.mjs';
import { VERSION_OUTPUTS } from './fixtures/version-outputs.mjs';

function createMockShell(responseMap) {
  return async (command, args) => {
    const key = `${command} ${args.join(' ')}`;
    return responseMap.get(key) || VERSION_OUTPUTS.installFailure;
  };
}

const sampleToolDef = {
  name: 'git',
  display_name: 'Git',
  install: {
    homebrew: { package: 'git' },
    winget: { package: 'Git.Git' },
    apt: { package: 'git' },
    manual: {
      url: 'https://git-scm.com/downloads',
      instructions: 'Download Git from the official website.',
    },
  },
};

const multiStepToolDef = {
  name: 'aws',
  display_name: 'AWS CLI',
  install: {
    apt: {
      commands: ['curl https://example.com -o aws.zip', 'unzip aws.zip', 'sudo ./install'],
    },
    manual: {
      url: 'https://example.com',
      instructions: 'Install manually.',
    },
  },
};

describe('BaseInstaller', () => {
  it('should throw if shellExecutor is not provided', () => {
    assert.throws(() => new BaseInstaller(null), /shellExecutor is required/);
  });

  it('should throw on unimplemented install', async () => {
    const mockShell = createMockShell(new Map());
    const installer = new BaseInstaller(mockShell);
    await assert.rejects(() => installer.install({}), /must be implemented/);
  });

  it('should throw on unimplemented getInstallCommand', () => {
    const mockShell = createMockShell(new Map());
    const installer = new BaseInstaller(mockShell);
    assert.throws(() => installer.getInstallCommand({}), /must be implemented/);
  });
});

describe('HomebrewInstaller', () => {
  it('should return correct install command', () => {
    const mockShell = createMockShell(new Map());
    const installer = new HomebrewInstaller(mockShell);
    const cmd = installer.getInstallCommand(sampleToolDef);
    assert.equal(cmd.method, 'homebrew');
    assert.equal(cmd.command, 'brew install git');
  });

  it('should install successfully', async () => {
    const mockShell = createMockShell(new Map([
      ['brew install git', VERSION_OUTPUTS.installSuccess],
    ]));
    const installer = new HomebrewInstaller(mockShell);
    const result = await installer.install(sampleToolDef);
    assert.equal(result.success, true);
  });

  it('should return failure on install error', async () => {
    const mockShell = createMockShell(new Map());
    const installer = new HomebrewInstaller(mockShell);
    const result = await installer.install(sampleToolDef);
    assert.equal(result.success, false);
  });

  it('should handle missing homebrew config', () => {
    const mockShell = createMockShell(new Map());
    const installer = new HomebrewInstaller(mockShell);
    const cmd = installer.getInstallCommand({ install: {} });
    assert.equal(cmd.command, null);
  });
});

describe('WingetInstaller', () => {
  it('should return correct install command', () => {
    const mockShell = createMockShell(new Map());
    const installer = new WingetInstaller(mockShell);
    const cmd = installer.getInstallCommand(sampleToolDef);
    assert.equal(cmd.method, 'winget');
    assert.equal(cmd.command, 'winget install Git.Git');
  });

  it('should install successfully', async () => {
    const mockShell = createMockShell(new Map([
      ['winget install Git.Git', VERSION_OUTPUTS.installSuccess],
    ]));
    const installer = new WingetInstaller(mockShell);
    const result = await installer.install(sampleToolDef);
    assert.equal(result.success, true);
  });
});

describe('AptInstaller', () => {
  it('should return correct install command for simple package', () => {
    const mockShell = createMockShell(new Map());
    const installer = new AptInstaller(mockShell);
    const cmd = installer.getInstallCommand(sampleToolDef);
    assert.equal(cmd.method, 'apt');
    assert.equal(cmd.command, 'sudo apt-get install -y git');
  });

  it('should return joined commands for multi-step installs', () => {
    const mockShell = createMockShell(new Map());
    const installer = new AptInstaller(mockShell);
    const cmd = installer.getInstallCommand(multiStepToolDef);
    assert.equal(cmd.method, 'apt');
    assert.ok(cmd.command.includes(' && '));
  });

  it('should install simple package successfully', async () => {
    const mockShell = createMockShell(new Map([
      ['sudo apt-get install -y git', VERSION_OUTPUTS.installSuccess],
    ]));
    const installer = new AptInstaller(mockShell);
    const result = await installer.install(sampleToolDef);
    assert.equal(result.success, true);
  });

  it('should execute multi-step commands sequentially', async () => {
    const executedCommands = [];
    const mockShell = async (command, args) => {
      executedCommands.push(`${command} ${args.join(' ')}`);
      return VERSION_OUTPUTS.installSuccess;
    };
    const installer = new AptInstaller(mockShell);
    const result = await installer.install(multiStepToolDef);
    assert.equal(result.success, true);
    assert.equal(executedCommands.length, 3);
  });

  it('should stop on first failure in multi-step', async () => {
    let callCount = 0;
    const mockShell = async () => {
      callCount++;
      if (callCount === 2) return VERSION_OUTPUTS.installFailure;
      return VERSION_OUTPUTS.installSuccess;
    };
    const installer = new AptInstaller(mockShell);
    const result = await installer.install(multiStepToolDef);
    assert.equal(result.success, false);
    assert.equal(callCount, 2);
  });
});

describe('ManualInstaller', () => {
  it('should return manual instructions', () => {
    const mockShell = createMockShell(new Map());
    const installer = new ManualInstaller(mockShell);
    const cmd = installer.getInstallCommand(sampleToolDef);
    assert.equal(cmd.method, 'manual');
    assert.ok(cmd.url);
    assert.ok(cmd.instructions);
  });

  it('should never actually install', async () => {
    const mockShell = createMockShell(new Map());
    const installer = new ManualInstaller(mockShell);
    const result = await installer.install(sampleToolDef);
    assert.equal(result.success, false);
    assert.ok(result.instructions);
  });
});
