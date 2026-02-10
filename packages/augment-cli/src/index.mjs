import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import Handlebars from 'handlebars';
import { loadManifest, createSymlink, flattenRef, isStale, ensureDir, validateSymlinks } from '@agentx/shared-node';

const __dirname = dirname(fileURLToPath(import.meta.url));
const TEMPLATES_DIR = join(__dirname, '..', 'templates');

/**
 * Load and compile a Handlebars template.
 * @param {string} name
 * @returns {HandlebarsTemplateDelegate}
 */
function loadHbsTemplate(name) {
  const templatePath = join(TEMPLATES_DIR, name);
  const source = readFileSync(templatePath, 'utf8');
  return Handlebars.compile(source);
}

/**
 * Generate Augment Code configuration files for a project.
 *
 * Creates:
 * - .augment/augment-guidelines.md — persona description + conventions + context ref
 * - .augment/context/<flattened-name> — symlinks to installed context
 *
 * @param {object} projectConfig - Parsed .agentx/project.yaml content
 * @param {string} installedPath - Path to installed types
 * @param {string} [projectPath='.'] - Project root directory
 * @returns {{ created: string[], updated: string[], symlinked: string[], warnings: string[] }}
 */
export async function generate(projectConfig, installedPath, projectPath = '.') {
  const result = { created: [], updated: [], symlinked: [], warnings: [] };
  const active = projectConfig.active || {};

  // Load persona data
  let personaData = null;
  const personas = active.personas || [];
  if (personas.length > 0) {
    const loaded = loadManifest(installedPath, personas[0]);
    if (loaded) {
      personaData = loaded.manifest;
    } else {
      result.warnings.push(`Persona not found: ${personas[0]}`);
    }
  }

  // Context references
  const contextRefs = active.context || [];

  // --- Generate augment-guidelines.md ---
  const augmentDir = join(projectPath, '.augment');
  ensureDir(augmentDir);

  const template = loadHbsTemplate('augment-guidelines.hbs');
  const content = template({
    persona: personaData,
    hasContext: contextRefs.length > 0,
  });

  const guidelinesPath = join(augmentDir, 'augment-guidelines.md');
  const existed = existsSync(guidelinesPath);
  writeFileSync(guidelinesPath, content);
  (existed ? result.updated : result.created).push(guidelinesPath);

  // --- Create context symlinks ---
  const contextDir = join(augmentDir, 'context');
  ensureDir(contextDir);

  for (const ref of contextRefs) {
    const flatName = flattenRef(ref);
    const linkPath = join(contextDir, flatName);
    const target = join(installedPath, ref);

    if (!existsSync(target)) {
      result.warnings.push(`Context not found: ${ref}`);
      continue;
    }

    createSymlink(target, linkPath);
    result.symlinked.push(linkPath);
  }

  return result;
}

/**
 * Check the status of Augment Code configuration in a project.
 *
 * @param {string} projectPath - Project root directory
 * @returns {{ tool: string, status: string, files: string[], symlinks: { total: number, valid: number } }}
 */
export async function status(projectPath) {
  const projectYaml = join(projectPath, '.agentx', 'project.yaml');
  const guidelinesPath = join(projectPath, '.augment', 'augment-guidelines.md');
  const contextDir = join(projectPath, '.augment', 'context');

  const files = [];
  if (existsSync(guidelinesPath)) {
    files.push(guidelinesPath);
  }

  const symlinkInfo = validateSymlinks(contextDir);

  let statusValue = 'up-to-date';
  if (!existsSync(guidelinesPath)) {
    statusValue = 'not-generated';
  } else if (isStale(projectYaml, files)) {
    statusValue = 'stale';
  }

  return {
    tool: 'augment',
    status: statusValue,
    files,
    symlinks: { total: symlinkInfo.total, valid: symlinkInfo.valid },
  };
}
