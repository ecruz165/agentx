import { BaseInstaller } from './base.mjs';

/**
 * Installer that uses apt-get on Debian/Ubuntu Linux.
 * Supports both simple package installs and multi-step command sequences.
 */
export class AptInstaller extends BaseInstaller {
  get name() {
    return 'apt';
  }

  /**
   * @param {object} toolDef
   * @returns {import('./base.mjs').InstallCommandInfo}
   */
  getInstallCommand(toolDef) {
    const aptDef = toolDef.install?.apt;
    if (!aptDef) {
      return { method: 'apt', command: null };
    }

    if (aptDef.package) {
      return {
        method: 'apt',
        command: `sudo apt-get install -y ${aptDef.package}`,
      };
    }

    if (aptDef.commands && aptDef.commands.length > 0) {
      return {
        method: 'apt',
        command: aptDef.commands.join(' && '),
      };
    }

    return { method: 'apt', command: null };
  }

  /**
   * @param {object} toolDef
   * @returns {Promise<import('./base.mjs').InstallResult>}
   */
  async install(toolDef) {
    const aptDef = toolDef.install?.apt;
    if (!aptDef) {
      return { success: false, error: 'No apt configuration defined for this tool' };
    }

    if (aptDef.package) {
      const result = await this._shell('sudo', ['apt-get', 'install', '-y', aptDef.package]);
      if (result.exitCode === 0) {
        return { success: true };
      }
      return { success: false, error: result.stderr || `apt-get install failed with exit code ${result.exitCode}` };
    }

    if (aptDef.commands && aptDef.commands.length > 0) {
      for (const cmd of aptDef.commands) {
        const parts = cmd.split(/\s+/);
        const executable = parts[0];
        const args = parts.slice(1);
        const result = await this._shell(executable, args);
        if (result.exitCode !== 0) {
          return { success: false, error: result.stderr || `Command '${cmd}' failed with exit code ${result.exitCode}` };
        }
      }
      return { success: true };
    }

    return { success: false, error: 'No apt package or commands defined for this tool' };
  }
}
