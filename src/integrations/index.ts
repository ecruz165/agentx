import { readFileSync, existsSync, writeFileSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import Handlebars from 'handlebars';
import { loadManifest, createSymlink, flattenRef, isStale, ensureDir, validateSymlinks } from './helpers.js';
import { PROVIDERS } from './providers.js';
import type { ProviderConfig } from './providers.js';

const __dirname = dirname(fileURLToPath(import.meta.url));
const TEMPLATES_DIR = join(__dirname, '..', 'src', 'integrations', 'templates');

// Register a helper to produce {{varName}} literal curly braces in command templates
Handlebars.registerHelper('curly', (value: string) => `{{${value}}}`);

function loadHbsTemplate(provider: string, name: string): Handlebars.TemplateDelegate {
  const templatePath = join(TEMPLATES_DIR, provider, name);
  const source = readFileSync(templatePath, 'utf8');
  return Handlebars.compile(source);
}

export interface GenerateInput {
  toolName: string;
  projectConfig: { active?: Record<string, string[]> };
  installedPath: string;
  projectPath?: string;
}

export interface GenerateOutput {
  created: string[];
  updated: string[];
  symlinked: string[];
  warnings: string[];
}

/**
 * Generate AI tool configuration files for a project.
 */
export async function generate(input: GenerateInput): Promise<GenerateOutput> {
  const { toolName, projectConfig, installedPath, projectPath = '.' } = input;

  const provider = PROVIDERS[toolName];
  if (!provider) {
    throw new Error(`Unknown tool: ${toolName}`);
  }

  const result: GenerateOutput = { created: [], updated: [], symlinked: [], warnings: [] };
  const active = projectConfig.active || {};

  // Load persona data
  let personaData: Record<string, unknown> | null = null;
  const personas = active.personas || [];
  if (personas.length > 0) {
    const loaded = loadManifest(installedPath, personas[0]);
    if (loaded) {
      personaData = loaded.manifest;
    } else {
      result.warnings.push(`Persona not found: ${personas[0]}`);
    }
  }

  // Load skills and workflows if this provider renders them
  const skills: Array<Record<string, unknown> & { ref: string }> = [];
  const workflows: Array<Record<string, unknown> & { ref: string }> = [];

  if (provider.renders.skills) {
    for (const ref of active.skills || []) {
      const loaded = loadManifest(installedPath, ref);
      if (loaded) {
        skills.push({ ...loaded.manifest, ref });
      } else {
        result.warnings.push(`Skill not found: ${ref}`);
      }
    }
  }

  if (provider.renders.workflows) {
    for (const ref of active.workflows || []) {
      const loaded = loadManifest(installedPath, ref);
      if (loaded) {
        workflows.push({ ...loaded.manifest, ref });
      } else {
        result.warnings.push(`Workflow not found: ${ref}`);
      }
    }
  }

  const contextRefs = active.context || [];

  // --- Generate main document ---
  const configDir = join(projectPath, provider.configDir);

  let mainDocDir: string;
  if (provider.mainDoc.atProjectRoot) {
    mainDocDir = projectPath;
  } else {
    mainDocDir = configDir;
    ensureDir(configDir);
  }

  const mainDocTemplate = loadHbsTemplate(toolName, provider.mainDoc.template);
  const mainDocContent = mainDocTemplate({
    persona: personaData,
    skills: skills.length > 0 ? skills : null,
    workflows: workflows.length > 0 ? workflows : null,
    hasContext: contextRefs.length > 0,
  });

  const mainDocPath = join(mainDocDir, provider.mainDoc.filename);
  const existed = existsSync(mainDocPath);
  writeFileSync(mainDocPath, mainDocContent);
  (existed ? result.updated : result.created).push(mainDocPath);

  // --- Generate command files (if supported) ---
  if (provider.commands.supported && provider.commands.template) {
    ensureDir(configDir);
    const commandsDir = join(configDir, 'commands');
    ensureDir(commandsDir);

    const commandTemplate = loadHbsTemplate(toolName, provider.commands.template);

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
  }

  // --- Create context symlinks ---
  ensureDir(configDir);
  const contextDir = join(configDir, provider.context.subdir);
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

export interface StatusInput {
  toolName: string;
  projectPath: string;
}

export interface StatusOutput {
  tool: string;
  status: string;
  files: string[];
  symlinks: { total: number; valid: number };
}

/**
 * Check the status of AI tool configuration in a project.
 */
export async function status(input: StatusInput): Promise<StatusOutput> {
  const { toolName, projectPath } = input;

  const provider = PROVIDERS[toolName];
  if (!provider) {
    throw new Error(`Unknown tool: ${toolName}`);
  }

  const projectYaml = join(projectPath, '.agentx', 'project.yaml');

  let mainDocPath: string;
  if (provider.mainDoc.atProjectRoot) {
    mainDocPath = join(projectPath, provider.mainDoc.filename);
  } else {
    mainDocPath = join(projectPath, provider.configDir, provider.mainDoc.filename);
  }

  const contextDir = join(projectPath, provider.configDir, provider.context.subdir);

  const files: string[] = [];
  if (existsSync(mainDocPath)) {
    files.push(mainDocPath);
  }

  const symlinkInfo = validateSymlinks(contextDir);

  let statusValue = 'up-to-date';
  if (!existsSync(mainDocPath)) {
    statusValue = 'not-generated';
  } else if (isStale(projectYaml, files)) {
    statusValue = 'stale';
  }

  return {
    tool: toolName,
    status: statusValue,
    files,
    symlinks: { total: symlinkInfo.total, valid: symlinkInfo.valid },
  };
}
