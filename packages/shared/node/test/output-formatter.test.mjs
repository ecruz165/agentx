import { describe, it } from 'node:test';
import assert from 'node:assert/strict';
import { parse } from 'yaml';
import { formatOutput, formatSuccess, formatError, formatTable } from '../src/output-formatter.mjs';

describe('formatOutput', () => {
  const data = { name: 'test', count: 42, items: ['a', 'b'] };

  it('should format data as JSON by default', () => {
    const result = formatOutput(data);
    const parsed = JSON.parse(result);
    assert.deepEqual(parsed, data);
  });

  it('should format data as YAML', () => {
    const result = formatOutput(data, 'yaml');
    const parsed = parse(result);
    assert.deepEqual(parsed, data);
  });

  it('should format array of objects as table', () => {
    const rows = [
      { name: 'git', version: '2.41.0', status: 'ok' },
      { name: 'aws', version: '2.15.0', status: 'outdated' },
    ];
    const result = formatOutput(rows, 'table');
    assert.ok(result.includes('name'));
    assert.ok(result.includes('git'));
    assert.ok(result.includes('aws'));
    // Verify aligned columns (header separator exists)
    const lines = result.split('\n');
    assert.ok(lines[1].includes('---'));
  });

  it('should throw on unknown format', () => {
    assert.throws(() => formatOutput(data, 'xml'), /Unknown output format: xml/);
  });

  it('should handle empty array in table format', () => {
    const result = formatOutput([], 'table');
    assert.equal(result, '[]');
  });
});

describe('formatSuccess', () => {
  it('should create success envelope', () => {
    const result = formatSuccess('Operation completed', { id: 1 });
    assert.equal(result.success, true);
    assert.equal(result.message, 'Operation completed');
    assert.deepEqual(result.data, { id: 1 });
  });

  it('should handle missing data', () => {
    const result = formatSuccess('Done');
    assert.equal(result.success, true);
    assert.equal(result.data, undefined);
  });
});

describe('formatError', () => {
  it('should create error envelope from Error instance', () => {
    const err = new Error('Something failed');
    const result = formatError(err, 'TEST_ERROR');
    assert.equal(result.success, false);
    assert.equal(result.error.message, 'Something failed');
    assert.equal(result.error.code, 'TEST_ERROR');
  });

  it('should create error envelope from string', () => {
    const result = formatError('Bad input', 'INVALID_INPUT');
    assert.equal(result.success, false);
    assert.equal(result.error.message, 'Bad input');
    assert.equal(result.error.code, 'INVALID_INPUT');
  });

  it('should default code to UNKNOWN_ERROR', () => {
    const result = formatError('Oops');
    assert.equal(result.error.code, 'UNKNOWN_ERROR');
  });

  it('should include details when provided', () => {
    const result = formatError('Fail', 'ERR', { field: 'name' });
    assert.deepEqual(result.error.details, { field: 'name' });
  });
});

describe('formatTable', () => {
  it('should align columns with space padding', () => {
    const headers = ['Name', 'Version', 'Status'];
    const rows = [
      ['git', '2.41.0', 'ok'],
      ['aws-cli', '2.15.0', 'outdated'],
    ];
    const result = formatTable(headers, rows);
    const lines = result.split('\n');
    assert.equal(lines.length, 4); // header + separator + 2 data rows
    // All lines should have consistent column positions
    assert.ok(lines[1].includes('---'));
  });

  it('should handle empty rows', () => {
    const headers = ['A', 'B'];
    const result = formatTable(headers, []);
    const lines = result.split('\n');
    assert.equal(lines.length, 2); // header + separator
  });

  it('should handle varying cell widths', () => {
    const headers = ['X'];
    const rows = [['short'], ['a much longer value']];
    const result = formatTable(headers, rows);
    const lines = result.split('\n');
    // Separator should match longest value
    assert.ok(lines[1].length >= 'a much longer value'.length);
  });
});
