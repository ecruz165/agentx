import { readFile, readdir } from 'node:fs/promises';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import { parse as parseYaml } from 'yaml';

const __dirname = dirname(fileURLToPath(import.meta.url));
const REGISTRY_DIR = join(__dirname, '..', 'registry');

/**
 * @typedef {object} ToolCheckDef
 * @property {string} command
 * @property {string} version_regex
 * @property {string} min_version
 */

/**
 * @typedef {object} ToolInstallMethod
 * @property {string} [package] - Package name for simple installs
 * @property {string[]} [commands] - Multi-step install commands
 */

/**
 * @typedef {object} ToolManualInstall
 * @property {string} url
 * @property {string} instructions
 */

/**
 * @typedef {object} ToolInstallDef
 * @property {ToolInstallMethod} [homebrew]
 * @property {ToolInstallMethod} [winget]
 * @property {ToolInstallMethod} [apt]
 * @property {ToolManualInstall} manual
 */

/**
 * @typedef {object} ToolDefinition
 * @property {string} name
 * @property {string} display_name
 * @property {string} description
 * @property {ToolCheckDef} check
 * @property {ToolInstallDef} install
 */

const REQUIRED_FIELDS = ['name', 'check', 'install'];

/**
 * Validate a tool definition has required fields.
 * @param {object} def
 * @param {string} filename
 * @returns {{ valid: boolean, errors: string[] }}
 */
function validateToolDef(def, filename) {
  const errors = [];

  for (const field of REQUIRED_FIELDS) {
    if (!def[field]) {
      errors.push(`${filename}: missing required field '${field}'`);
    }
  }

  if (def.check) {
    if (!def.check.command) {
      errors.push(`${filename}: missing required field 'check.command'`);
    }
    if (!def.check.version_regex) {
      errors.push(`${filename}: missing required field 'check.version_regex'`);
    }
  }

  if (def.install && !def.install.manual) {
    errors.push(`${filename}: missing required field 'install.manual'`);
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Load all YAML tool definitions from the registry directory.
 * @param {string} [registryDir] - Override registry directory (for testing)
 * @returns {Promise<Map<string, ToolDefinition>>}
 */
export async function loadRegistry(registryDir = REGISTRY_DIR) {
  const registry = new Map();
  const files = await readdir(registryDir);
  const yamlFiles = files.filter(f => f.endsWith('.yaml') || f.endsWith('.yml'));

  for (const file of yamlFiles) {
    const content = await readFile(join(registryDir, file), 'utf-8');
    const def = parseYaml(content);

    const { valid, errors } = validateToolDef(def, file);
    if (!valid) {
      throw new Error(`Invalid tool definition:\n${errors.join('\n')}`);
    }

    registry.set(def.name, def);
  }

  return registry;
}
