import type { Command } from 'commander';
import { existsSync } from 'node:fs';
import { initGlobal, getCatalogRepoRoot, catalogExists } from '../core/userdata.js';
import { initProject, projectConfigPath } from '../core/linker.js';
import { clone } from '../core/catalog.js';
import { ALL_TOOLS } from '../types/integrations.js';
import { ok, warn, fail } from '../ui/output.js';
import { withSpinner } from '../ui/spinner.js';

export function registerInit(program: Command): void {
  program
    .command('init')
    .description('Initialize AgentX configuration')
    .option('--global', 'Initialize global userdata directory')
    .option('--tools <list>', 'Comma-separated AI tools to configure', ALL_TOOLS.join(','))
    .action(async (opts) => {
      try {
        if (opts.global) {
          console.log('Initializing global userdata...');
          initGlobal((msg) => console.log(msg));

          if (!catalogExists()) {
            const catalogDir = getCatalogRepoRoot();
            await withSpinner('Cloning catalog...', () => clone(catalogDir));
          }
          ok('Global initialization complete.');
          return;
        }

        const projectPath = process.cwd();
        const configPath = projectConfigPath(projectPath);
        if (existsSync(configPath)) {
          warn('Project already initialized.');
          return;
        }

        const tools = opts.tools.split(',').map((t: string) => t.trim());
        initProject(projectPath, tools);
        ok(`Project initialized with tools: ${tools.join(', ')}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
