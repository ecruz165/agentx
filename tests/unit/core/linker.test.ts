import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, writeFileSync, readFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  initProject,
  loadProject,
  projectConfigPath,
} from '../../../src/core/linker.js';

describe('linker', () => {
  let projectDir: string;

  beforeEach(() => {
    projectDir = join(tmpdir(), `agentx-linker-test-${Date.now()}`);
    mkdirSync(projectDir, { recursive: true });
  });

  afterEach(() => {
    rmSync(projectDir, { recursive: true, force: true });
  });

  describe('initProject', () => {
    it('creates project config with tools', () => {
      initProject(projectDir, ['claude-code', 'copilot']);
      const config = loadProject(projectDir);
      expect(config.tools).toEqual(['claude-code', 'copilot']);
      expect(config.active.personas).toEqual([]);
      expect(config.active.skills).toEqual([]);
    });

    it('creates .agentx directory and overrides', () => {
      initProject(projectDir, ['claude-code']);
      const { existsSync } = require('node:fs');
      expect(existsSync(join(projectDir, '.agentx'))).toBe(true);
      expect(existsSync(join(projectDir, '.agentx', 'overrides'))).toBe(true);
    });
  });

  describe('loadProject', () => {
    it('loads project config', () => {
      initProject(projectDir, ['augment']);
      const config = loadProject(projectDir);
      expect(config.tools).toEqual(['augment']);
    });
  });

  describe('projectConfigPath', () => {
    it('returns correct path', () => {
      expect(projectConfigPath('/test')).toBe('/test/.agentx/project.yaml');
    });
  });
});
