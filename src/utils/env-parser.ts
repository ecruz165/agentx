export interface EnvEntry {
  key: string;
  value: string;
}

const SENSITIVE_PATTERNS = ['TOKEN', 'SECRET', 'PASSWORD', 'KEY', 'CREDENTIAL'];

export function parseEnvFile(content: string): EnvEntry[] {
  const entries: EnvEntry[] = [];
  for (const line of content.split('\n')) {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) continue;
    const eqIndex = trimmed.indexOf('=');
    if (eqIndex === -1) continue;
    entries.push({
      key: trimmed.slice(0, eqIndex).trim(),
      value: trimmed.slice(eqIndex + 1).trim(),
    });
  }
  return entries;
}

export function redactValue(key: string, value: string): string {
  const upper = key.toUpperCase();
  if (SENSITIVE_PATTERNS.some((p) => upper.includes(p))) {
    return value.length >= 4 ? value.slice(0, 4) + '***' : '***';
  }
  return value;
}
