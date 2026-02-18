import {
  mkdirSync,
  readdirSync,
  copyFileSync,
  statSync,
} from 'node:fs';
import { join } from 'node:path';

const SKIP_DIRS = new Set(['node_modules', '.git', 'dist']);

export function copyDir(src: string, dest: string): void {
  mkdirSync(dest, { recursive: true });
  for (const entry of readdirSync(src, { withFileTypes: true })) {
    const srcPath = join(src, entry.name);
    const destPath = join(dest, entry.name);
    if (entry.isDirectory()) {
      if (!SKIP_DIRS.has(entry.name)) {
        copyDir(srcPath, destPath);
      }
    } else {
      copyFileSync(srcPath, destPath);
    }
  }
}

export function ensureDir(path: string, mode?: number): void {
  mkdirSync(path, { recursive: true, mode });
}

export function dirExists(path: string): boolean {
  try {
    return statSync(path).isDirectory();
  } catch {
    return false;
  }
}

export function fileExists(path: string): boolean {
  try {
    return statSync(path).isFile();
  } catch {
    return false;
  }
}
