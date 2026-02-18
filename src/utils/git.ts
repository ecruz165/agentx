import { execFileSync } from 'node:child_process';

export function findRepoRoot(cwd?: string): string | null {
  try {
    return execFileSync('git', ['rev-parse', '--show-toplevel'], {
      cwd: cwd ?? process.cwd(),
      encoding: 'utf-8',
    }).trim();
  } catch {
    return null;
  }
}
