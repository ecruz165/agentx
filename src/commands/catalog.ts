import type { Command } from 'commander';
import {
  getCatalogRepoRoot,
  detectMode,
  catalogExists,
} from '../core/userdata.js';
import {
  update,
  isStale,
  readFreshnessMarker,
  repoURL,
} from '../core/catalog.js';
import { APP_NAME } from '../config/branding.js';
import { ok, warn, fail } from '../ui/output.js';
import { withSpinner } from '../ui/spinner.js';

export function registerCatalog(program: Command): void {
  const cmd = program
    .command('catalog')
    .description('Manage the type catalog');

  cmd
    .command('update')
    .description('Pull latest catalog from remote')
    .action(async () => {
      const mode = detectMode();
      if (mode === 'platform-team') {
        console.log(`Platform-team mode: use \`git pull\` in your catalog repository.`);
        return;
      }

      const catalogRepoDir = getCatalogRepoRoot();
      try {
        await withSpinner('Updating catalog...', () => update(catalogRepoDir));
        ok('Catalog updated.');
      } catch (err) {
        fail(`Failed to update catalog: ${err}`);
        process.exit(1);
      }
    });

  cmd
    .command('status')
    .description('Show catalog status and location')
    .action(() => {
      const mode = detectMode();
      const catalogRepoDir = getCatalogRepoRoot();
      const exists = catalogExists();

      console.log(`  Mode:     ${mode}`);
      console.log(`  Path:     ${catalogRepoDir}`);
      console.log(`  Repo URL: ${repoURL()}`);

      if (!exists) {
        warn(`Catalog not installed. Run \`${APP_NAME} catalog update\` to clone.`);
        return;
      }

      const lastUpdated = readFreshnessMarker(catalogRepoDir);
      const age = Date.now() - lastUpdated.getTime();
      const days = Math.floor(age / (1000 * 60 * 60 * 24));
      console.log(`  Updated:  ${lastUpdated.toISOString()} (${days} days ago)`);

      if (isStale(catalogRepoDir)) {
        warn(`Catalog is stale. Run \`${APP_NAME} catalog update\`.`);
      } else {
        ok('Catalog is up to date.');
      }
    });
}
