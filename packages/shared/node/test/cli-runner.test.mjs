import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { runCommand, parseOutput } from '../src/cli-runner.mjs';
import { AgentXError } from '../src/error-handler.mjs';

describe('runCommand', () => {
  it('should capture stdout from echo', async () => {
    const result = await runCommand('echo', ['hello world']);
    assert.equal(result.stdout.trim(), 'hello world');
    assert.equal(result.exitCode, 0);
  });

  it('should capture stderr', async () => {
    // Use node to write to stderr
    const result = await runCommand('node', ['-e', 'process.stderr.write("err msg")']);
    assert.equal(result.stderr, 'err msg');
    assert.equal(result.exitCode, 0);
  });

  it('should return non-zero exit code without throwing', async () => {
    const result = await runCommand('node', ['-e', 'process.exit(42)']);
    assert.equal(result.exitCode, 42);
  });

  it('should throw COMMAND_NOT_FOUND for nonexistent command', async () => {
    await assert.rejects(
      () => runCommand('nonexistent_command_xyz_123'),
      (err) => {
        assert.ok(err instanceof AgentXError);
        assert.equal(err.code, 'COMMAND_NOT_FOUND');
        return true;
      }
    );
  });

  it('should throw COMMAND_TIMEOUT on timeout', async () => {
    await assert.rejects(
      () => runCommand('sleep', ['10'], { timeout: 100 }),
      (err) => {
        assert.ok(err instanceof AgentXError);
        assert.equal(err.code, 'COMMAND_TIMEOUT');
        return true;
      }
    );
  });

  it('should respect cwd option', async () => {
    const result = await runCommand('pwd', [], { cwd: '/tmp' });
    // On macOS /tmp is symlinked to /private/tmp
    assert.ok(
      result.stdout.trim() === '/tmp' || result.stdout.trim() === '/private/tmp',
      `Expected /tmp or /private/tmp, got ${result.stdout.trim()}`
    );
  });

  it('should merge env option with process.env', async () => {
    const result = await runCommand('node', ['-e', 'console.log(process.env.TEST_VAR_XYZ)'], {
      env: { TEST_VAR_XYZ: 'hello123' },
    });
    assert.equal(result.stdout.trim(), 'hello123');
  });
});

describe('parseOutput', () => {
  it('should parse JSON', () => {
    const result = parseOutput('{"a":1,"b":"two"}', 'json');
    assert.deepEqual(result, { a: 1, b: 'two' });
  });

  it('should parse YAML', () => {
    const result = parseOutput('name: test\ncount: 42\n', 'yaml');
    assert.deepEqual(result, { name: 'test', count: 42 });
  });

  it('should return raw string by default', () => {
    const result = parseOutput('just some text');
    assert.equal(result, 'just some text');
  });

  it('should return raw string for explicit raw format', () => {
    const result = parseOutput('raw text', 'raw');
    assert.equal(result, 'raw text');
  });

  it('should throw PARSE_ERROR on invalid JSON', () => {
    assert.throws(
      () => parseOutput('not json', 'json'),
      (err) => {
        assert.ok(err instanceof AgentXError);
        assert.equal(err.code, 'PARSE_ERROR');
        return true;
      }
    );
  });
});
