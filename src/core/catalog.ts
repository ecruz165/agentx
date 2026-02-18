import { existsSync, writeFileSync, readFileSync, renameSync, rmSync } from 'node:fs';
import { simpleGit } from 'simple-git';
import { CATALOG_REPO_URL, envVar } from '../config/branding.js';
import * as settings from '../config/settings.js';

const FRESHNESS_FILE = '.catalog-updated';
const DEFAULT_MAX_AGE_MS = 7 * 24 * 60 * 60 * 1000; // 7 days

export function repoURL(): string {
  return process.env[envVar('CATALOG_URL')]
    ?? settings.get('catalog_url')
    ?? CATALOG_REPO_URL;
}

export async function clone(targetDir: string): Promise<void> {
  const url = repoURL();
  const tmpDir = targetDir + '.tmp';

  // Clean up any stale tmp directory
  if (existsSync(tmpDir)) {
    rmSync(tmpDir, { recursive: true });
  }

  const git = simpleGit();

  // Try sparse checkout first (git >= 2.25.0)
  try {
    await git.clone(url, tmpDir, [
      '--depth', '1',
      '--filter=blob:none',
      '--sparse',
    ]);
    const tmpGit = simpleGit(tmpDir);
    await tmpGit.raw(['sparse-checkout', 'set', 'catalog']);
  } catch {
    // Fallback: full shallow clone
    if (existsSync(tmpDir)) {
      rmSync(tmpDir, { recursive: true });
    }
    await git.clone(url, tmpDir, ['--depth', '1']);
  }

  // Atomic rename
  if (existsSync(targetDir)) {
    rmSync(targetDir, { recursive: true });
  }
  renameSync(tmpDir, targetDir);
  writeFreshnessMarker(targetDir);
}

export async function update(catalogRepoDir: string): Promise<void> {
  if (!existsSync(catalogRepoDir)) {
    await clone(catalogRepoDir);
    return;
  }

  const git = simpleGit(catalogRepoDir);
  await git.pull();
  writeFreshnessMarker(catalogRepoDir);
}

export function writeFreshnessMarker(catalogRepoDir: string): void {
  const marker = `${catalogRepoDir}/${FRESHNESS_FILE}`;
  writeFileSync(marker, String(Math.floor(Date.now() / 1000)));
}

export function readFreshnessMarker(catalogRepoDir: string): Date {
  const marker = `${catalogRepoDir}/${FRESHNESS_FILE}`;
  try {
    const ts = parseInt(readFileSync(marker, 'utf-8').trim(), 10);
    return new Date(ts * 1000);
  } catch {
    return new Date(0);
  }
}

export function isStale(
  catalogRepoDir: string,
  maxAgeMs = DEFAULT_MAX_AGE_MS,
): boolean {
  const lastUpdated = readFreshnessMarker(catalogRepoDir);
  return Date.now() - lastUpdated.getTime() > maxAgeMs;
}
