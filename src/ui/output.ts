import chalk from 'chalk';

export const ok = (msg: string) => console.log(chalk.green('✓'), msg);
export const fail = (msg: string) => console.error(chalk.red('✗'), msg);
export const warn = (msg: string) => console.error(chalk.yellow('⚠'), msg);
export const info = (msg: string) => console.log(chalk.blue('ℹ'), msg);

export function die(msg: string): never {
  fail(msg);
  process.exit(1);
}
