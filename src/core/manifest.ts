import { readFileSync } from 'node:fs';
import yaml from 'js-yaml';
import { ManifestSchema, type ManifestType } from '../config/schema.js';
import type { Manifest, BaseManifest } from '../types/manifest.js';

export function parseManifest(raw: string): Manifest {
  const data = yaml.load(raw) as Record<string, unknown>;
  return ManifestSchema.parse(data);
}

export function parseManifestFile(path: string): Manifest {
  const raw = readFileSync(path, 'utf-8');
  return parseManifest(raw);
}

export function detectType(raw: string): ManifestType | null {
  const data = yaml.load(raw) as Record<string, unknown> | undefined;
  if (!data || typeof data.type !== 'string') return null;
  const valid: ManifestType[] = [
    'context',
    'persona',
    'skill',
    'workflow',
    'prompt',
    'template',
  ];
  return valid.includes(data.type as ManifestType)
    ? (data.type as ManifestType)
    : null;
}

export function parseBase(raw: string): BaseManifest {
  const data = yaml.load(raw) as Record<string, unknown>;
  return {
    name: String(data.name ?? ''),
    type: String(data.type ?? ''),
    version: String(data.version ?? ''),
    description: String(data.description ?? ''),
    tags: Array.isArray(data.tags) ? data.tags.map(String) : undefined,
    author: typeof data.author === 'string' ? data.author : undefined,
    vendor:
      data.vendor === null
        ? null
        : typeof data.vendor === 'string'
          ? data.vendor
          : undefined,
  };
}

export function parseBaseFile(path: string): BaseManifest {
  const raw = readFileSync(path, 'utf-8');
  return parseBase(raw);
}
