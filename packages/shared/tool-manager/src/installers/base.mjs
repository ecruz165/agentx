/**
 * @typedef {import('../shell.mjs').ShellExecutor} ShellExecutor
 */

/**
 * @typedef {object} InstallResult
 * @property {boolean} success
 * @property {string} [error]
 * @property {string} [instructions]
 */

/**
 * @typedef {object} InstallCommandInfo
 * @property {string} method
 * @property {string} [command]
 * @property {string} [url]
 * @property {string} [instructions]
 */

/**
 * Base class for package manager installers.
 * Subclasses must override `install()`, `getInstallCommand()`, and the `name` getter.
 */
export class BaseInstaller {
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
   * Install a tool using this package manager.
   * @param {object} toolDef - The tool definition from the YAML registry
   * @returns {Promise<InstallResult>}
   */
  async install(toolDef) {
    throw new Error('install() must be implemented by subclass');
  }

  /**
   * Get the install command without executing it.
   * @param {object} toolDef - The tool definition from the YAML registry
   * @returns {InstallCommandInfo}
   */
  getInstallCommand(toolDef) {
    throw new Error('getInstallCommand() must be implemented by subclass');
  }

  /**
   * The package manager name (e.g., 'homebrew', 'apt', 'winget', 'manual').
   * @returns {string}
   */
  get name() {
    throw new Error('name getter must be implemented by subclass');
  }
}
