import type { Command } from 'commander';
import {
  addType,
  removeType,
  sync,
  status,
} from '../core/linker.js';
import { ok, fail, warn } from '../ui/output.js';
import { printTable } from '../ui/table.js';

export function registerLink(program: Command): void {
  const cmd = program
    .command('link')
    .description('Manage linked types in project');

  cmd
    .command('add')
    .description('Add a type reference to the project')
    .argument('<type-path>', 'Type path (e.g., personas/senior-java-dev)')
    .action(async (typePath) => {
      try {
        await addType(process.cwd(), typePath);
        ok(`Linked: ${typePath}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('remove')
    .description('Remove a type reference from the project')
    .argument('<type-path>', 'Type path to remove')
    .action(async (typePath) => {
      try {
        await removeType(process.cwd(), typePath);
        ok(`Unlinked: ${typePath}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('sync')
    .description('Regenerate all AI tool configuration files')
    .action(async () => {
      try {
        const results = await sync(process.cwd());
        for (const r of results) {
          if (r.warnings.length) {
            for (const w of r.warnings) warn(`${r.tool}: ${w}`);
          } else {
            ok(`${r.tool}: ${r.created.length} created, ${r.updated.length} updated, ${r.symlinked.length} symlinked`);
          }
        }
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('status')
    .description('Show link status for all tools')
    .action(async () => {
      try {
        const results = await status(process.cwd());
        if (results.length === 0) {
          console.log('No tools configured.');
          return;
        }
        printTable(
          ['Tool', 'Status', 'Files', 'Symlinks'],
          results.map((r) => [
            r.tool,
            r.status,
            String(r.files.length),
            `${r.symlinks.valid}/${r.symlinks.total}`,
          ]),
        );
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
