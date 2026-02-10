import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import Handlebars from 'handlebars';
import { loadManifest, createSymlink, flattenRef, isStale, ensureDir, validateSymlinks } from '@agentx/shared-node';

const __dirname = dirname(fileURLToPath(import.meta.url));
const TEMPLATES_DIR = join(__dirname, '..', 'templates');

// Register a helper to produce {{varName}} literal curly braces in command templates
Handlebars.registerHelper('curly', (value) => `{{${value}}}`);

/**
 * Load and compile a Handlebars template from the templates directory.
 * @param {string} name - Template filename (e.g., 'agents-md.hbs')
 * @returns {HandlebarsTemplateDelegate}
 */
function loadHbsTemplate(name) {
  const templatePath = join(TEMPLATES_DIR, name);
  const source = readFileSync(templatePath, 'utf8');
  return Handlebars.compile(source);
}

/**
 * Generate OpenCode configuration files for a project.
 *
 * Creates:
 * - AGENTS.md — persona + skills + workflows + context reference (project root)
 * - .opencode/commands/<name>.md — one per skill and workflow
 * - .opencode/context/<flattened-name> — symlinks to installed context
 *
 * @param {object} projectConfig - Parsed .agentx/project.yaml content
 * @param {string} installedPath - Path to installed types (e.g., ~/.agentx/installed/)
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
    // Use the first persona (primary)
    const loaded = loadManifest(installedPath, personas[0]);
    if (loaded) {
      personaData = loaded.manifest;
    } else {
      result.warnings.push(`Persona not found: ${personas[0]}`);
    }
  }

  // Load skill manifests
  const skills = [];
  for (const ref of active.skills || []) {
    const loaded = loadManifest(installedPath, ref);
    if (loaded) {
      skills.push({ ...loaded.manifest, ref });
    } else {
      result.warnings.push(`Skill not found: ${ref}`);
    }
  }

  // Load workflow manifests
  const workflows = [];
  for (const ref of active.workflows || []) {
    const loaded = loadManifest(installedPath, ref);
    if (loaded) {
      workflows.push({ ...loaded.manifest, ref });
    } else {
      result.warnings.push(`Workflow not found: ${ref}`);
    }
  }

  // Context references
  const contextRefs = active.context || [];

  // --- Generate AGENTS.md in PROJECT ROOT (not inside .opencode/) ---
  const agentsMdTemplate = loadHbsTemplate('agents-md.hbs');
  const agentsMdContent = agentsMdTemplate({
    persona: personaData,
    skills: skills.length > 0 ? skills : null,
    workflows: workflows.length > 0 ? workflows : null,
    hasContext: contextRefs.length > 0,
  });

  const agentsMdPath = join(projectPath, 'AGENTS.md');
  const existed = existsSync(agentsMdPath);
  writeFileSync(agentsMdPath, agentsMdContent);
  (existed ? result.updated : result.created).push(agentsMdPath);

  // --- Generate command files in .opencode/commands/ ---
  const opencodeDir = join(projectPath, '.opencode');
  ensureDir(opencodeDir);
  const commandsDir = join(opencodeDir, 'commands');
  ensureDir(commandsDir);

  const commandTemplate = loadHbsTemplate('command.hbs');

  for (const skill of skills) {
    const commandPath = join(commandsDir, `${skill.name}.md`);
    const cmdExisted = existsSync(commandPath);
    const content = commandTemplate({
      description: skill.description,
      ref: skill.ref,
      inputs: skill.inputs || null,
    });
    writeFileSync(commandPath, content);
    (cmdExisted ? result.updated : result.created).push(commandPath);
  }

  for (const workflow of workflows) {
    const commandPath = join(commandsDir, `${workflow.name}.md`);
    const cmdExisted = existsSync(commandPath);
    const content = commandTemplate({
      description: workflow.description,
      ref: workflow.ref,
      inputs: workflow.inputs || null,
    });
    writeFileSync(commandPath, content);
    (cmdExisted ? result.updated : result.created).push(commandPath);
  }

  // --- Create context symlinks in .opencode/context/ ---
  const contextDir = join(opencodeDir, 'context');
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
 * Check the status of OpenCode configuration in a project.
 *
 * @param {string} projectPath - Project root directory
 * @returns {{ tool: string, status: string, files: string[], symlinks: { total: number, valid: number } }}
 */
export async function status(projectPath) {
  const projectYaml = join(projectPath, '.agentx', 'project.yaml');
  const agentsMdPath = join(projectPath, 'AGENTS.md');
  const contextDir = join(projectPath, '.opencode', 'context');

  const files = [];
  if (existsSync(agentsMdPath)) {
    files.push(agentsMdPath);
  }

  const symlinkInfo = validateSymlinks(contextDir);

  // Determine staleness
  let statusValue = 'up-to-date';
  if (!existsSync(agentsMdPath)) {
    statusValue = 'not-generated';
  } else if (isStale(projectYaml, files)) {
    statusValue = 'stale';
  }

  return {
    tool: 'opencode',
    status: statusValue,
    files,
    symlinks: { total: symlinkInfo.total, valid: symlinkInfo.valid },
  };
}
