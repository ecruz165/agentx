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
 * Generate GitHub Copilot configuration files for a project.
 *
 * Creates:
 * - .github/copilot-instructions.md — persona description inline + conventions + context ref
 * - .github/copilot-context/<flattened-name> — symlinks to installed context
 *
 * Only touches copilot-specific files; never modifies other .github/ content.
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

  // --- Generate copilot-instructions.md ---
  const githubDir = join(projectPath, '.github');
  ensureDir(githubDir);

  const template = loadHbsTemplate('copilot-instructions.hbs');
  const content = template({
    persona: personaData,
    hasContext: contextRefs.length > 0,
  });

  const instructionsPath = join(githubDir, 'copilot-instructions.md');
  const existed = existsSync(instructionsPath);
  writeFileSync(instructionsPath, content);
  (existed ? result.updated : result.created).push(instructionsPath);

  // --- Create context symlinks ---
  const contextDir = join(githubDir, 'copilot-context');
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
 * Check the status of Copilot configuration in a project.
 *
 * @param {string} projectPath - Project root directory
 * @returns {{ tool: string, status: string, files: string[], symlinks: { total: number, valid: number } }}
 */
export async function status(projectPath) {
  const projectYaml = join(projectPath, '.agentx', 'project.yaml');
  const instructionsPath = join(projectPath, '.github', 'copilot-instructions.md');
  const contextDir = join(projectPath, '.github', 'copilot-context');

  const files = [];
  if (existsSync(instructionsPath)) {
    files.push(instructionsPath);
  }

  const symlinkInfo = validateSymlinks(contextDir);

  let statusValue = 'up-to-date';
  if (!existsSync(instructionsPath)) {
    statusValue = 'not-generated';
  } else if (isStale(projectYaml, files)) {
    statusValue = 'stale';
  }

  return {
    tool: 'copilot',
    status: statusValue,
    files,
    symlinks: { total: symlinkInfo.total, valid: symlinkInfo.valid },
  };
}
