import type { Command } from 'commander';
import { execFileSync } from 'node:child_process';
import { existsSync } from 'node:fs';
import {
  getInstalledRoot,
  getUserdataRoot,
  getSkillsDir,
  getCatalogRepoRoot,
  getExtensionsRoot,
  detectMode,
} from '../core/userdata.js';
import { discoverTypes } from '../core/registry.js';
import { ok, fail, warn, info } from '../ui/output.js';

function checkCommand(name: string): boolean {
  try {
    execFileSync('which', [name], { stdio: 'ignore' });
    return true;
  } catch {
    return false;
  }
}

export function registerDoctor(program: Command): void {
  program
    .command('doctor')
    .description('Health check for installation')
    .option('--check-cli', 'Check CLI dependencies for installed skills')
    .option('--check-runtime', 'Check node/git availability')
    .option('--check-links', 'Check symlinks')
    .option('--check-extensions', 'Check extensions')
    .option('--check-userdata', 'Check userdata directory')
    .option('--check-registry', 'Check skill registries')
    .option('--check-manifest <path>', 'Validate a specific manifest file')
    .action((opts) => {
      const anyCheck = opts.checkCli || opts.checkRuntime || opts.checkLinks ||
        opts.checkExtensions || opts.checkUserdata || opts.checkRegistry || opts.checkManifest;
      const runAll = !anyCheck;

      console.log('\nAgentX Doctor\n');
      console.log(`  Mode: ${detectMode()}`);
      console.log('');

      // Runtime checks
      if (runAll || opts.checkRuntime) {
        console.log('Runtime:');
        for (const cmd of ['node', 'npm', 'git']) {
          if (checkCommand(cmd)) {
            ok(`  ${cmd} — available`);
          } else {
            fail(`  ${cmd} — not found`);
          }
        }
        console.log('');
      }

      // Userdata checks
      if (runAll || opts.checkUserdata) {
        console.log('Userdata:');
        for (const [label, path] of [
          ['Userdata root', getUserdataRoot()],
          ['Installed root', getInstalledRoot()],
          ['Skills dir', getSkillsDir()],
          ['Catalog repo', getCatalogRepoRoot()],
        ] as const) {
          if (existsSync(path)) {
            ok(`  ${label} — ${path}`);
          } else {
            warn(`  ${label} — missing (${path})`);
          }
        }
        console.log('');
      }

      // CLI dependency checks
      if (runAll || opts.checkCli) {
        console.log('CLI Dependencies:');
        const installedRoot = getInstalledRoot();
        if (!existsSync(installedRoot)) {
          info('  No installed types found.');
        } else {
          const types = discoverTypes([{ name: 'installed', basePath: installedRoot }]);
          const skills = types.filter((t) => t.category === 'skill');
          if (skills.length === 0) {
            info('  No skills installed.');
          } else {
            const { readFileSync } = require('node:fs');
            const yaml = require('js-yaml');
            for (const skill of skills) {
              try {
                const raw = readFileSync(skill.manifestPath, 'utf-8');
                const data = yaml.load(raw) as { cli_dependencies?: { name: string }[] };
                if (data.cli_dependencies) {
                  for (const dep of data.cli_dependencies) {
                    if (checkCommand(dep.name)) {
                      ok(`  ${dep.name} (for ${skill.typePath})`);
                    } else {
                      fail(`  ${dep.name} (for ${skill.typePath}) — not found`);
                    }
                  }
                }
              } catch {
                // Skip unreadable manifests
              }
            }
          }
        }
        console.log('');
      }

      // Manifest validation
      if (opts.checkManifest) {
        console.log('Manifest Validation:');
        try {
          const { parseManifestFile } = require('../core/manifest.js');
          parseManifestFile(opts.checkManifest);
          ok(`  Valid: ${opts.checkManifest}`);
        } catch (err) {
          fail(`  Invalid: ${opts.checkManifest} — ${err}`);
        }
        console.log('');
      }

      ok('Doctor complete.');
    });
}
