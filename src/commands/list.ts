import type { Command } from 'commander';
import { getInstalledRoot } from '../core/userdata.js';
import { discoverTypes } from '../core/registry.js';
import { printTable } from '../ui/table.js';
import { parseBaseFile } from '../core/manifest.js';

export function registerList(program: Command): void {
  program
    .command('list')
    .description('List installed types')
    .option('--type <category>', 'Filter by type')
    .option('--json', 'Output as JSON')
    .action((opts) => {
      try {
        const installedRoot = getInstalledRoot();
        const sources = [{ name: 'installed', basePath: installedRoot }];
        let types = discoverTypes(sources);

        if (opts.type) {
          types = types.filter((t) => t.category === opts.type);
        }

        if (opts.json) {
          const enriched = types.map((t) => {
            try {
              const base = parseBaseFile(t.manifestPath);
              return { ...t, version: base.version, description: base.description };
            } catch {
              return t;
            }
          });
          console.log(JSON.stringify(enriched, null, 2));
          return;
        }

        if (types.length === 0) {
          console.log('No installed types found.');
          return;
        }

        const rows = types.map((t) => {
          try {
            const base = parseBaseFile(t.manifestPath);
            return [t.category, t.typePath, base.version];
          } catch {
            return [t.category, t.typePath, '?'];
          }
        });

        printTable(['Type', 'Path', 'Version'], rows);
      } catch (err) {
        console.error(String(err));
        process.exit(1);
      }
    });
}
