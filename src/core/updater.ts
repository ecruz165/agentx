import { execFileSync } from 'node:child_process';
import { NPM_PACKAGE } from '../config/branding.js';

declare const __VERSION__: string;

export function currentVersion(): string {
  return typeof __VERSION__ !== 'undefined' ? __VERSION__ : 'dev';
}

export async function checkForUpdate(): Promise<string | null> {
  try {
    const latest = execFileSync('npm', ['view', NPM_PACKAGE, 'version'], {
      encoding: 'utf-8',
    }).trim();

    if (latest && latest !== currentVersion()) {
      return latest;
    }
    return null;
  } catch {
    return null;
  }
}

export async function update(version?: string): Promise<void> {
  const pkg = version ? `${NPM_PACKAGE}@${version}` : NPM_PACKAGE;
  execFileSync('npm', ['install', '-g', pkg], { stdio: 'inherit' });
}
