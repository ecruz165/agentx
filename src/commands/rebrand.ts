import { Command } from 'commander';
import { readFileSync, writeFileSync, readdirSync } from 'node:fs';
import { join } from 'node:path';
import yaml from 'js-yaml';
import chalk from 'chalk';
import {
  APP_NAME,
  DISPLAY_NAME,
  DESCRIPTION,
  HOME_DIR,
  ENV_PREFIX,
  GITHUB_REPO,
  CATALOG_REPO_URL,
  NPM_PACKAGE,
} from '../config/branding.js';

interface BrandingConfig {
  cli_name: string;
  display_name: string;
  description: string;
  npm_package: string;
  home_dir: string;
  env_prefix: string;
  github_repo: string;
  catalog_repo_url: string;
}

interface Replacement {
  from: string;
  to: string;
  label: string;
}

function findProjectRoot(): string {
  // At runtime, code runs from dist/ — go up one level to project root
  return join(import.meta.dirname ?? '.', '..');
}

function loadBrandingYaml(projectRoot: string): BrandingConfig {
  const raw = readFileSync(join(projectRoot, 'branding.yaml'), 'utf-8');
  return yaml.load(raw) as BrandingConfig;
}

function buildReplacements(current: BrandingConfig, next: BrandingConfig): Replacement[] {
  const pairs: Replacement[] = [];

  const fields: Array<{ key: keyof BrandingConfig; label: string }> = [
    { key: 'cli_name', label: 'CLI name' },
    { key: 'display_name', label: 'Display name' },
    { key: 'description', label: 'Description' },
    { key: 'npm_package', label: 'npm package' },
    { key: 'home_dir', label: 'Home dir' },
    { key: 'env_prefix', label: 'Env prefix' },
    { key: 'github_repo', label: 'GitHub repo' },
    { key: 'catalog_repo_url', label: 'Catalog URL' },
  ];

  for (const { key, label } of fields) {
    if (current[key] !== next[key]) {
      pairs.push({ from: current[key], to: next[key], label });
    }
  }

  return pairs;
}

function currentBranding(): BrandingConfig {
  return {
    cli_name: APP_NAME,
    display_name: DISPLAY_NAME,
    description: DESCRIPTION,
    npm_package: NPM_PACKAGE,
    home_dir: HOME_DIR,
    env_prefix: ENV_PREFIX,
    github_repo: GITHUB_REPO,
    catalog_repo_url: CATALOG_REPO_URL,
  };
}

function applyReplacements(
  content: string,
  replacements: Replacement[],
): string {
  let result = content;
  for (const { from, to } of replacements) {
    result = result.replaceAll(from, to);
  }
  return result;
}

function collectFiles(projectRoot: string): string[] {
  const files: string[] = [];

  // src/**/*.ts
  const srcDir = join(projectRoot, 'src');
  const srcEntries = readdirSync(srcDir, { recursive: true, encoding: 'utf-8' });
  for (const entry of srcEntries) {
    if (entry.endsWith('.ts')) {
      files.push(join(srcDir, entry));
    }
  }

  // bin/**/*.js
  const binDir = join(projectRoot, 'bin');
  const binEntries = readdirSync(binDir, { recursive: true, encoding: 'utf-8' });
  for (const entry of binEntries) {
    if (entry.endsWith('.js')) {
      files.push(join(binDir, entry));
    }
  }

  // package.json
  files.push(join(projectRoot, 'package.json'));

  return files;
}

export function registerRebrand(program: Command): void {
  program
    .command('rebrand', { hidden: true })
    .description('Apply branding from branding.yaml across the source tree')
    .option('--dry-run', 'Show what would change without writing files')
    .action(async (opts: { dryRun?: boolean }) => {
      const projectRoot = findProjectRoot();
      const next = loadBrandingYaml(projectRoot);
      const current = currentBranding();
      const replacements = buildReplacements(current, next);

      if (replacements.length === 0) {
        console.log(chalk.green('Branding is already up to date.'));
        return;
      }

      console.log(chalk.bold('Rebrand replacements:'));
      for (const r of replacements) {
        console.log(`  ${r.label}: ${chalk.red(r.from)} → ${chalk.green(r.to)}`);
      }
      console.log();

      const files = collectFiles(projectRoot);
      let changed = 0;

      for (const file of files) {
        const original = readFileSync(file, 'utf-8');
        const updated = applyReplacements(original, replacements);

        if (original !== updated) {
          changed++;
          const rel = file.replace(projectRoot + '/', '');
          if (opts.dryRun) {
            console.log(chalk.yellow(`  would update: ${rel}`));
          } else {
            writeFileSync(file, updated, 'utf-8');
            console.log(chalk.green(`  updated: ${rel}`));
          }
        }
      }

      if (opts.dryRun) {
        console.log(`\n${chalk.yellow(`Dry run complete — ${changed} file(s) would change.`)}`);
      } else {
        console.log(`\n${chalk.green(`Rebrand complete — ${changed} file(s) updated.`)}`);
        console.log(chalk.dim('Run `pnpm run build` to rebuild with new branding.'));
      }
    });
}
