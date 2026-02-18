import ora, { type Ora } from 'ora';

export function startSpinner(text: string): Ora {
  return ora(text).start();
}

export async function withSpinner<T>(
  text: string,
  fn: () => Promise<T>,
): Promise<T> {
  const spinner = ora(text).start();
  try {
    const result = await fn();
    spinner.succeed();
    return result;
  } catch (err) {
    spinner.fail();
    throw err;
  }
}
