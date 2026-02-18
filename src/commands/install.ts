import type { Command } from 'commander';
import { join } from 'node:path';
import { getInstalledRoot, getSkillsDir } from '../core/userdata.js';
import {
  buildInstallPlan,
  installType,
  installNodeDeps,
  initSkillRegistry,
  printTree,
  nameFromPath,
} from '../core/registry.js';
import { buildSources } from '../core/extension.js';
import { findRepoRoot } from '../utils/git.js';
import { ok, fail, warn, info } from '../ui/output.js';
import { askConfirm } from '../ui/prompts.js';

export function registerInstall(program: Command): void {
  program
    .command('install')
    .description('Install a type and its dependencies')
    .argument('<type-path>', 'Path to the type (e.g., skills/scm/git/commit-analyzer)')
    .option('--no-deps', 'Skip dependency resolution')
    .option('-y, --yes', 'Skip confirmation prompt')
    .action(async (typePath, opts) => {
      try {
        const repoRoot = findRepoRoot() ?? process.cwd();
        const sources = buildSources(repoRoot);
        const installedRoot = getInstalledRoot();
        const noDeps = opts.deps === false;

        const plan = buildInstallPlan(typePath, sources, installedRoot, noDeps);

        if (plan.allTypes.length === 0) {
          info('Nothing to install — all types already present.');
          return;
        }

        // Show plan
        console.log('\nInstall plan:\n');
        console.log(printTree(plan.root));

        const counts = Object.entries(plan.counts)
          .map(([k, v]) => `${v} ${k}(s)`)
          .join(', ');
        console.log(`Types to install: ${counts}`);

        if (plan.skipCount > 0) {
          console.log(`Already installed: ${plan.skipCount}`);
        }

        if (plan.cliDeps.length > 0) {
          console.log('\nCLI dependencies:');
          for (const dep of plan.cliDeps) {
            console.log(`  ${dep.available ? '✓' : '✗'} ${dep.name}`);
          }
        }

        // Confirm
        if (!opts.yes) {
          const confirmed = await askConfirm('\nProceed with installation?');
          if (!confirmed) {
            console.log('Cancelled.');
            return;
          }
        }

        // Install
        for (const resolved of plan.allTypes) {
          const name = nameFromPath(resolved.typePath);
          process.stdout.write(`Installing ${name}...`);
          installType(resolved, installedRoot);

          // npm install for Node skills/workflows
          const typeDir = join(installedRoot, resolved.typePath);
          const npmWarning = installNodeDeps(typeDir);
          if (npmWarning) warn(npmWarning);

          // Init skill registry
          if (resolved.category === 'skill') {
            const warnings = initSkillRegistry(resolved, getSkillsDir());
            for (const w of warnings) warn(w);
          }

          console.log(' done');
        }

        ok(`Installed ${plan.allTypes.length} type(s).`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
