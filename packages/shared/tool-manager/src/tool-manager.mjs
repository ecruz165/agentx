import { gte, coerce } from 'semver';
import { createShellExecutor } from './shell.mjs';
import { loadRegistry } from './registry-loader.mjs';
import { promptInstall } from './prompt-install.mjs';
import { HomebrewDetector } from './detectors/homebrew.mjs';
import { WingetDetector } from './detectors/winget.mjs';
import { AptDetector } from './detectors/apt.mjs';
import { ManualDetector } from './detectors/manual.mjs';
import { HomebrewInstaller } from './installers/homebrew.mjs';
import { WingetInstaller } from './installers/winget.mjs';
import { AptInstaller } from './installers/apt.mjs';
import { ManualInstaller } from './installers/manual.mjs';

/**
 * @typedef {object} CheckResult
 * @property {boolean} installed
 * @property {string} [version]
 * @property {boolean} [meetsMinimum]
 */

/**
 * @typedef {object} CheckAllResult
 * @property {boolean} satisfied
 * @property {string[]} missing
 * @property {string[]} outdated
 * @property {Map<string, CheckResult>} results
 */

/**
 * @typedef {object} PackageManagerInfo
 * @property {string} name
 * @property {boolean} available
 */

/**
 * Main ToolManager class for CLI dependency detection and installation guidance.
 * Accepts a shell executor via constructor for testability.
 */
export class ToolManager {
  /**
   * @param {object} [options]
   * @param {import('./shell.mjs').ShellExecutor} [options.shellExecutor] - Custom shell executor for testing
   * @param {string} [options.registryDir] - Custom registry directory for testing
   */
  constructor(options = {}) {
    this._shell = options.shellExecutor || createShellExecutor();
    this._registryDir = options.registryDir || undefined;
    this._cache = new Map();
    this._registry = null;
    this._detectors = this._createDetectors();
    this._installers = this._createInstallers();
  }

  /**
   * @private
   * @returns {object}
   */
  _createDetectors() {
    return {
      darwin: [new HomebrewDetector(this._shell), new ManualDetector(this._shell)],
      win32: [new WingetDetector(this._shell), new ManualDetector(this._shell)],
      linux: [new AptDetector(this._shell), new ManualDetector(this._shell)],
    };
  }

  /**
   * @private
   * @returns {object}
   */
  _createInstallers() {
    return {
      homebrew: new HomebrewInstaller(this._shell),
      winget: new WingetInstaller(this._shell),
      apt: new AptInstaller(this._shell),
      manual: new ManualInstaller(this._shell),
    };
  }

  /**
   * Lazy-load the tool registry on first use.
   * @private
   * @returns {Promise<Map<string, import('./registry-loader.mjs').ToolDefinition>>}
   */
  async _ensureRegistry() {
    if (!this._registry) {
      this._registry = await loadRegistry(this._registryDir);
    }
    return this._registry;
  }

  /**
   * Parse a check command string into executable and arguments.
   * @private
   * @param {string} commandStr - e.g., "git --version"
   * @returns {{ executable: string, args: string[] }}
   */
  _parseCommand(commandStr) {
    const parts = commandStr.split(/\s+/);
    return { executable: parts[0], args: parts.slice(1) };
  }

  /**
   * Check if a tool is installed and meets version requirements.
   * @param {string} toolName - Tool name as defined in registry (e.g., 'git', 'aws')
   * @param {{ minVersion?: string }} [options]
   * @returns {Promise<CheckResult>}
   */
  async check(toolName, options = {}) {
    const cacheKey = `${toolName}:${options.minVersion || ''}`;
    if (this._cache.has(cacheKey)) {
      return this._cache.get(cacheKey);
    }

    const registry = await this._ensureRegistry();
    const toolDef = registry.get(toolName);
    if (!toolDef) {
      const result = { installed: false, error: `Unknown tool: ${toolName}` };
      this._cache.set(cacheKey, result);
      return result;
    }

    const { executable, args } = this._parseCommand(toolDef.check.command);
    const shellResult = await this._shell(executable, args);

    if (shellResult.exitCode !== 0 && shellResult.exitCode !== 127) {
      // Some tools output version to stderr (e.g., aws --version)
      // Try to match version from combined output
      const combined = shellResult.stdout + shellResult.stderr;
      const regex = new RegExp(toolDef.check.version_regex);
      const match = combined.match(regex);
      if (!match) {
        const result = { installed: false };
        this._cache.set(cacheKey, result);
        return result;
      }
    }

    if (shellResult.exitCode === 127) {
      const result = { installed: false };
      this._cache.set(cacheKey, result);
      return result;
    }

    // Try to extract version from stdout, then stderr (aws outputs to stderr)
    const combined = shellResult.stdout + shellResult.stderr;
    const regex = new RegExp(toolDef.check.version_regex);
    const match = combined.match(regex);

    if (!match || !match[1]) {
      const result = { installed: true, version: null, meetsMinimum: null };
      this._cache.set(cacheKey, result);
      return result;
    }

    const version = match[1];
    const minVersion = options.minVersion || toolDef.check.min_version;
    let meetsMinimum = true;

    if (minVersion) {
      const coerced = coerce(version);
      const coercedMin = coerce(minVersion);
      if (coerced && coercedMin) {
        meetsMinimum = gte(coerced, coercedMin);
      }
    }

    const result = { installed: true, version, meetsMinimum };
    this._cache.set(cacheKey, result);
    return result;
  }

  /**
   * Check all dependencies at once.
   * @param {{ name: string, minVersion?: string }[]} deps
   * @returns {Promise<CheckAllResult>}
   */
  async checkAll(deps) {
    const results = new Map();
    const missing = [];
    const outdated = [];

    const checks = await Promise.all(
      deps.map(async (dep) => {
        const result = await this.check(dep.name, { minVersion: dep.minVersion });
        return { dep, result };
      })
    );

    for (const { dep, result } of checks) {
      results.set(dep.name, result);
      if (!result.installed) {
        missing.push(dep.name);
      } else if (result.meetsMinimum === false) {
        outdated.push(dep.name);
      }
    }

    return {
      satisfied: missing.length === 0 && outdated.length === 0,
      missing,
      outdated,
      results,
    };
  }

  /**
   * Detect the available package manager for the current platform.
   * @returns {Promise<PackageManagerInfo>}
   */
  async detectPackageManager() {
    const platform = process.platform;
    const detectors = this._detectors[platform] || this._detectors.linux || [new ManualDetector(this._shell)];

    for (const detector of detectors) {
      const result = await detector.isAvailable();
      if (result.success) {
        return { name: detector.name, available: true };
      }
    }

    return { name: 'manual', available: true };
  }

  /**
   * Get the install command for a tool without executing it.
   * @param {string} toolName
   * @returns {Promise<import('./installers/base.mjs').InstallCommandInfo>}
   */
  async getInstallCommand(toolName) {
    const registry = await this._ensureRegistry();
    const toolDef = registry.get(toolName);
    if (!toolDef) {
      return { method: 'manual', instructions: `Unknown tool: ${toolName}` };
    }

    const pm = await this.detectPackageManager();
    const installer = this._installers[pm.name];
    if (!installer) {
      return this._installers.manual.getInstallCommand(toolDef);
    }

    const cmdInfo = installer.getInstallCommand(toolDef);
    if (!cmdInfo.command && pm.name !== 'manual') {
      return this._installers.manual.getInstallCommand(toolDef);
    }

    return cmdInfo;
  }

  /**
   * Get the tool definition from the registry.
   * @param {string} toolName
   * @returns {Promise<import('./registry-loader.mjs').ToolDefinition | undefined>}
   */
  async getToolDefinition(toolName) {
    const registry = await this._ensureRegistry();
    return registry.get(toolName);
  }

  /**
   * Get the installer for a given package manager name.
   * @param {string} pmName
   * @returns {import('./installers/base.mjs').BaseInstaller | undefined}
   */
  getInstaller(pmName) {
    return this._installers[pmName];
  }

  /**
   * Interactive install prompt. In CI/non-TTY, returns instructions without prompting.
   * @param {string} toolName
   * @returns {Promise<import('./prompt-install.mjs').PromptInstallResult>}
   */
  async promptInstall(toolName) {
    return promptInstall(this, toolName);
  }

  /**
   * Clear the in-memory cache.
   */
  clearCache() {
    this._cache.clear();
  }
}
