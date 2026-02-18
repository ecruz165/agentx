import { spawn } from 'node:child_process';
import { join } from 'node:path';
import { readFileSync, existsSync } from 'node:fs';
import type { SkillManifest } from '../types/manifest.js';
import { getSkillRegistryPath, getUserdataRoot } from './userdata.js';
import { parseEnvFile } from '../utils/env-parser.js';
import { envVar } from '../config/branding.js';
import { nameFromPath } from './registry.js';

export interface RuntimeOutput {
  exitCode: number;
  stdout: string;
  stderr: string;
}

export async function runSkill(
  skillPath: string,
  manifest: SkillManifest,
  args: Record<string, string>,
): Promise<RuntimeOutput> {
  switch (manifest.runtime) {
    case 'node':
      return runNodeSkill(skillPath, manifest, args);
    case 'go':
      throw new Error('Go runtime is not yet supported');
    default:
      throw new Error(`Unknown runtime: ${manifest.runtime}`);
  }
}

async function runNodeSkill(
  skillPath: string,
  manifest: SkillManifest,
  args: Record<string, string>,
): Promise<RuntimeOutput> {
  const entryPoint = join(skillPath, 'index.mjs');
  if (!existsSync(entryPoint)) {
    throw new Error(`Skill entry point not found: ${entryPoint}`);
  }

  const env = buildNodeEnv(skillPath, manifest);

  return new Promise((resolve, reject) => {
    const child = spawn('node', [entryPoint, 'run', JSON.stringify(args)], {
      env: { ...process.env, ...env },
      stdio: ['pipe', 'pipe', 'pipe'],
    });

    let stdout = '';
    let stderr = '';
    child.stdout.on('data', (data: Buffer) => {
      stdout += data.toString();
    });
    child.stderr.on('data', (data: Buffer) => {
      stderr += data.toString();
    });

    child.on('error', reject);
    child.on('close', (code) => {
      resolve({ exitCode: code ?? 1, stdout, stderr });
    });
  });
}

function buildNodeEnv(
  skillPath: string,
  manifest: SkillManifest,
): Record<string, string> {
  const env: Record<string, string> = {};

  env[envVar('USERDATA')] = getUserdataRoot();
  env[envVar('SKILL_PATH')] = skillPath;

  const registryName = nameFromPath(
    skillPath.includes('/installed/')
      ? skillPath.split('/installed/')[1]
      : skillPath,
  );
  const registryPath = getSkillRegistryPath(registryName);
  env[envVar('SKILL_REGISTRY')] = registryPath;

  // Load tokens.env
  const tokensPath = join(registryPath, 'tokens.env');
  if (existsSync(tokensPath)) {
    const content = readFileSync(tokensPath, 'utf-8');
    for (const entry of parseEnvFile(content)) {
      if (entry.value) {
        env[entry.key] = entry.value;
      }
    }
  }

  return env;
}
