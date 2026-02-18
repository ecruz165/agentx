import type { InputField } from '../types/manifest.js';

export function parseInputArgs(args: string[]): Record<string, string> {
  const result: Record<string, string> = {};
  for (const arg of args) {
    const eqIndex = arg.indexOf('=');
    if (eqIndex === -1) {
      throw new Error(`Invalid input format: "${arg}". Expected key=value.`);
    }
    result[arg.slice(0, eqIndex)] = arg.slice(eqIndex + 1);
  }
  return result;
}

export function validateInputs(
  provided: Record<string, string>,
  schema: InputField[],
): string[] {
  const errors: string[] = [];
  for (const field of schema) {
    if (field.required && !(field.name in provided)) {
      errors.push(`Missing required input: ${field.name}`);
    }
  }
  return errors;
}
