import type { Command } from 'commander';
import {
  addExtension,
  removeExtension,
  listExtensions,
  syncExtensions,
} from '../core/extension.js';
import { findRepoRoot } from '../utils/git.js';
import { ok, fail } from '../ui/output.js';
import { printTable } from '../ui/table.js';
import { withSpinner } from '../ui/spinner.js';

export function registerExtension(program: Command): void {
  const cmd = program
    .command('extension')
    .alias('ext')
    .description('Manage extension repositories');

  cmd
    .command('add')
    .description('Add an extension repository')
    .argument('<name>', 'Extension name')
    .argument('<git-url>', 'Git repository URL')
    .option('--branch <branch>', 'Git branch to track', 'main')
    .action(async (name, gitURL, opts) => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        await withSpinner(`Adding extension ${name}...`, () =>
          addExtension(repoRoot, name, gitURL, opts.branch),
        );
        ok(`Extension added: ${name}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('remove')
    .description('Remove an extension')
    .argument('<name>', 'Extension name')
    .action(async (name) => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        await removeExtension(repoRoot, name);
        ok(`Extension removed: ${name}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('list')
    .description('List extensions')
    .action(async () => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        const extensions = await listExtensions(repoRoot);
        if (extensions.length === 0) {
          console.log('No extensions found.');
          return;
        }
        printTable(
          ['Name', 'Path', 'Branch', 'Status'],
          extensions.map((e) => [e.name, e.path, e.branch, e.status]),
        );
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('sync')
    .description('Sync all extensions')
    .action(async () => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        await withSpinner('Syncing extensions...', () => syncExtensions(repoRoot));
        ok('Extensions synced.');
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
