import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, '..');

function loadJSON(relativePath) {
  return JSON.parse(readFileSync(resolve(root, relativePath), 'utf8'));
}

// Set up AJV with 2020-12, strict mode, all errors, format validation
const ajv = new Ajv2020.default({ strict: true, allErrors: true });
addFormats.default(ajv);

const manifestSchema = loadJSON('manifest.schema.json');
const skillOutputSchema = loadJSON('skill-output.schema.json');
const skillErrorSchema = loadJSON('skill-error.schema.json');

const validateManifest = ajv.compile(manifestSchema);
const validateSkillOutput = ajv.compile(skillOutputSchema);
const validateSkillError = ajv.compile(skillErrorSchema);

// -- Valid manifest fixtures --------------------------------------------------

describe('manifest.schema.json - valid manifests', () => {
  it('validates a context manifest', () => {
    const data = loadJSON('test/fixtures/valid/context.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a persona manifest', () => {
    const data = loadJSON('test/fixtures/valid/persona.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a simple skill manifest', () => {
    const data = loadJSON('test/fixtures/valid/skill.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a complex skill manifest with tokens and templates', () => {
    const data = loadJSON('test/fixtures/valid/skill-complex.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a workflow manifest', () => {
    const data = loadJSON('test/fixtures/valid/workflow.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a prompt manifest', () => {
    const data = loadJSON('test/fixtures/valid/prompt.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });

  it('validates a template manifest', () => {
    const data = loadJSON('test/fixtures/valid/template.json');
    const valid = validateManifest(data);
    assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
  });
});

// -- Invalid manifest fixtures ------------------------------------------------

describe('manifest.schema.json - invalid manifests', () => {
  it('rejects a manifest missing the name field', () => {
    const data = loadJSON('test/fixtures/invalid/missing-name.json');
    const valid = validateManifest(data);
    assert.equal(valid, false);
  });

  it('rejects a manifest with invalid type enum', () => {
    const data = loadJSON('test/fixtures/invalid/bad-type.json');
    const valid = validateManifest(data);
    assert.equal(valid, false);
  });

  it('rejects a manifest with invalid version format', () => {
    const data = loadJSON('test/fixtures/invalid/bad-version.json');
    const valid = validateManifest(data);
    assert.equal(valid, false);
  });

  it('rejects a prompt with invalid reference paths', () => {
    const data = loadJSON('test/fixtures/invalid/bad-ref-path.json');
    const valid = validateManifest(data);
    assert.equal(valid, false);
  });

  it('rejects a skill missing the runtime field', () => {
    const data = loadJSON('test/fixtures/invalid/skill-missing-runtime.json');
    const valid = validateManifest(data);
    assert.equal(valid, false);
  });
});

// -- Type discriminator -------------------------------------------------------

describe('manifest.schema.json - type discriminator', () => {
  it('discriminates each of the 6 types correctly', () => {
    const types = ['context', 'persona', 'skill', 'workflow', 'prompt', 'template'];
    const files = {
      context: 'test/fixtures/valid/context.json',
      persona: 'test/fixtures/valid/persona.json',
      skill: 'test/fixtures/valid/skill.json',
      workflow: 'test/fixtures/valid/workflow.json',
      prompt: 'test/fixtures/valid/prompt.json',
      template: 'test/fixtures/valid/template.json',
    };

    for (const type of types) {
      const data = loadJSON(files[type]);
      assert.equal(data.type, type, `Fixture for ${type} has wrong type field`);
      const valid = validateManifest(data);
      assert.equal(valid, true, `${type} fixture failed: ${JSON.stringify(validateManifest.errors, null, 2)}`);
    }
  });
});

// -- Version pattern ----------------------------------------------------------

describe('manifest.schema.json - version patterns', () => {
  const baseContext = {
    name: 'test',
    type: 'context',
    description: 'test',
    format: 'markdown',
    sources: ['file.md'],
  };

  const validVersions = ['1.0.0', '1.0', '1', 'v1.2.3', '1.2.3-beta.1', '0.0.1-alpha', 'v2.0.0-rc.1'];

  for (const version of validVersions) {
    it(`accepts version: ${version}`, () => {
      const valid = validateManifest({ ...baseContext, version });
      assert.equal(valid, true, JSON.stringify(validateManifest.errors, null, 2));
    });
  }

  const invalidVersions = ['', 'abc', '1.2.3.4.5!', '-1.0', '.1.0'];

  for (const version of invalidVersions) {
    it(`rejects version: "${version}"`, () => {
      const valid = validateManifest({ ...baseContext, version });
      assert.equal(valid, false);
    });
  }
});

// -- skill-output.schema.json ------------------------------------------------

describe('skill-output.schema.json', () => {
  it('validates a valid skill output', () => {
    const data = loadJSON('test/fixtures/valid/skill-output.json');
    const valid = validateSkillOutput(data);
    assert.equal(valid, true, JSON.stringify(validateSkillOutput.errors, null, 2));
  });

  it('rejects output missing success field', () => {
    const valid = validateSkillOutput({
      data: {},
      metadata: { duration_ms: 100, timestamp: '2026-02-09T15:30:00.000Z' },
    });
    assert.equal(valid, false);
  });

  it('rejects output with invalid timestamp format', () => {
    const valid = validateSkillOutput({
      success: true,
      data: {},
      metadata: { duration_ms: 100, timestamp: 'not-a-date' },
    });
    assert.equal(valid, false);
  });

  it('rejects output with extra properties', () => {
    const valid = validateSkillOutput({
      success: true,
      data: {},
      metadata: { duration_ms: 100, timestamp: '2026-02-09T15:30:00.000Z' },
      extra: 'not allowed',
    });
    assert.equal(valid, false);
  });
});

// -- skill-error.schema.json -------------------------------------------------

describe('skill-error.schema.json', () => {
  it('validates a valid skill error', () => {
    const data = loadJSON('test/fixtures/valid/skill-error.json');
    const valid = validateSkillError(data);
    assert.equal(valid, true, JSON.stringify(validateSkillError.errors, null, 2));
  });

  it('rejects error missing code field', () => {
    const valid = validateSkillError({
      message: 'something failed',
      recoverable: false,
    });
    assert.equal(valid, false);
  });

  it('rejects error missing recoverable field', () => {
    const valid = validateSkillError({
      code: 'TEST_ERROR',
      message: 'something failed',
    });
    assert.equal(valid, false);
  });

  it('rejects error with extra properties', () => {
    const valid = validateSkillError({
      code: 'TEST_ERROR',
      message: 'something failed',
      recoverable: false,
      stackTrace: 'not allowed',
    });
    assert.equal(valid, false);
  });
});
