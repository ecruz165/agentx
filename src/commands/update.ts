import type { Command } from 'commander';
import { checkForUpdate, update, currentVersion } from '../core/updater.js';
import { ok, info, fail } from '../ui/output.js';
import { withSpinner } from '../ui/spinner.js';

export function registerUpdate(program: Command): void {
  program
    .command('update')
    .alias('self-update')
    .description('Update agentx CLI')
    .option('--check', 'Check for updates without installing')
    .option('--force', 'Force update even if on latest')
    .option('--version <version>', 'Install specific version')
    .action(async (opts) => {
      try {
        if (opts.check) {
          info(`Current version: ${currentVersion()}`);
          const latest = await checkForUpdate();
          if (latest) {
            info(`New version available: ${latest}`);
            console.log('Run `agentx update` to install.');
          } else {
            ok('Already on the latest version.');
          }
          return;
        }

        if (opts.version) {
          await withSpinner(`Installing version ${opts.version}...`, () =>
            update(opts.version),
          );
          ok(`Updated to ${opts.version}`);
          return;
        }

        const latest = await checkForUpdate();
        if (!latest && !opts.force) {
          ok(`Already on the latest version (${currentVersion()}).`);
          return;
        }

        await withSpinner('Updating...', () => update());
        ok('Updated successfully.');
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
