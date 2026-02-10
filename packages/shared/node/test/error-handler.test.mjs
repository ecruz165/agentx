import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { AgentXError, handleError, wrapAsync } from '../src/error-handler.mjs';

describe('AgentXError', () => {
  it('should have correct name, message, code, and details', () => {
    const err = new AgentXError('Test failed', 'TEST_ERR', { field: 'x' });
    assert.equal(err.name, 'AgentXError');
    assert.equal(err.message, 'Test failed');
    assert.equal(err.code, 'TEST_ERR');
    assert.deepEqual(err.details, { field: 'x' });
    assert.ok(err instanceof Error);
    assert.ok(err instanceof AgentXError);
  });

  it('should work without code and details', () => {
    const err = new AgentXError('Simple error');
    assert.equal(err.message, 'Simple error');
    assert.equal(err.code, undefined);
    assert.equal(err.details, undefined);
  });
});

describe('handleError', () => {
  it('should return error envelope when exit is false', () => {
    const err = new AgentXError('Bad request', 'BAD_REQUEST', { input: 'abc' });
    const result = handleError(err, { exit: false });
    assert.equal(result.success, false);
    assert.equal(result.error.message, 'Bad request');
    assert.equal(result.error.code, 'BAD_REQUEST');
    assert.deepEqual(result.error.details, { input: 'abc' });
  });

  it('should use UNKNOWN_ERROR code for plain errors', () => {
    const err = new Error('generic error');
    const result = handleError(err, { exit: false });
    assert.equal(result.error.code, 'UNKNOWN_ERROR');
  });
});

describe('wrapAsync', () => {
  it('should pass through successful results', async () => {
    const fn = async (x) => x * 2;
    const wrapped = wrapAsync(fn, { exit: false });
    const result = await wrapped(5);
    assert.equal(result, 10);
  });

  it('should catch errors and return envelope', async () => {
    const fn = async () => {
      throw new AgentXError('Async failure', 'ASYNC_ERR');
    };
    const wrapped = wrapAsync(fn, { exit: false });
    const result = await wrapped();
    assert.equal(result.success, false);
    assert.equal(result.error.message, 'Async failure');
    assert.equal(result.error.code, 'ASYNC_ERR');
  });

  it('should preserve function arguments', async () => {
    const fn = async (a, b, c) => {
      throw new AgentXError(`args: ${a},${b},${c}`, 'ARG_ERR');
    };
    const wrapped = wrapAsync(fn, { exit: false });
    const result = await wrapped(1, 'two', true);
    assert.equal(result.error.message, 'args: 1,two,true');
  });
});
