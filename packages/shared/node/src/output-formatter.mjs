import { stringify } from 'yaml';

/**
 * Format data as a string in the given format.
 * @param {any} data
 * @param {'json' | 'yaml' | 'table'} [format='json']
 * @returns {string}
 */
export function formatOutput(data, format = 'json') {
  switch (format) {
    case 'json':
      return JSON.stringify(data, null, 2);
    case 'yaml':
      return stringify(data);
    case 'table':
      return formatTableFromData(data);
    default:
      throw new Error(`Unknown output format: ${format}`);
  }
}

/**
 * Create a success envelope.
 * @param {string} message
 * @param {any} [data]
 * @returns {{ success: true, message: string, data: any }}
 */
export function formatSuccess(message, data) {
  return { success: true, message, data };
}

/**
 * Create an error envelope.
 * @param {Error | string} error
 * @param {string} [code]
 * @param {any} [details]
 * @returns {{ success: false, error: { message: string, code: string, details: any } }}
 */
export function formatError(error, code, details) {
  const message = error instanceof Error ? error.message : String(error);
  return {
    success: false,
    error: {
      message,
      code: code || (error instanceof Error && error.code) || 'UNKNOWN_ERROR',
      details: details || (error instanceof Error && error.details) || undefined,
    },
  };
}

/**
 * Format headers and rows as aligned columns with space padding.
 * @param {string[]} headers
 * @param {string[][]} rows
 * @returns {string}
 */
export function formatTable(headers, rows) {
  const allRows = [headers, ...rows];
  const colWidths = headers.map((_, i) =>
    Math.max(...allRows.map(row => String(row[i] ?? '').length))
  );
  const lines = allRows.map(row =>
    row.map((cell, i) => String(cell ?? '').padEnd(colWidths[i])).join('  ')
  );
  // Insert separator after header
  const separator = colWidths.map(w => '-'.repeat(w)).join('  ');
  lines.splice(1, 0, separator);
  return lines.join('\n');
}

/**
 * Internal: convert data to table format.
 * Handles arrays of objects (keys become headers) and arrays of arrays.
 * @param {any} data
 * @returns {string}
 */
function formatTableFromData(data) {
  if (!Array.isArray(data) || data.length === 0) {
    return JSON.stringify(data, null, 2);
  }
  // Array of objects
  if (typeof data[0] === 'object' && data[0] !== null && !Array.isArray(data[0])) {
    const headers = Object.keys(data[0]);
    const rows = data.map(item => headers.map(h => String(item[h] ?? '')));
    return formatTable(headers, rows);
  }
  // Array of arrays
  if (Array.isArray(data[0])) {
    const headers = data[0].map(String);
    const rows = data.slice(1).map(row => row.map(String));
    return formatTable(headers, rows);
  }
  // Fallback: array of primitives
  return data.map(String).join('\n');
}
