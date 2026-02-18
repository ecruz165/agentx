import { join, relative, sep } from 'node:path';
import {
  existsSync,
  readdirSync,
  readFileSync,
  writeFileSync,
  mkdirSync,
  rmSync,
  statSync,
  copyFileSync,
} from 'node:fs';
import { execFileSync } from 'node:child_process';
import yaml from 'js-yaml';
import type {
  Source,
  ResolvedType,
  DependencyNode,
  InstallPlan,
  CLIDepStatus,
  DiscoveredType,
  InstallResult,
} from '../types/registry.js';
import type { ManifestType } from '../config/schema.js';
import type {
  BaseManifest,
  SkillManifest,
  WorkflowManifest,
  PersonaManifest,
  PromptManifest,
} from '../types/manifest.js';
import { getHomeRoot } from './userdata.js';
import { copyDir as copyDirUtil, ensureDir } from '../utils/fs.js';

// ── Constants ───────────────────────────────────────────────────────

const KNOWN_CATEGORIES = [
  'context',
  'personas',
  'skills',
  'workflows',
  'prompts',
  'templates',
];

const MANIFEST_FILES = new Set([
  'manifest.yaml',
  'manifest.json',
  'context.yaml',
  'persona.yaml',
  'skill.yaml',
  'workflow.yaml',
  'prompt.yaml',
  'template.yaml',
]);

const EXCLUDED_NAMES = new Set(['node_modules', '.git', '.DS_Store']);

const PLURAL_TO_SINGULAR: Record<string, ManifestType> = {
  context: 'context',
  personas: 'persona',
  skills: 'skill',
  workflows: 'workflow',
  prompts: 'prompt',
  templates: 'template',
};

// ── Resolution ──────────────────────────────────────────────────────

export function categoryFromPath(typePath: string): ManifestType {
  const first = typePath.split('/')[0];
  return PLURAL_TO_SINGULAR[first] ?? (first as ManifestType);
}

export function nameFromPath(typePath: string): string {
  const idx = typePath.indexOf('/');
  return idx === -1 ? typePath : typePath.slice(idx + 1);
}

function pluralize(category: string): string {
  const map: Record<string, string> = {
    context: 'context',
    persona: 'personas',
    skill: 'skills',
    workflow: 'workflows',
    prompt: 'prompts',
    template: 'templates',
  };
  return map[category] ?? category;
}

function findManifest(dir: string, typePath: string): string | null {
  const category = categoryFromPath(typePath);

  const candidates = [
    'manifest.yaml',
    'manifest.json',
    `${category}.yaml`,
  ];

  for (const name of candidates) {
    const path = join(dir, name);
    if (existsSync(path)) return path;
  }
  return null;
}

export function resolveType(
  typePath: string,
  sources: Source[],
): ResolvedType | null {
  const category = categoryFromPath(typePath);

  for (const source of sources) {
    const dir = join(source.basePath, typePath);
    const manifestPath = findManifest(dir, typePath);
    if (manifestPath) {
      return {
        typePath,
        manifestPath,
        sourceDir: dir,
        sourceName: source.name,
        category,
      };
    }
  }
  return null;
}

// ── Discovery ───────────────────────────────────────────────────────

function walkSource(source: Source): ResolvedType[] {
  const results: ResolvedType[] = [];
  const seen = new Set<string>();

  for (const catDir of KNOWN_CATEGORIES) {
    const catPath = join(source.basePath, catDir);
    if (!existsSync(catPath)) continue;

    walkDir(catPath, source.basePath, source.name, seen, results);
  }
  return results;
}

function walkDir(
  dir: string,
  basePath: string,
  sourceName: string,
  seen: Set<string>,
  results: ResolvedType[],
): void {
  let entries;
  try {
    entries = readdirSync(dir, { withFileTypes: true });
  } catch {
    return;
  }

  // Check if this directory has a manifest
  for (const entry of entries) {
    if (entry.isFile() && MANIFEST_FILES.has(entry.name)) {
      const rel = relative(basePath, dir).split(sep).join('/');
      if (!seen.has(rel)) {
        seen.add(rel);
        const category = categoryFromPath(rel);
        results.push({
          typePath: rel,
          manifestPath: join(dir, entry.name),
          sourceDir: dir,
          sourceName,
          category,
        });
      }
      return; // Don't recurse deeper once manifest found
    }
  }

  // Recurse into subdirectories
  for (const entry of entries) {
    if (entry.isDirectory() && !EXCLUDED_NAMES.has(entry.name)) {
      walkDir(join(dir, entry.name), basePath, sourceName, seen, results);
    }
  }
}

export function discoverTypes(sources: Source[]): ResolvedType[] {
  const seen = new Set<string>();
  const results: ResolvedType[] = [];

  for (const source of sources) {
    for (const resolved of walkSource(source)) {
      if (!seen.has(resolved.typePath)) {
        seen.add(resolved.typePath);
        results.push(resolved);
      }
    }
  }
  return results;
}

export function discoverByCategory(
  sources: Source[],
  category: ManifestType,
): ResolvedType[] {
  return discoverTypes(sources).filter((t) => t.category === category);
}

export function discoverAll(sources: Source[]): DiscoveredType[] {
  const resolved = discoverTypes(sources);
  const enriched: DiscoveredType[] = [];

  for (const r of resolved) {
    try {
      const raw = readFileSync(r.manifestPath, 'utf-8');
      const data = yaml.load(raw) as Record<string, unknown>;
      const base = data as BaseManifest;

      const d: DiscoveredType = {
        ...r,
        version: String(base.version ?? ''),
        description: String(base.description ?? ''),
        tags: Array.isArray(base.tags) ? base.tags.map(String) : [],
      };
      enriched.push(d);
    } catch {
      // Skip types with unparseable manifests
    }
  }
  return enriched;
}

// ── Dependency Tree ─────────────────────────────────────────────────

function extractDependencies(manifestPath: string): string[] {
  const raw = readFileSync(manifestPath, 'utf-8');
  const data = yaml.load(raw) as Record<string, unknown>;
  const type = data.type as string;
  const deps: string[] = [];

  switch (type) {
    case 'prompt': {
      const p = data as unknown as PromptManifest;
      if (p.persona) deps.push(p.persona);
      if (p.context) deps.push(...p.context);
      if (p.skills) deps.push(...p.skills);
      if (p.workflows) deps.push(...p.workflows);
      break;
    }
    case 'workflow': {
      const w = data as unknown as WorkflowManifest;
      if (w.steps) {
        for (const step of w.steps) {
          deps.push(step.skill);
        }
      }
      break;
    }
    case 'persona': {
      const per = data as unknown as PersonaManifest;
      if (per.context) deps.push(...per.context);
      break;
    }
    // context, skill, template have no type-level deps
  }
  return deps;
}

function buildNode(
  typePath: string,
  sources: Source[],
  installedRoot: string,
  seen: Map<string, boolean>,
): DependencyNode {
  const category = categoryFromPath(typePath);
  const node: DependencyNode = {
    typePath,
    category,
    resolved: null,
    children: [],
    deduped: false,
    installed: false,
  };

  if (seen.has(typePath)) {
    node.deduped = true;
    return node;
  }
  seen.set(typePath, true);

  // Check if already installed
  const installedDir = join(installedRoot, typePath);
  if (existsSync(installedDir)) {
    node.installed = true;
  }

  const resolved = resolveType(typePath, sources);
  if (!resolved) return node;
  node.resolved = resolved;

  const deps = extractDependencies(resolved.manifestPath);
  for (const dep of deps) {
    node.children.push(buildNode(dep, sources, installedRoot, seen));
  }

  return node;
}

export function buildDependencyTree(
  typePath: string,
  sources: Source[],
  installedRoot: string,
): DependencyNode {
  return buildNode(typePath, sources, installedRoot, new Map());
}

export function flattenTree(root: DependencyNode): ResolvedType[] {
  const seen = new Set<string>();
  const result: ResolvedType[] = [];
  flattenRecursive(root, seen, result);
  return result;
}

function flattenRecursive(
  node: DependencyNode,
  seen: Set<string>,
  result: ResolvedType[],
): void {
  if (!node || node.deduped || node.installed || seen.has(node.typePath)) return;
  seen.add(node.typePath);

  // Children first = topological order (deps before dependents)
  for (const child of node.children) {
    flattenRecursive(child, seen, result);
  }

  if (node.resolved) {
    result.push(node.resolved);
  }
}

// ── Install Plan ────────────────────────────────────────────────────

function countByCategory(types: ResolvedType[]): Record<string, number> {
  const counts: Record<string, number> = {};
  for (const t of types) {
    counts[t.category] = (counts[t.category] ?? 0) + 1;
  }
  return counts;
}

function countInstalled(node: DependencyNode): number {
  let count = node.installed ? 1 : 0;
  for (const child of node.children) {
    count += countInstalled(child);
  }
  return count;
}

function checkCLIDeps(types: ResolvedType[]): CLIDepStatus[] {
  const seen = new Set<string>();
  const results: CLIDepStatus[] = [];

  for (const t of types) {
    if (t.category !== 'skill') continue;
    try {
      const raw = readFileSync(t.manifestPath, 'utf-8');
      const data = yaml.load(raw) as SkillManifest;
      if (!data.cli_dependencies) continue;
      for (const dep of data.cli_dependencies) {
        if (seen.has(dep.name)) continue;
        seen.add(dep.name);
        let available = false;
        try {
          execFileSync('which', [dep.name], { stdio: 'ignore' });
          available = true;
        } catch {
          // Not found
        }
        results.push({ name: dep.name, available });
      }
    } catch {
      // Skip
    }
  }
  return results;
}

export function buildInstallPlan(
  typePath: string,
  sources: Source[],
  installedRoot: string,
  noDeps = false,
): InstallPlan {
  if (noDeps) {
    const resolved = resolveType(typePath, sources);
    const root: DependencyNode = {
      typePath,
      category: categoryFromPath(typePath),
      resolved,
      children: [],
      deduped: false,
      installed: false,
    };
    const allTypes = resolved ? [resolved] : [];
    return {
      root,
      allTypes,
      counts: countByCategory(allTypes),
      cliDeps: checkCLIDeps(allTypes),
      skipCount: 0,
    };
  }

  const root = buildDependencyTree(typePath, sources, installedRoot);
  const allTypes = flattenTree(root);
  return {
    root,
    allTypes,
    counts: countByCategory(allTypes),
    cliDeps: checkCLIDeps(allTypes),
    skipCount: countInstalled(root),
  };
}

// ── Install / Remove ────────────────────────────────────────────────

export function installType(
  resolved: ResolvedType,
  installedRoot: string,
): void {
  const dst = join(installedRoot, resolved.typePath);
  if (existsSync(dst)) {
    rmSync(dst, { recursive: true });
  }
  copyDirUtil(resolved.sourceDir, dst);
}

export function installNodeDeps(typeDir: string): string | null {
  const pkgPath = join(typeDir, 'package.json');
  if (!existsSync(pkgPath)) return null;

  try {
    execFileSync('which', ['node'], { stdio: 'ignore' });
  } catch {
    return 'Node.js not found — skipping npm install';
  }

  try {
    execFileSync('which', ['npm'], { stdio: 'ignore' });
  } catch {
    return 'npm not found — skipping npm install';
  }

  execFileSync('npm', ['install', '--prefer-offline'], {
    cwd: typeDir,
    stdio: 'ignore',
  });
  return null;
}

export function removeType(
  typePath: string,
  installedRoot: string,
): void {
  const dir = join(installedRoot, typePath);
  if (!existsSync(dir)) {
    throw new Error(`Type not found: ${typePath}`);
  }
  rmSync(dir, { recursive: true });
}

// ── Skill Registry Init ─────────────────────────────────────────────

export function initSkillRegistry(
  resolved: ResolvedType,
  skillsDir: string,
): string[] {
  if (resolved.category !== 'skill') return [];

  const raw = readFileSync(resolved.manifestPath, 'utf-8');
  const data = yaml.load(raw) as SkillManifest;
  if (!data.registry) return [];

  const registryPath = nameFromPath(resolved.typePath);
  const regDir = join(skillsDir, registryPath);
  ensureDir(regDir);

  const warnings: string[] = [];

  // Generate tokens.env
  if (data.registry.tokens?.length) {
    const lines = [`# Environment tokens for ${data.name}`, ''];
    for (const token of data.registry.tokens) {
      if (token.description) lines.push(`# ${token.description}`);
      if (token.required) lines.push('# (required)');
      lines.push(`${token.name}=${token.default ?? ''}`);
      lines.push('');
      if (token.required && !token.default) {
        warnings.push(`Required token ${token.name} has no default value`);
      }
    }
    const tokensPath = join(regDir, 'tokens.env');
    if (!existsSync(tokensPath)) {
      writeFileSync(tokensPath, lines.join('\n'), { mode: 0o600 });
    }
  }

  // Generate config.yaml
  if (data.registry.config && Object.keys(data.registry.config).length > 0) {
    const configPath = join(regDir, 'config.yaml');
    if (!existsSync(configPath)) {
      const content = `# Configuration for ${data.name}\n` + yaml.dump(data.registry.config);
      writeFileSync(configPath, content, { mode: 0o644 });
    }
  }

  // Create optional directories
  if (data.registry.state?.length) ensureDir(join(regDir, 'state'));
  if (data.registry.output) ensureDir(join(regDir, 'output'));
  if (data.registry.templates) ensureDir(join(regDir, 'templates'));

  return warnings;
}

// ── Cache ───────────────────────────────────────────────────────────

interface CachedIndex {
  types: DiscoveredType[];
  sourceMods: Record<string, number>;
  cachedAt: string;
}

export function defaultCachePath(): string {
  return join(getHomeRoot(), 'registry-cache.json');
}

function latestMtime(basePath: string): number {
  let latest = 0;
  try {
    latest = statSync(basePath).mtimeMs;
  } catch {
    return 0;
  }

  for (const cat of KNOWN_CATEGORIES) {
    const catPath = join(basePath, cat);
    try {
      const st = statSync(catPath);
      if (st.mtimeMs > latest) latest = st.mtimeMs;
      // One level deeper
      for (const entry of readdirSync(catPath, { withFileTypes: true })) {
        if (entry.isDirectory()) {
          try {
            const sub = statSync(join(catPath, entry.name));
            if (sub.mtimeMs > latest) latest = sub.mtimeMs;
          } catch {
            // ignore
          }
        }
      }
    } catch {
      // Category dir doesn't exist
    }
  }
  return latest;
}

function loadCache(path: string): CachedIndex | null {
  try {
    const raw = readFileSync(path, 'utf-8');
    return JSON.parse(raw) as CachedIndex;
  } catch {
    return null;
  }
}

function isCacheValid(
  cached: CachedIndex,
  sources: Source[],
): boolean {
  if (Object.keys(cached.sourceMods).length !== sources.length) return false;
  for (const source of sources) {
    const cachedMtime = cached.sourceMods[source.name];
    if (cachedMtime == null) return false;
    if (Math.floor(latestMtime(source.basePath)) !== Math.floor(cachedMtime)) {
      return false;
    }
  }
  return true;
}

function writeCache(
  path: string,
  types: DiscoveredType[],
  sources: Source[],
): void {
  const sourceMods: Record<string, number> = {};
  for (const source of sources) {
    sourceMods[source.name] = latestMtime(source.basePath);
  }
  const index: CachedIndex = {
    types,
    sourceMods,
    cachedAt: new Date().toISOString(),
  };
  try {
    mkdirSync(join(path, '..'), { recursive: true });
    writeFileSync(path, JSON.stringify(index, null, 2));
  } catch {
    // Best-effort cache write
  }
}

export function discoverAllCached(
  sources: Source[],
  cachePath?: string,
): DiscoveredType[] {
  const path = cachePath ?? defaultCachePath();
  const cached = loadCache(path);
  if (cached && isCacheValid(cached, sources)) {
    return cached.types;
  }
  const types = discoverAll(sources);
  writeCache(path, types, sources);
  return types;
}

// ── Print Helpers ───────────────────────────────────────────────────

export function printTree(
  node: DependencyNode,
  prefix = '',
  isLast = true,
): string {
  const connector = prefix === '' ? '' : isLast ? '└── ' : '├── ';
  let label = node.typePath;
  if (node.deduped) label += ' (deduped)';
  if (node.installed) label += ' (already installed)';

  let output = prefix + connector + label + '\n';

  const childPrefix = prefix + (prefix === '' ? '' : isLast ? '    ' : '│   ');
  for (let i = 0; i < node.children.length; i++) {
    output += printTree(
      node.children[i],
      childPrefix,
      i === node.children.length - 1,
    );
  }
  return output;
}
