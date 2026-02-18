import { join } from 'node:path';
import { existsSync, rmSync } from 'node:fs';
import { simpleGit } from 'simple-git';
import type { Source } from '../types/registry.js';
import { getExtensionsRoot, detectMode } from './userdata.js';

export interface ExtensionStatus {
  name: string;
  path: string;
  branch: string;
  status: string; // 'ok' | 'uninitialized' | 'modified' | 'missing'
}

export async function addExtension(
  repoRoot: string,
  name: string,
  gitURL: string,
  branch = 'main',
): Promise<void> {
  const mode = detectMode();
  if (mode === 'platform-team') {
    const git = simpleGit(repoRoot);
    const extPath = join('extensions', name);
    await git.submoduleAdd(gitURL, extPath);
    if (branch !== 'main') {
      const extGit = simpleGit(join(repoRoot, extPath));
      await extGit.checkout(branch);
    }
  } else {
    const extDir = join(getExtensionsRoot(), name);
    const git = simpleGit();
    await git.clone(gitURL, extDir, ['--branch', branch, '--depth', '1']);
  }
}

export async function removeExtension(
  repoRoot: string,
  name: string,
): Promise<void> {
  const mode = detectMode();
  if (mode === 'platform-team') {
    const git = simpleGit(repoRoot);
    const extPath = join('extensions', name);
    await git.raw(['submodule', 'deinit', '-f', extPath]);
    await git.rm(extPath);
    const modulesDir = join(repoRoot, '.git', 'modules', extPath);
    if (existsSync(modulesDir)) {
      rmSync(modulesDir, { recursive: true });
    }
  } else {
    const extDir = join(getExtensionsRoot(), name);
    if (existsSync(extDir)) {
      rmSync(extDir, { recursive: true });
    }
  }
}

export async function listExtensions(
  repoRoot: string,
): Promise<ExtensionStatus[]> {
  const mode = detectMode();
  const results: ExtensionStatus[] = [];

  if (mode === 'platform-team') {
    const git = simpleGit(repoRoot);
    try {
      const output = await git.raw(['submodule', 'status']);
      for (const line of output.trim().split('\n')) {
        if (!line.trim()) continue;
        const parts = line.trim().split(/\s+/);
        const path = parts[1] ?? '';
        if (!path.startsWith('extensions/')) continue;
        const name = path.replace('extensions/', '');
        const statusChar = line.charAt(0);
        let status = 'ok';
        if (statusChar === '-') status = 'uninitialized';
        else if (statusChar === '+') status = 'modified';
        results.push({ name, path: join(repoRoot, path), branch: '', status });
      }
    } catch {
      // No submodules
    }
  } else {
    const extRoot = getExtensionsRoot();
    if (!existsSync(extRoot)) return [];
    const { readdirSync } = await import('node:fs');
    for (const entry of readdirSync(extRoot, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const extDir = join(extRoot, entry.name);
      let status = 'ok';
      if (!existsSync(join(extDir, '.git'))) status = 'missing';
      results.push({ name: entry.name, path: extDir, branch: '', status });
    }
  }
  return results;
}

export async function syncExtensions(repoRoot: string): Promise<void> {
  const mode = detectMode();
  if (mode === 'platform-team') {
    const git = simpleGit(repoRoot);
    await git.raw(['submodule', 'update', '--init', '--recursive']);
  } else {
    const extRoot = getExtensionsRoot();
    if (!existsSync(extRoot)) return;
    const { readdirSync } = await import('node:fs');
    for (const entry of readdirSync(extRoot, { withFileTypes: true })) {
      if (!entry.isDirectory()) continue;
      const extGit = simpleGit(join(extRoot, entry.name));
      await extGit.pull(['--rebase']);
    }
  }
}

export function buildSources(repoRoot: string): Source[] {
  const sources: Source[] = [];
  const mode = detectMode();

  // Catalog source
  const { getCatalogRoot, getExtensionsRoot: getExtRoot } = require('./userdata.js');
  const catalogRoot = getCatalogRoot();
  if (existsSync(catalogRoot)) {
    sources.push({ name: 'catalog', basePath: catalogRoot });
  }

  // Extension sources
  const extRoot = getExtRoot();
  if (existsSync(extRoot)) {
    try {
      const { readdirSync: readdir } = require('node:fs');
      for (const entry of readdir(extRoot, { withFileTypes: true })) {
        if (entry.isDirectory()) {
          sources.push({ name: entry.name, basePath: join(extRoot, entry.name) });
        }
      }
    } catch {
      // ignore
    }
  }

  return sources;
}
