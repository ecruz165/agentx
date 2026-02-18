import { homedir } from 'node:os';
import { join } from 'node:path';
import {
  readFileSync,
  writeFileSync,
  readdirSync,
  mkdirSync,
  existsSync,
} from 'node:fs';
import yaml from 'js-yaml';
import { HOME_DIR, envVar } from '../config/branding.js';
import { createSymlink, readSymlinkTarget } from '../utils/platform.js';
import { ensureDir, fileExists } from '../utils/fs.js';

// ── Directory constants ─────────────────────────────────────────────

const USERDATA_DIR = 'userdata';
const INSTALLED_DIR = 'installed';
const ENV_DIR = 'env';
const PROFILES_DIR = 'profiles';
const SKILLS_DIR = 'skills';
const PREFERENCES_FILE = 'preferences.yaml';
const DEFAULT_ENV_FILE = 'default.env';
const ACTIVE_PROFILE_LINK = 'active';
const DEFAULT_PROFILE_FILE = 'default.yaml';
const CATALOG_REPO_DIR = 'catalog-repo';
const CATALOG_DIR = 'catalog';
const EXTENSIONS_DIR = 'extensions';

const DIR_PERM_SECURE = 0o700;
const DIR_PERM_NORMAL = 0o755;
const FILE_PERM_SECURE = 0o600;

// ── Mode detection ──────────────────────────────────────────────────

export type Mode = 'end-user' | 'platform-team';

export function detectMode(): Mode {
  return process.env[envVar('HOME')] ? 'platform-team' : 'end-user';
}

// ── Path resolution ─────────────────────────────────────────────────

export function getHomeRoot(): string {
  return process.env[envVar('HOME')] ?? join(homedir(), HOME_DIR);
}

export function getInstalledRoot(): string {
  return process.env[envVar('INSTALLED')] ?? join(getHomeRoot(), INSTALLED_DIR);
}

export function getUserdataRoot(): string {
  return process.env[envVar('USERDATA')] ?? join(getHomeRoot(), USERDATA_DIR);
}

export function getEnvDir(): string {
  return join(getUserdataRoot(), ENV_DIR);
}

export function getProfilesDir(): string {
  return join(getUserdataRoot(), PROFILES_DIR);
}

export function getPreferencesPath(): string {
  return join(getUserdataRoot(), PREFERENCES_FILE);
}

export function getSkillsDir(): string {
  return join(getUserdataRoot(), SKILLS_DIR);
}

export function getVendorEnvPath(vendor: string): string {
  return join(getEnvDir(), `${vendor}.env`);
}

export function getSkillRegistryPath(skillPath: string): string {
  return join(getSkillsDir(), skillPath);
}

export function getCatalogRepoRoot(): string {
  return process.env[envVar('CATALOG')] ?? join(getHomeRoot(), CATALOG_REPO_DIR);
}

export function getCatalogRoot(): string {
  return join(getCatalogRepoRoot(), CATALOG_DIR);
}

export function getExtensionsRoot(): string {
  return process.env[envVar('EXTENSIONS')] ?? join(getHomeRoot(), EXTENSIONS_DIR);
}

export function getConfigDir(): string {
  return getHomeRoot();
}

export function getConfigPath(): string {
  return join(getConfigDir(), 'config.yaml');
}

export function catalogExists(): boolean {
  const root = getCatalogRoot();
  if (!existsSync(root)) return false;
  const entries = readdirSync(root, { withFileTypes: true });
  return entries.some((e) => e.isDirectory());
}

// ── Profile management ──────────────────────────────────────────────

export interface Profile {
  name: string;
  aws_profile?: string;
  aws_region?: string;
  github_org?: string;
  splunk_host?: string;
  default_branch?: string;
  [key: string]: unknown;
}

export function loadProfile(): Profile | null {
  const linkPath = join(getProfilesDir(), ACTIVE_PROFILE_LINK);
  try {
    const target = readSymlinkTarget(linkPath);
    const raw = readFileSync(target, 'utf-8');
    return yaml.load(raw) as Profile;
  } catch {
    return null;
  }
}

export function listProfiles(): string[] {
  try {
    return readdirSync(getProfilesDir())
      .filter((f) => f.endsWith('.yaml'))
      .map((f) => f.replace(/\.yaml$/, ''));
  } catch {
    return [];
  }
}

export function activeProfileName(): string | null {
  const linkPath = join(getProfilesDir(), ACTIVE_PROFILE_LINK);
  try {
    const target = readSymlinkTarget(linkPath);
    const basename = target.split('/').pop() ?? '';
    return basename.replace(/\.yaml$/, '');
  } catch {
    return null;
  }
}

export function switchProfile(name: string): void {
  const profilesDir = getProfilesDir();
  const profilePath = join(profilesDir, `${name}.yaml`);
  if (!fileExists(profilePath)) {
    throw new Error(`Profile "${name}" not found`);
  }
  const linkPath = join(profilesDir, ACTIVE_PROFILE_LINK);
  try {
    const { unlinkSync } = require('node:fs') as typeof import('node:fs');
    unlinkSync(linkPath);
  } catch {
    // Link doesn't exist yet
  }
  createSymlink(profilePath, linkPath);
}

// ── Env file management ─────────────────────────────────────────────

export function listEnvFiles(): { shared: string[]; skillSpecific: string[] } {
  const shared: string[] = [];
  const skillSpecific: string[] = [];

  try {
    const envDir = getEnvDir();
    if (existsSync(envDir)) {
      shared.push(
        ...readdirSync(envDir)
          .filter((f) => f.endsWith('.env'))
          .map((f) => f.replace(/\.env$/, '')),
      );
    }
  } catch {
    // ignore
  }

  try {
    const skillsDir = getSkillsDir();
    if (existsSync(skillsDir)) {
      for (const entry of readdirSync(skillsDir, { withFileTypes: true })) {
        if (entry.isDirectory()) {
          const tokensPath = join(skillsDir, entry.name, 'tokens.env');
          if (fileExists(tokensPath)) {
            skillSpecific.push(entry.name);
          }
        }
      }
    }
  } catch {
    // ignore
  }

  return { shared, skillSpecific };
}

export function resolveEnvTarget(target: string): string {
  if (target.includes('/')) {
    return join(getSkillsDir(), target, 'tokens.env');
  }
  return join(getEnvDir(), `${target}.env`);
}

// ── Preferences ─────────────────────────────────────────────────────

export interface Preferences {
  output_format?: string;
  color?: boolean;
  verbose?: boolean;
  default_persona?: string;
  default_branch?: string;
  editor?: string;
  [key: string]: unknown;
}

export function loadPreferences(): Preferences {
  try {
    const raw = readFileSync(getPreferencesPath(), 'utf-8');
    return (yaml.load(raw) as Preferences) ?? {};
  } catch {
    return {};
  }
}

// ── Init ────────────────────────────────────────────────────────────

const DEFAULT_ENV_CONTENT = `# Shared environment variables loaded by all skills.
LOG_LEVEL=info
OUTPUT_FORMAT=json
`;

const DEFAULT_PROFILE_CONTENT = `name: default
# aws_profile: my-profile
# aws_region: us-east-1
# github_org: my-org
# splunk_host: splunk.example.com
# default_branch: main
`;

const DEFAULT_PREFERENCES_CONTENT = `output_format: json
color: true
verbose: false
# default_persona: senior-java-dev
# default_branch: main
# editor: vim
`;

export function initGlobal(log: (msg: string) => void): void {
  const userdataRoot = getUserdataRoot();
  const envDir = getEnvDir();
  const profilesDir = getProfilesDir();
  const skillsDir = getSkillsDir();

  ensureLogDir(log, userdataRoot, DIR_PERM_NORMAL);
  ensureLogDir(log, envDir, DIR_PERM_SECURE);
  ensureLogDir(log, profilesDir, DIR_PERM_SECURE);
  ensureLogDir(log, skillsDir, DIR_PERM_NORMAL);

  ensureLogFile(log, join(envDir, DEFAULT_ENV_FILE), DEFAULT_ENV_CONTENT, FILE_PERM_SECURE);
  ensureLogFile(log, join(profilesDir, DEFAULT_PROFILE_FILE), DEFAULT_PROFILE_CONTENT, FILE_PERM_SECURE);
  ensureLogFile(log, getPreferencesPath(), DEFAULT_PREFERENCES_CONTENT, FILE_PERM_SECURE);

  // Create active profile symlink
  const linkPath = join(profilesDir, ACTIVE_PROFILE_LINK);
  if (!existsSync(linkPath)) {
    createSymlink(join(profilesDir, DEFAULT_PROFILE_FILE), linkPath);
    log(`  Created: ${linkPath}`);
  }
}

function ensureLogDir(log: (msg: string) => void, path: string, mode: number): void {
  if (!existsSync(path)) {
    mkdirSync(path, { recursive: true, mode });
    log(`  Created: ${path}`);
  }
}

function ensureLogFile(
  log: (msg: string) => void,
  path: string,
  content: string,
  mode: number,
): void {
  if (!existsSync(path)) {
    writeFileSync(path, content, { mode });
    log(`  Created: ${path}`);
  }
}
