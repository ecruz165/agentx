/**
 * Stubs for @agentx/shared-node functions used by integrations.
 * These will be replaced by `npx toolz` calls later.
 */

import { readFileSync, mkdirSync, symlinkSync, lstatSync, readdirSync, statSync, existsSync, unlinkSync } from 'node:fs';
import { join } from 'node:path';
import yaml from 'js-yaml';

export interface LoadedManifest {
  manifest: Record<string, unknown>;
  path: string;
}

/** Load a manifest.yaml from the installed types directory. */
export function loadManifest(installedPath: string, ref: string): LoadedManifest | null {
  const manifestPath = join(installedPath, ref, 'manifest.yaml');
  try {
    const raw = readFileSync(manifestPath, 'utf-8');
    const manifest = yaml.load(raw) as Record<string, unknown>;
    return { manifest, path: manifestPath };
  } catch {
    return null;
  }
}

/** Create a symlink, replacing any existing one. */
export function createSymlink(target: string, linkPath: string): void {
  try {
    lstatSync(linkPath);
    unlinkSync(linkPath);
  } catch {
    // Link doesn't exist — that's fine
  }
  symlinkSync(target, linkPath);
}

/** Flatten a type ref like "context/security/owasp" → "context--security--owasp". */
export function flattenRef(ref: string): string {
  return ref.replace(/\//g, '--');
}

/** Check if any target file is older than the source file. */
export function isStale(sourcePath: string, targetPaths: string[]): boolean {
  try {
    const sourceStat = statSync(sourcePath);
    for (const target of targetPaths) {
      try {
        const targetStat = statSync(target);
        if (sourceStat.mtimeMs > targetStat.mtimeMs) {
          return true;
        }
      } catch {
        return true;
      }
    }
    return false;
  } catch {
    return false;
  }
}

/** Ensure a directory exists (recursive). */
export function ensureDir(dirPath: string): void {
  mkdirSync(dirPath, { recursive: true });
}

/** Validate symlinks in a directory, returning total and valid counts. */
export function validateSymlinks(dirPath: string): { total: number; valid: number } {
  try {
    const entries = readdirSync(dirPath);
    let total = 0;
    let valid = 0;
    for (const entry of entries) {
      const fullPath = join(dirPath, entry);
      try {
        const stat = lstatSync(fullPath);
        if (stat.isSymbolicLink()) {
          total++;
          if (existsSync(fullPath)) {
            valid++;
          }
        }
      } catch {
        // Skip unreadable entries
      }
    }
    return { total, valid };
  } catch {
    return { total: 0, valid: 0 };
  }
}
