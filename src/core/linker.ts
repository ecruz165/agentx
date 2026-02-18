import { join, dirname } from 'node:path';
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'node:fs';
import yaml from 'js-yaml';
import type { ToolName, GenerateResult, StatusResult } from '../types/integrations.js';
import { ALL_TOOLS } from '../types/integrations.js';

// ── Project config ──────────────────────────────────────────────────

export interface ActiveConfig {
  personas?: string[];
  context?: string[];
  skills?: string[];
  workflows?: string[];
  prompts?: string[];
}

export interface ProjectConfig {
  tools: string[];
  active: ActiveConfig;
}

const PROJECT_DIR = '.agentx';
const PROJECT_FILE = 'project.yaml';

export function projectConfigPath(projectPath: string): string {
  return join(projectPath, PROJECT_DIR, PROJECT_FILE);
}

export function loadProject(projectPath: string): ProjectConfig {
  const path = projectConfigPath(projectPath);
  const raw = readFileSync(path, 'utf-8');
  const data = yaml.load(raw) as ProjectConfig;
  return {
    tools: data.tools ?? [],
    active: {
      personas: data.active?.personas ?? [],
      context: data.active?.context ?? [],
      skills: data.active?.skills ?? [],
      workflows: data.active?.workflows ?? [],
      prompts: data.active?.prompts ?? [],
    },
  };
}

export function saveProject(
  projectPath: string,
  config: ProjectConfig,
): void {
  const path = projectConfigPath(projectPath);
  mkdirSync(dirname(path), { recursive: true });
  writeFileSync(path, yaml.dump(config, { lineWidth: -1 }), 'utf-8');
}

export function initProject(projectPath: string, tools: string[]): void {
  const agentxDir = join(projectPath, PROJECT_DIR);
  mkdirSync(agentxDir, { recursive: true });
  mkdirSync(join(agentxDir, 'overrides'), { recursive: true });

  const config: ProjectConfig = {
    tools,
    active: {
      personas: [],
      context: [],
      skills: [],
      workflows: [],
      prompts: [],
    },
  };
  saveProject(projectPath, config);
}

// ── Type management ─────────────────────────────────────────────────

function typeSection(typeRef: string): keyof ActiveConfig {
  const prefix = typeRef.split('/')[0];
  const map: Record<string, keyof ActiveConfig> = {
    personas: 'personas',
    context: 'context',
    skills: 'skills',
    workflows: 'workflows',
    prompts: 'prompts',
  };
  const section = map[prefix];
  if (!section) {
    throw new Error(`Unknown type prefix: "${prefix}". Expected: personas, context, skills, workflows, or prompts.`);
  }
  return section;
}

export async function addType(projectPath: string, typeRef: string): Promise<void> {
  const config = loadProject(projectPath);
  const section = typeSection(typeRef);
  const list = config.active[section] ?? [];
  if (list.includes(typeRef)) {
    throw new Error(`Type "${typeRef}" is already linked.`);
  }
  list.push(typeRef);
  config.active[section] = list;
  saveProject(projectPath, config);
  await sync(projectPath);
}

export async function removeType(projectPath: string, typeRef: string): Promise<void> {
  const config = loadProject(projectPath);
  const section = typeSection(typeRef);
  const list = config.active[section] ?? [];
  if (!list.includes(typeRef)) {
    throw new Error(`Type "${typeRef}" is not linked.`);
  }
  config.active[section] = list.filter((t) => t !== typeRef);
  saveProject(projectPath, config);
  await sync(projectPath);
}

// ── Sync & Status ───────────────────────────────────────────────────

export async function sync(projectPath: string): Promise<GenerateResult[]> {
  const config = loadProject(projectPath);
  const { getInstalledRoot } = await import('./userdata.js');
  const installedPath = getInstalledRoot();

  const { generate } = await import('../integrations/index.js');
  const results: GenerateResult[] = [];

  for (const toolName of config.tools) {
    try {
      const result = await generate({
        toolName,
        projectConfig: config,
        installedPath,
        projectPath,
      });
      results.push(result as GenerateResult);
    } catch (err) {
      results.push({
        tool: toolName as ToolName,
        created: [],
        updated: [],
        symlinked: [],
        warnings: [String(err)],
      });
    }
  }
  return results;
}

export async function status(projectPath: string): Promise<StatusResult[]> {
  const config = loadProject(projectPath);

  const { status: getStatus } = await import('../integrations/index.js');
  const results: StatusResult[] = [];

  for (const toolName of config.tools) {
    try {
      const result = await getStatus({ toolName, projectPath });
      results.push(result as StatusResult);
    } catch (err) {
      results.push({
        tool: toolName,
        status: 'error',
        files: [],
        symlinks: { total: 0, valid: 0 },
      });
    }
  }
  return results;
}
