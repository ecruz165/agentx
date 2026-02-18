import {
  symlinkSync,
  unlinkSync,
  readlinkSync,
  lstatSync,
  copyFileSync,
  readFileSync,
  writeFileSync,
  existsSync,
} from 'node:fs';
import { resolve, dirname } from 'node:path';

const isWindows = process.platform === 'win32';

export function createSymlink(target: string, link: string): void {
  if (isWindows) {
    try {
      symlinkSync(target, link);
    } catch {
      // Fallback: copy file + write .target sidecar for Windows without symlink perms
      const absTarget = resolve(dirname(link), target);
      copyFileSync(absTarget, link);
      writeFileSync(`${link}.target`, target, 'utf-8');
    }
  } else {
    symlinkSync(target, link);
  }
}

export function removeSymlink(path: string): void {
  unlinkSync(path);
  const sidecar = `${path}.target`;
  if (existsSync(sidecar)) {
    unlinkSync(sidecar);
  }
}

export function readSymlinkTarget(path: string): string {
  try {
    return readlinkSync(path);
  } catch {
    // Windows fallback: read .target sidecar
    const sidecar = `${path}.target`;
    if (existsSync(sidecar)) {
      return readFileSync(sidecar, 'utf-8').trim();
    }
    throw new Error(`Cannot read symlink target: ${path}`);
  }
}

export function isSymlinkSupported(): boolean {
  return !isWindows;
}

export function isSymlink(path: string): boolean {
  try {
    return lstatSync(path).isSymbolicLink();
  } catch {
    return false;
  }
}
