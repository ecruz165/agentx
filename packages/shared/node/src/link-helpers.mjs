import { readFileSync, writeFileSync, mkdirSync, existsSync, statSync, symlinkSync, unlinkSync, readdirSync, lstatSync } from 'node:fs';
import { join, dirname } from 'node:path';
import { parse as parseYaml } from 'yaml';

/**
 * Load a manifest file for an installed type.
 * Looks for manifest.yaml first, falls back to manifest.json.
 * @param {string} installedPath - Root of installed types (e.g., ~/.agentx/installed/)
 * @param {string} typeRef - Type reference (e.g., "personas/senior-java-dev", "context/spring-boot/error-handling")
 * @returns {{ manifest: object, path: string } | null}
 */
export function loadManifest(installedPath, typeRef) {
  const typeDir = join(installedPath, typeRef);
  const yamlPath = join(typeDir, 'manifest.yaml');
  const jsonPath = join(typeDir, 'manifest.json');

  if (existsSync(yamlPath)) {
    const content = readFileSync(yamlPath, 'utf8');
    return { manifest: parseYaml(content), path: yamlPath };
  }

  if (existsSync(jsonPath)) {
    const content = readFileSync(jsonPath, 'utf8');
    return { manifest: JSON.parse(content), path: jsonPath };
  }

  return null;
}

/**
 * Create a symlink, removing an existing one first if present.
 * @param {string} target - Absolute path the symlink points to
 * @param {string} linkPath - Absolute path where the symlink is created
 */
export function createSymlink(target, linkPath) {
  ensureDir(dirname(linkPath));
  try {
    const lstats = lstatSync(linkPath);
    // Remove existing file or symlink (including broken symlinks)
    if (lstats.isSymbolicLink() || lstats.isFile() || lstats.isDirectory()) {
      unlinkSync(linkPath);
    }
  } catch {
    // Path doesn't exist — nothing to remove
  }
  symlinkSync(target, linkPath);
}

/**
 * Flatten a type reference to a single-level hyphenated name.
 * Strips the type prefix and replaces slashes with hyphens.
 * e.g., "context/spring-boot/error-handling" -> "spring-boot-error-handling"
 * @param {string} ref - Type reference path
 * @returns {string}
 */
export function flattenRef(ref) {
  const parts = ref.split('/');
  // Strip the type prefix (context/, personas/, skills/, workflows/)
  if (parts.length > 1) {
    return parts.slice(1).join('-');
  }
  return parts[0];
}

/**
 * Check if generated files are stale relative to a source file.
 * @param {string} sourceFile - Path to the source file (e.g., project.yaml)
 * @param {string[]} generatedFiles - Paths to generated files
 * @returns {boolean} True if source is newer than any generated file
 */
export function isStale(sourceFile, generatedFiles) {
  if (!existsSync(sourceFile)) return false;
  const sourceMtime = statSync(sourceFile).mtimeMs;

  for (const file of generatedFiles) {
    if (!existsSync(file)) return true;
    if (sourceMtime > statSync(file).mtimeMs) return true;
  }

  return false;
}

/**
 * Recursively create directories.
 * @param {string} dirPath - Directory path to create
 */
export function ensureDir(dirPath) {
  mkdirSync(dirPath, { recursive: true });
}

/**
 * Validate that all symlinks in a directory are pointing to valid targets.
 * @param {string} dirPath - Directory containing symlinks
 * @returns {{ total: number, valid: number, broken: string[] }}
 */
export function validateSymlinks(dirPath) {
  if (!existsSync(dirPath)) {
    return { total: 0, valid: 0, broken: [] };
  }

  const entries = readdirSync(dirPath);
  let total = 0;
  let valid = 0;
  const broken = [];

  for (const entry of entries) {
    const fullPath = join(dirPath, entry);
    try {
      const lstats = lstatSync(fullPath);
      if (lstats.isSymbolicLink()) {
        total++;
        if (existsSync(fullPath)) {
          valid++;
        } else {
          broken.push(entry);
        }
      }
    } catch {
      // Skip entries that can't be read
    }
  }

  return { total, valid, broken };
}
