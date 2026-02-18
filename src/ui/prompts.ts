import { confirm, select, input } from '@inquirer/prompts';

export async function askConfirm(message: string, defaultValue = true): Promise<boolean> {
  return confirm({ message, default: defaultValue });
}

export async function askSelect<T extends string>(
  message: string,
  choices: { name: string; value: T }[],
): Promise<T> {
  return select({ message, choices });
}

export async function askInput(message: string, defaultValue?: string): Promise<string> {
  return input({ message, default: defaultValue });
}
