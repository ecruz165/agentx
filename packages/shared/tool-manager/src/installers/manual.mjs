import { BaseInstaller } from './base.mjs';

/**
 * Manual fallback installer. Never actually installs anything.
 * Returns human-readable instructions with a URL.
 */
export class ManualInstaller extends BaseInstaller {
  get name() {
    return 'manual';
  }

  /**
   * @param {object} toolDef
   * @returns {import('./base.mjs').InstallCommandInfo}
   */
  getInstallCommand(toolDef) {
    const manualDef = toolDef.install?.manual;
    return {
      method: 'manual',
      url: manualDef?.url || '',
      instructions: manualDef?.instructions || `Install ${toolDef.display_name || toolDef.name} manually.`,
    };
  }

  /**
   * Manual installer never actually installs. Returns instructions instead.
   * @param {object} toolDef
   * @returns {Promise<import('./base.mjs').InstallResult>}
   */
  async install(toolDef) {
    const manualDef = toolDef.install?.manual;
    return {
      success: false,
      instructions: manualDef?.instructions || `Install ${toolDef.display_name || toolDef.name} manually.`,
    };
  }
}
