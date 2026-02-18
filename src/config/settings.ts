import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import yaml from 'js-yaml';

let configPath = '';
let configData: Record<string, unknown> = {};

export function init(path: string): void {
  configPath = path;
  try {
    const raw = readFileSync(path, 'utf-8');
    configData = (yaml.load(raw) as Record<string, unknown>) ?? {};
  } catch {
    configData = {};
  }
}

export function get(key: string): string {
  const value = configData[key];
  return value != null ? String(value) : '';
}

export function set(key: string, value: string): void {
  configData[key] = value;
  mkdirSync(dirname(configPath), { recursive: true });
  writeFileSync(configPath, yaml.dump(configData), 'utf-8');
}

export function all(): Record<string, unknown> {
  return { ...configData };
}
