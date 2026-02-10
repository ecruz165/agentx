import { execFile as nodeExecFile } from 'node:child_process';
import { promisify } from 'node:util';

const execFileAsync = promisify(nodeExecFile);

/**
 * @typedef {object} ShellResult
 * @property {string} stdout
 * @property {string} stderr
 * @property {number} exitCode
 */

/**
 * @typedef {(command: string, args: string[]) => Promise<ShellResult>} ShellExecutor
 */

/**
 * Create the default shell executor that wraps node:child_process execFile.
 * @returns {ShellExecutor}
 */
export function createShellExecutor() {
  /**
   * @param {string} command
   * @param {string[]} args
   * @returns {Promise<ShellResult>}
   */
  return async function shellExecutor(command, args) {
    try {
      const { stdout, stderr } = await execFileAsync(command, args);
      return { stdout: stdout.toString(), stderr: stderr.toString(), exitCode: 0 };
    } catch (err) {
      return {
        stdout: err.stdout ? err.stdout.toString() : '',
        stderr: err.stderr ? err.stderr.toString() : err.message,
        exitCode: err.code === 'ENOENT' ? 127 : (err.status ?? 1),
      };
    }
  };
}
