import type { Command } from 'commander';
import { discoverAllCached } from '../core/registry.js';
import { buildSources } from '../core/extension.js';
import { findRepoRoot } from '../utils/git.js';
import { printTable } from '../ui/table.js';
import type { DiscoveredType } from '../types/registry.js';

export function registerSearch(program: Command): void {
  program
    .command('search')
    .description('Search available types across all sources')
    .argument('[query]', 'Substring match on name/description/path')
    .option('--type <category>', 'Filter by type (skill, workflow, prompt, persona, context, template)')
    .option('--tag <tags>', 'Comma-separated tags (matches any)')
    .option('--topic <topic>', 'Filter by topic (exact)')
    .option('--vendor <vendor>', 'Filter by vendor (exact)')
    .option('--cli <dependency>', 'Filter by CLI dependency')
    .option('--json', 'Output as JSON')
    .action((query, opts) => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        const sources = buildSources(repoRoot);
        let types = discoverAllCached(sources);

        if (query) {
          const q = query.toLowerCase();
          types = types.filter(
            (t) =>
              t.typePath.toLowerCase().includes(q) ||
              t.description.toLowerCase().includes(q),
          );
        }

        if (opts.type) {
          types = types.filter((t) => t.category === opts.type);
        }

        if (opts.tag) {
          const tags = opts.tag.split(',').map((t: string) => t.trim().toLowerCase());
          types = types.filter((t) =>
            t.tags.some((tag) => tags.includes(tag.toLowerCase())),
          );
        }

        if (opts.json) {
          console.log(JSON.stringify(types, null, 2));
          return;
        }

        if (types.length === 0) {
          console.log('No types found.');
          return;
        }

        printTable(
          ['Type', 'Name', 'Version', 'Description'],
          types.map((t) => [t.category, t.typePath, t.version, t.description]),
        );
      } catch (err) {
        console.error(String(err));
        process.exit(1);
      }
    });
}
