import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import { homedir } from 'node:os';
import { config } from 'dotenv';
import { parse as parseYaml } from 'yaml';

/**
 * Get the userdata root path.
 * Respects AGENTX_USERDATA env var, defaults to ~/.agentx/userdata.
 * @returns {string}
 */
export function getUserdataRoot() {
  return process.env.AGENTX_USERDATA || join(homedir(), '.agentx', 'userdata');
}

/**
 * Get the registry path for a skill.
 * @param {string} topic - e.g., 'cloud'
 * @param {string} [vendor] - e.g., 'aws' (omitted if falsy)
 * @param {string} name - e.g., 'ssm-lookup'
 * @returns {string}
 */
export function getSkillRegistryPath(topic, vendor, name) {
  const root = getUserdataRoot();
  if (vendor) {
    return join(root, 'skills', topic, vendor, name);
  }
  return join(root, 'skills', topic, name);
}

/**
 * Load env chain: default.env -> vendor.env -> skill tokens.env
 * Each successive file overrides earlier values.
 * @param {string} skillPath - Full skill registry path
 * @param {string} [vendor] - Vendor name for vendor env file
 */
export function loadEnvChain(skillPath, vendor) {
  const root = getUserdataRoot();

  // 1. Shared global env
  const defaultEnv = join(root, 'env', 'default.env');
  if (existsSync(defaultEnv)) {
    config({ path: defaultEnv });
  }

  // 2. Shared vendor env
  if (vendor) {
    const vendorEnv = join(root, 'env', `${vendor}.env`);
    if (existsSync(vendorEnv)) {
      config({ path: vendorEnv, override: true });
    }
  }

  // 3. Skill-specific tokens (highest priority)
  const tokensEnv = join(skillPath, 'tokens.env');
  if (existsSync(tokensEnv)) {
    config({ path: tokensEnv, override: true });
  }
}

/**
 * Read a state file from the skill's state/ directory.
 * @param {string} registryPath - Skill registry root
 * @param {string} filename
 * @returns {any | null}
 */
export function readState(registryPath, filename) {
  const filepath = join(registryPath, 'state', filename);
  if (!existsSync(filepath)) return null;
  return JSON.parse(readFileSync(filepath, 'utf8'));
}

/**
 * Write a state file to the skill's state/ directory.
 * @param {string} registryPath
 * @param {string} filename
 * @param {any} data
 */
export function writeState(registryPath, filename, data) {
  const dir = join(registryPath, 'state');
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, filename), JSON.stringify(data, null, 2));
}

/**
 * Save output to output/latest.json and a timestamped copy.
 * @param {string} registryPath
 * @param {any} data
 */
export function saveOutput(registryPath, data) {
  const dir = join(registryPath, 'output');
  mkdirSync(dir, { recursive: true });
  const timestamp = new Date().toISOString().replace(/[:.]/g, '-');
  const payload = JSON.stringify(data, null, 2);
  writeFileSync(join(dir, 'latest.json'), payload);
  writeFileSync(join(dir, `${timestamp}.json`), payload);
}

/**
 * Load a template by name.
 * @param {string} registryPath
 * @param {string} name
 * @returns {string | null}
 */
export function loadTemplate(registryPath, name) {
  const filepath = join(registryPath, 'templates', name);
  if (!existsSync(filepath)) return null;
  return readFileSync(filepath, 'utf8');
}

/**
 * Save a template.
 * @param {string} registryPath
 * @param {string} name
 * @param {string} content
 */
export function saveTemplate(registryPath, name, content) {
  const dir = join(registryPath, 'templates');
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, name), content);
}

/**
 * List all saved templates.
 * @param {string} registryPath
 * @returns {string[]}
 */
export function listTemplates(registryPath) {
  const dir = join(registryPath, 'templates');
  if (!existsSync(dir)) return [];
  return readdirSync(dir);
}

/**
 * Read the skill's config.yaml.
 * @param {string} registryPath
 * @returns {object}
 */
export function readConfig(registryPath) {
  const filepath = join(registryPath, 'config.yaml');
  if (!existsSync(filepath)) return {};
  return parseYaml(readFileSync(filepath, 'utf8'));
}
