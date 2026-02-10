import { BaseInstaller } from './base.mjs';

/**
 * Installer that uses winget on Windows.
 */
export class WingetInstaller extends BaseInstaller {
  get name() {
    return 'winget';
  }

  /**
   * @param {object} toolDef
   * @returns {import('./base.mjs').InstallCommandInfo}
   */
  getInstallCommand(toolDef) {
    const wingetDef = toolDef.install?.winget;
    if (!wingetDef?.package) {
      return { method: 'winget', command: null };
    }
    return {
      method: 'winget',
      command: `winget install ${wingetDef.package}`,
    };
  }

  /**
   * @param {object} toolDef
   * @returns {Promise<import('./base.mjs').InstallResult>}
   */
  async install(toolDef) {
    const wingetDef = toolDef.install?.winget;
    if (!wingetDef?.package) {
      return { success: false, error: 'No winget package defined for this tool' };
    }

    const result = await this._shell('winget', ['install', wingetDef.package]);
    if (result.exitCode === 0) {
      return { success: true };
    }
    return { success: false, error: result.stderr || `winget install failed with exit code ${result.exitCode}` };
  }
}
