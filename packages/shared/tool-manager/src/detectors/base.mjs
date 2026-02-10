/**
 * @typedef {import('../shell.mjs').ShellExecutor} ShellExecutor
 */

/**
 * @typedef {object} DetectResult
 * @property {boolean} success
 * @property {string} [error]
 */

/**
 * Base class for package manager detectors.
 * Subclasses must override `isAvailable()` and the `name` getter.
 */
export class BaseDetector {
  /**
   * @param {ShellExecutor} shellExecutor
   */
  constructor(shellExecutor) {
    if (!shellExecutor) {
      throw new Error('shellExecutor is required');
    }
    /** @protected */
    this._shell = shellExecutor;
  }

  /**
   * Check whether this package manager is available on the system.
   * @returns {Promise<DetectResult>}
   */
  async isAvailable() {
    throw new Error('isAvailable() must be implemented by subclass');
  }

  /**
   * The package manager name (e.g., 'homebrew', 'apt', 'winget', 'manual').
   * @returns {string}
   */
  get name() {
    throw new Error('name getter must be implemented by subclass');
  }
}
