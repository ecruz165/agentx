import type { Command } from 'commander';
import { readFileSync, writeFileSync, existsSync, mkdirSync } from 'node:fs';
import { dirname } from 'node:path';
import { execFileSync } from 'node:child_process';
import { listEnvFiles, resolveEnvTarget } from '../core/userdata.js';
import { parseEnvFile, redactValue } from '../utils/env-parser.js';

export function registerEnv(program: Command): void {
  const cmd = program
    .command('env')
    .description('Manage environment/secret files');

  cmd
    .command('list')
    .description('List environment files')
    .action(() => {
      const { shared, skillSpecific } = listEnvFiles();
      if (shared.length) {
        console.log('Shared:');
        for (const name of shared) console.log(`  ${name}`);
      }
      if (skillSpecific.length) {
        console.log('Skill-specific:');
        for (const name of skillSpecific) console.log(`  ${name}`);
      }
      if (!shared.length && !skillSpecific.length) {
        console.log('No environment files found.');
      }
    });

  cmd
    .command('edit')
    .description('Edit an environment file')
    .argument('<target>', 'Target name (e.g., aws, cloud/aws/ssm)')
    .action((target) => {
      const path = resolveEnvTarget(target);
      if (!existsSync(path)) {
        mkdirSync(dirname(path), { recursive: true });
        writeFileSync(path, `# Environment variables for ${target}\n`, { mode: 0o600 });
      }
      const editor = process.env.EDITOR ?? 'vi';
      try {
        execFileSync(editor, [path], { stdio: 'inherit' });
      } catch (err) {
        console.error(`Failed to open editor: ${err}`);
        process.exit(1);
      }
    });

  cmd
    .command('show')
    .description('Show environment file contents')
    .argument('<target>', 'Target name')
    .option('--no-redact', 'Show actual values')
    .action((target, opts) => {
      const path = resolveEnvTarget(target);
      if (!existsSync(path)) {
        console.error(`File not found: ${path}`);
        process.exit(1);
      }
      const content = readFileSync(path, 'utf-8');
      const entries = parseEnvFile(content);
      for (const entry of entries) {
        const value = opts.redact === false ? entry.value : redactValue(entry.key, entry.value);
        console.log(`${entry.key}=${value}`);
      }
    });
}
