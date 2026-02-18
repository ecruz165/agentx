import type { Command } from 'commander';
import { removeType } from '../core/registry.js';
import { getInstalledRoot } from '../core/userdata.js';
import { ok, fail } from '../ui/output.js';

export function registerUninstall(program: Command): void {
  program
    .command('uninstall')
    .description('Remove an installed type')
    .argument('<type-path>', 'Path to the type to remove')
    .action((typePath) => {
      try {
        const installedRoot = getInstalledRoot();
        removeType(typePath, installedRoot);
        ok(`Removed: ${typePath}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
