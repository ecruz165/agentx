import { join } from 'node:path';
import { readFileSync, existsSync } from 'node:fs';
import yaml from 'js-yaml';
import type { PromptManifest, PersonaManifest, ContextManifest } from '../types/manifest.js';

export interface PersonaSection {
  name: string;
  expertise: string[];
  tone: string;
  conventions: string[];
}

export interface ContextSection {
  name: string;
  content: string;
}

export interface SkillRef {
  name: string;
  description: string;
}

export interface WorkflowRef {
  name: string;
  description: string;
}

export interface ComposedPrompt {
  promptName: string;
  persona: PersonaSection | null;
  context: ContextSection[];
  skills: SkillRef[];
  workflows: WorkflowRef[];
  warnings: string[];
}

function findManifest(dir: string): string | null {
  for (const name of ['manifest.yaml', 'manifest.json']) {
    const path = join(dir, name);
    if (existsSync(path)) return path;
  }
  return null;
}

function formatContextName(name: string): string {
  return name
    .split('-')
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(' ');
}

function loadPersona(
  personaPath: string,
  installedRoot: string,
): { section: PersonaSection | null; warnings: string[] } {
  const dir = join(installedRoot, personaPath);
  const manifestPath = findManifest(dir);
  if (!manifestPath) {
    return { section: null, warnings: [`Persona not found: ${personaPath}`] };
  }

  try {
    const raw = readFileSync(manifestPath, 'utf-8');
    const data = yaml.load(raw) as PersonaManifest;
    return {
      section: {
        name: data.name,
        expertise: data.expertise ?? [],
        tone: data.tone ?? '',
        conventions: data.conventions ?? [],
      },
      warnings: [],
    };
  } catch {
    return { section: null, warnings: [`Failed to parse persona: ${personaPath}`] };
  }
}

function loadContext(
  ctxPath: string,
  installedRoot: string,
): { sections: ContextSection[]; warnings: string[] } {
  const dir = join(installedRoot, ctxPath);
  const manifestPath = findManifest(dir);
  if (!manifestPath) {
    return { sections: [], warnings: [`Context not found: ${ctxPath}`] };
  }

  try {
    const raw = readFileSync(manifestPath, 'utf-8');
    const data = yaml.load(raw) as ContextManifest;
    const sections: ContextSection[] = [];

    for (const source of data.sources) {
      const filePath = join(dir, source);
      try {
        const content = readFileSync(filePath, 'utf-8');
        sections.push({ name: formatContextName(data.name), content });
      } catch {
        // Skip missing source files
      }
    }
    return { sections, warnings: [] };
  } catch {
    return { sections: [], warnings: [`Failed to parse context: ${ctxPath}`] };
  }
}

function loadSkillRef(
  skillPath: string,
  installedRoot: string,
): { ref: SkillRef | null; warnings: string[] } {
  const dir = join(installedRoot, skillPath);
  const manifestPath = findManifest(dir);
  if (!manifestPath) {
    return { ref: null, warnings: [`Skill not found: ${skillPath}`] };
  }
  try {
    const raw = readFileSync(manifestPath, 'utf-8');
    const data = yaml.load(raw) as { name: string; description: string };
    return { ref: { name: data.name, description: data.description }, warnings: [] };
  } catch {
    return { ref: null, warnings: [`Failed to parse skill: ${skillPath}`] };
  }
}

function loadWorkflowRef(
  wfPath: string,
  installedRoot: string,
): { ref: WorkflowRef | null; warnings: string[] } {
  const dir = join(installedRoot, wfPath);
  const manifestPath = findManifest(dir);
  if (!manifestPath) {
    return { ref: null, warnings: [`Workflow not found: ${wfPath}`] };
  }
  try {
    const raw = readFileSync(manifestPath, 'utf-8');
    const data = yaml.load(raw) as { name: string; description: string };
    return { ref: { name: data.name, description: data.description }, warnings: [] };
  } catch {
    return { ref: null, warnings: [`Failed to parse workflow: ${wfPath}`] };
  }
}

export function compose(
  promptPath: string,
  installedRoot: string,
): ComposedPrompt {
  const dir = join(installedRoot, promptPath);
  const manifestPath = findManifest(dir);
  if (!manifestPath) {
    throw new Error(`Prompt not found: ${promptPath}`);
  }

  const raw = readFileSync(manifestPath, 'utf-8');
  const data = yaml.load(raw) as PromptManifest;
  const warnings: string[] = [];

  let persona: PersonaSection | null = null;
  if (data.persona) {
    const res = loadPersona(data.persona, installedRoot);
    persona = res.section;
    warnings.push(...res.warnings);
  }

  const context: ContextSection[] = [];
  if (data.context) {
    for (const ctxPath of data.context) {
      const res = loadContext(ctxPath, installedRoot);
      context.push(...res.sections);
      warnings.push(...res.warnings);
    }
  }

  const skills: SkillRef[] = [];
  if (data.skills) {
    for (const skillPath of data.skills) {
      const res = loadSkillRef(skillPath, installedRoot);
      if (res.ref) skills.push(res.ref);
      warnings.push(...res.warnings);
    }
  }

  const workflows: WorkflowRef[] = [];
  if (data.workflows) {
    for (const wfPath of data.workflows) {
      const res = loadWorkflowRef(wfPath, installedRoot);
      if (res.ref) workflows.push(res.ref);
      warnings.push(...res.warnings);
    }
  }

  return {
    promptName: data.name,
    persona,
    context,
    skills,
    workflows,
    warnings,
  };
}

export function render(cp: ComposedPrompt): string {
  const parts: string[] = [];

  if (cp.persona) {
    parts.push(`# Persona: ${cp.persona.name}`);
    if (cp.persona.expertise.length) {
      parts.push(`\n**Expertise:** ${cp.persona.expertise.join(', ')}`);
    }
    if (cp.persona.tone) {
      parts.push(`**Tone:** ${cp.persona.tone}`);
    }
    if (cp.persona.conventions.length) {
      parts.push(`\n**Conventions:**`);
      for (const c of cp.persona.conventions) {
        parts.push(`- ${c}`);
      }
    }
    parts.push('');
  }

  for (const ctx of cp.context) {
    parts.push(`## Context: ${ctx.name}\n`);
    parts.push(ctx.content);
    parts.push('');
  }

  if (cp.skills.length) {
    parts.push('## Available Skills\n');
    for (const s of cp.skills) {
      parts.push(`- **${s.name}**: ${s.description}`);
    }
    parts.push('');
  }

  if (cp.workflows.length) {
    parts.push('## Available Workflows\n');
    for (const w of cp.workflows) {
      parts.push(`- **${w.name}**: ${w.description}`);
    }
    parts.push('');
  }

  return parts.join('\n');
}
