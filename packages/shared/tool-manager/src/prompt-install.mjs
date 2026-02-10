import { select, confirm } from '@inquirer/prompts';

/**
 * @typedef {object} PromptInstallResult
 * @property {boolean} installed
 * @property {boolean} [skipped]
 * @property {string} [version]
 * @property {string} [method]
 * @property {string} [error]
 * @property {string} [instructions]
 */

/**
 * Check whether the current environment is non-interactive.
 * @returns {boolean}
 */
function isNonInteractive() {
  return !!(process.env.CI || !process.stdin.isTTY);
}

/**
 * Build install options from a tool definition and available installers.
 * @param {object} toolDef
 * @param {object} installers - Map of installer name to installer instance
 * @param {string} detectedPm - The detected package manager name
 * @returns {{ name: string, value: string, description: string }[]}
 */
function buildInstallOptions(toolDef, installers, detectedPm) {
  const options = [];
  const pmOrder = [detectedPm, 'manual'].filter((v, i, a) => a.indexOf(v) === i);

  for (const pmName of pmOrder) {
    const installer = installers[pmName];
    if (!installer) continue;

    const cmdInfo = installer.getInstallCommand(toolDef);
    if (pmName === 'manual') {
      options.push({
        name: `Manual: ${cmdInfo.instructions || cmdInfo.url}`,
        value: 'manual',
        description: cmdInfo.url || '',
      });
    } else if (cmdInfo.command) {
      options.push({
        name: `Install via ${pmName}: ${cmdInfo.command}`,
        value: pmName,
        description: cmdInfo.command,
      });
    }
  }

  return options;
}

/**
 * Run the interactive promptInstall flow.
 * In CI/non-TTY environments, returns instructions without prompting.
 *
 * @param {import('./tool-manager.mjs').ToolManager} toolManager
 * @param {string} toolName
 * @returns {Promise<PromptInstallResult>}
 */
export async function promptInstall(toolManager, toolName) {
  const toolDef = await toolManager.getToolDefinition(toolName);
  if (!toolDef) {
    return { installed: false, error: `Unknown tool: ${toolName}` };
  }

  // Non-interactive mode: return instructions without prompting
  if (isNonInteractive()) {
    const cmdInfo = await toolManager.getInstallCommand(toolName);
    return {
      installed: false,
      skipped: true,
      instructions: cmdInfo.command || cmdInfo.instructions || `Install ${toolDef.display_name || toolName} manually.`,
    };
  }

  const pm = await toolManager.detectPackageManager();
  const installers = {
    homebrew: toolManager.getInstaller('homebrew'),
    winget: toolManager.getInstaller('winget'),
    apt: toolManager.getInstaller('apt'),
    manual: toolManager.getInstaller('manual'),
  };

  const options = buildInstallOptions(toolDef, installers, pm.name);

  if (options.length === 0) {
    return { installed: false, error: 'No install methods available' };
  }

  // If only manual option, show it and return
  if (options.length === 1 && options[0].value === 'manual') {
    const manualInfo = installers.manual.getInstallCommand(toolDef);
    return {
      installed: false,
      skipped: true,
      instructions: manualInfo.instructions,
      method: 'manual',
    };
  }

  const choice = await select({
    message: `${toolDef.display_name || toolName} is not installed. How would you like to install it?`,
    choices: options,
  });

  if (choice === 'manual') {
    const manualInfo = installers.manual.getInstallCommand(toolDef);
    return {
      installed: false,
      skipped: true,
      instructions: manualInfo.instructions,
      method: 'manual',
    };
  }

  const installer = installers[choice];
  if (!installer) {
    return { installed: false, error: `No installer found for ${choice}` };
  }

  const shouldProceed = await confirm({
    message: `Run: ${installer.getInstallCommand(toolDef).command}?`,
    default: true,
  });

  if (!shouldProceed) {
    return { installed: false, skipped: true, method: choice };
  }

  const installResult = await installer.install(toolDef);
  if (!installResult.success) {
    return { installed: false, error: installResult.error, method: choice };
  }

  // Clear cache and re-check to verify installation
  toolManager.clearCache();
  const checkResult = await toolManager.check(toolName);

  return {
    installed: checkResult.installed,
    version: checkResult.version || undefined,
    method: choice,
  };
}
