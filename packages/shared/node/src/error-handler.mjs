import { formatError } from './output-formatter.mjs';

/**
 * Custom error class with code and details.
 */
export class AgentXError extends Error {
  /**
   * @param {string} message
   * @param {string} [code]
   * @param {any} [details]
   */
  constructor(message, code, details) {
    super(message);
    this.name = 'AgentXError';
    this.code = code;
    this.details = details;
  }
}

/**
 * Handle an error: format it and optionally exit.
 * @param {Error} error
 * @param {{ exit?: boolean }} [options] - Default { exit: true }
 * @returns {{ success: false, error: { message: string, code: string, details: any } } | never}
 */
export function handleError(error, options = {}) {
  const code = error instanceof AgentXError ? error.code : 'UNKNOWN_ERROR';
  const details = error instanceof AgentXError ? error.details : undefined;
  const envelope = formatError(error, code, details);

  if (options.exit === false) {
    return envelope;
  }

  process.stderr.write(JSON.stringify(envelope, null, 2) + '\n');
  process.exit(1);
}

/**
 * Wrap an async function with error handling.
 * @param {Function} fn
 * @param {{ exit?: boolean }} [options]
 * @returns {Function}
 */
export function wrapAsync(fn, options = {}) {
  return async function (...args) {
    try {
      return await fn(...args);
    } catch (error) {
      return handleError(error, options);
    }
  };
}
