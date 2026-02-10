import { execFile as nodeExecFile } from 'node:child_process';
import { promisify } from 'node:util';
import { parse as parseYaml } from 'yaml';
import { AgentXError } from './error-handler.mjs';

const execFileAsync = promisify(nodeExecFile);

/**
 * @typedef {object} RunCommandOptions
 * @property {number} [timeout=30000] - Timeout in ms
 * @property {string} [cwd] - Working directory
 * @property {object} [env] - Additional env vars (merged with process.env)
 */

/**
 * @typedef {object} RunCommandResult
 * @property {string} stdout
 * @property {string} stderr
 * @property {number} exitCode
 */

/**
 * Run a command and capture output.
 * Uses execFile (safe, no shell injection).
 * @param {string} command
 * @param {string[]} [args=[]]
 * @param {RunCommandOptions} [options={}]
 * @returns {Promise<RunCommandResult>}
 */
export async function runCommand(command, args = [], options = {}) {
  const { timeout = 30000, cwd, env } = options;
  const execOptions = {
    timeout,
    cwd,
    env: env ? { ...process.env, ...env } : process.env,
  };

  try {
    const { stdout, stderr } = await execFileAsync(command, args, execOptions);
    return { stdout: stdout.toString(), stderr: stderr.toString(), exitCode: 0 };
  } catch (err) {
    if (err.killed && err.signal === 'SIGTERM') {
      throw new AgentXError(
        `Command timed out after ${timeout}ms: ${command}`,
        'COMMAND_TIMEOUT',
        { command, args, timeout }
      );
    }
    if (err.code === 'ENOENT') {
      throw new AgentXError(
        `Command not found: ${command}`,
        'COMMAND_NOT_FOUND',
        { command, args }
      );
    }
    // Non-zero exit code — return result, don't throw
    const exitCode = typeof err.code === 'number' ? err.code : (err.status ?? 1);
    return {
      stdout: err.stdout ? err.stdout.toString() : '',
      stderr: err.stderr ? err.stderr.toString() : err.message,
      exitCode,
    };
  }
}

/**
 * Parse command output in a given format.
 * @param {string} output
 * @param {'json' | 'yaml' | 'raw'} [format='raw']
 * @returns {any}
 */
export function parseOutput(output, format = 'raw') {
  if (format === 'raw') {
    return output;
  }
  try {
    if (format === 'json') {
      return JSON.parse(output);
    }
    if (format === 'yaml') {
      return parseYaml(output);
    }
    return output;
  } catch (err) {
    throw new AgentXError(
      `Failed to parse output as ${format}: ${err.message}`,
      'PARSE_ERROR',
      { format, output: output.slice(0, 200) }
    );
  }
}
