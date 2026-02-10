import { BaseInstaller } from './base.mjs';

/**
 * Installer that uses Homebrew on macOS.
 */
export class HomebrewInstaller extends BaseInstaller {
  get name() {
    return 'homebrew';
  }

  /**
   * @param {object} toolDef
   * @returns {import('./base.mjs').InstallCommandInfo}
   */
  getInstallCommand(toolDef) {
    const brewDef = toolDef.install?.homebrew;
    if (!brewDef?.package) {
      return { method: 'homebrew', command: null };
    }
    return {
      method: 'homebrew',
      command: `brew install ${brewDef.package}`,
    };
  }

  /**
   * @param {object} toolDef
   * @returns {Promise<import('./base.mjs').InstallResult>}
   */
  async install(toolDef) {
    const brewDef = toolDef.install?.homebrew;
    if (!brewDef?.package) {
      return { success: false, error: 'No Homebrew package defined for this tool' };
    }

    const result = await this._shell('brew', ['install', brewDef.package]);
    if (result.exitCode === 0) {
      return { success: true };
    }
    return { success: false, error: result.stderr || `brew install failed with exit code ${result.exitCode}` };
  }
}
