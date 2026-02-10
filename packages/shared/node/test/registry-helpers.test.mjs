import { describe, it, before, after } from 'node:test';
import assert from 'node:assert/strict';
import { mkdtemp, rm, readFile, readdir } from 'node:fs/promises';
import { writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  getUserdataRoot,
  getSkillRegistryPath,
  loadEnvChain,
  readState,
  writeState,
  saveOutput,
  loadTemplate,
  saveTemplate,
  listTemplates,
  readConfig,
} from '../src/registry-helpers.mjs';

describe('getUserdataRoot', () => {
  const originalEnv = process.env.AGENTX_USERDATA;

  after(() => {
    if (originalEnv !== undefined) {
      process.env.AGENTX_USERDATA = originalEnv;
    } else {
      delete process.env.AGENTX_USERDATA;
    }
  });

  it('should respect AGENTX_USERDATA env var', () => {
    process.env.AGENTX_USERDATA = '/custom/path';
    assert.equal(getUserdataRoot(), '/custom/path');
  });

  it('should fall back to ~/.agentx/userdata', () => {
    delete process.env.AGENTX_USERDATA;
    const result = getUserdataRoot();
    assert.ok(result.endsWith('.agentx/userdata'));
  });
});

describe('getSkillRegistryPath', () => {
  const originalEnv = process.env.AGENTX_USERDATA;

  before(() => {
    process.env.AGENTX_USERDATA = '/test/userdata';
  });

  after(() => {
    if (originalEnv !== undefined) {
      process.env.AGENTX_USERDATA = originalEnv;
    } else {
      delete process.env.AGENTX_USERDATA;
    }
  });

  it('should build path with vendor', () => {
    const result = getSkillRegistryPath('cloud', 'aws', 'ssm-lookup');
    assert.equal(result, '/test/userdata/skills/cloud/aws/ssm-lookup');
  });

  it('should build path without vendor', () => {
    const result = getSkillRegistryPath('ai', null, 'token-counter');
    assert.equal(result, '/test/userdata/skills/ai/token-counter');
  });

  it('should build path with empty vendor string', () => {
    const result = getSkillRegistryPath('scm', '', 'commit-analyzer');
    assert.equal(result, '/test/userdata/skills/scm/commit-analyzer');
  });
});

describe('loadEnvChain', () => {
  let tmpDir;
  const originalEnv = process.env.AGENTX_USERDATA;
  const savedVars = {};

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
    process.env.AGENTX_USERDATA = tmpDir;

    // Create env directory structure
    mkdirSync(join(tmpDir, 'env'), { recursive: true });
    mkdirSync(join(tmpDir, 'skills', 'cloud', 'aws', 'test-skill'), { recursive: true });

    writeFileSync(join(tmpDir, 'env', 'default.env'), 'DEFAULT_VAR=from_default\nSHARED_VAR=default_value\n');
    writeFileSync(join(tmpDir, 'env', 'aws.env'), 'VENDOR_VAR=from_aws\nSHARED_VAR=vendor_value\n');
    writeFileSync(
      join(tmpDir, 'skills', 'cloud', 'aws', 'test-skill', 'tokens.env'),
      'SKILL_VAR=from_skill\nSHARED_VAR=skill_value\n'
    );

    // Save original env vars
    for (const key of ['DEFAULT_VAR', 'VENDOR_VAR', 'SKILL_VAR', 'SHARED_VAR']) {
      savedVars[key] = process.env[key];
    }
  });

  after(async () => {
    // Restore env
    if (originalEnv !== undefined) {
      process.env.AGENTX_USERDATA = originalEnv;
    } else {
      delete process.env.AGENTX_USERDATA;
    }
    for (const [key, val] of Object.entries(savedVars)) {
      if (val !== undefined) {
        process.env[key] = val;
      } else {
        delete process.env[key];
      }
    }
    await rm(tmpDir, { recursive: true });
  });

  it('should load env files in resolution order', () => {
    const skillPath = join(tmpDir, 'skills', 'cloud', 'aws', 'test-skill');
    loadEnvChain(skillPath, 'aws');

    assert.equal(process.env.DEFAULT_VAR, 'from_default');
    assert.equal(process.env.VENDOR_VAR, 'from_aws');
    assert.equal(process.env.SKILL_VAR, 'from_skill');
    // Skill-specific should override vendor which overrides default
    assert.equal(process.env.SHARED_VAR, 'skill_value');
  });
});

describe('readState / writeState', () => {
  let tmpDir;

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
  });

  after(async () => {
    await rm(tmpDir, { recursive: true });
  });

  it('should return null for missing state file', () => {
    const result = readState(tmpDir, 'nonexistent.json');
    assert.equal(result, null);
  });

  it('should write and read state round-trip', () => {
    const data = { lastRun: '2026-02-09', count: 5 };
    writeState(tmpDir, 'cache.json', data);
    const result = readState(tmpDir, 'cache.json');
    assert.deepEqual(result, data);
  });
});

describe('saveOutput', () => {
  let tmpDir;

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
  });

  after(async () => {
    await rm(tmpDir, { recursive: true });
  });

  it('should create latest.json and a timestamped file', async () => {
    const data = { results: [1, 2, 3] };
    saveOutput(tmpDir, data);

    const outputDir = join(tmpDir, 'output');
    const files = await readdir(outputDir);
    assert.ok(files.includes('latest.json'));
    assert.ok(files.length >= 2, 'Should have latest.json + timestamped file');

    const latest = JSON.parse(await readFile(join(outputDir, 'latest.json'), 'utf8'));
    assert.deepEqual(latest, data);
  });
});

describe('loadTemplate / saveTemplate', () => {
  let tmpDir;

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
  });

  after(async () => {
    await rm(tmpDir, { recursive: true });
  });

  it('should return null for missing template', () => {
    const result = loadTemplate(tmpDir, 'nonexistent.hbs');
    assert.equal(result, null);
  });

  it('should save and load template round-trip', () => {
    const content = '# Report\n{{#each items}}{{this}}{{/each}}';
    saveTemplate(tmpDir, 'report.hbs', content);
    const result = loadTemplate(tmpDir, 'report.hbs');
    assert.equal(result, content);
  });
});

describe('listTemplates', () => {
  let tmpDir;

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
  });

  after(async () => {
    await rm(tmpDir, { recursive: true });
  });

  it('should return empty array when templates dir does not exist', () => {
    const result = listTemplates(tmpDir);
    assert.deepEqual(result, []);
  });

  it('should list template filenames', () => {
    saveTemplate(tmpDir, 'first.spl', 'query 1');
    saveTemplate(tmpDir, 'second.spl', 'query 2');
    const result = listTemplates(tmpDir);
    assert.ok(result.includes('first.spl'));
    assert.ok(result.includes('second.spl'));
    assert.equal(result.length, 2);
  });
});

describe('readConfig', () => {
  let tmpDir;

  before(async () => {
    tmpDir = await mkdtemp(join(tmpdir(), 'agentx-test-'));
  });

  after(async () => {
    await rm(tmpDir, { recursive: true });
  });

  it('should return empty object when config.yaml does not exist', () => {
    const result = readConfig(tmpDir);
    assert.deepEqual(result, {});
  });

  it('should parse config.yaml', () => {
    writeFileSync(join(tmpDir, 'config.yaml'), 'cache_ttl_minutes: 30\nmax_results: 50\n');
    const result = readConfig(tmpDir);
    assert.deepEqual(result, { cache_ttl_minutes: 30, max_results: 50 });
  });
});
