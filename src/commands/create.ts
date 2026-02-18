import type { Command } from 'commander';
import { join } from 'node:path';
import { newScaffoldData, generate } from '../core/scaffold.js';
import { ok, fail } from '../ui/output.js';

const NAME_PATTERN = /^[a-z0-9][a-z0-9-]*$/;

function validateName(name: string, label: string): void {
  if (!NAME_PATTERN.test(name)) {
    throw new Error(
      `Invalid ${label}: "${name}". Must be lowercase alphanumeric with hyphens.`,
    );
  }
}

export function registerCreate(program: Command): void {
  const cmd = program
    .command('create')
    .description('Scaffold new types from templates');

  // ── create skill ──────────────────────────────────────────────
  cmd
    .command('skill')
    .description('Create a new skill')
    .argument('<name>', 'Skill name (kebab-case)')
    .requiredOption('--topic <topic>', 'Skill topic (kebab-case)')
    .option('--vendor <vendor>', 'Vendor name')
    .option('--runtime <runtime>', 'Runtime: node or go', 'node')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        validateName(opts.topic, 'topic');
        if (opts.vendor) validateName(opts.vendor, 'vendor');
        const data = newScaffoldData(name, 'skill', opts.topic, opts.vendor ?? '', opts.runtime);
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('skill', data, outDir);
        ok(`Created skill at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  // ── create workflow ───────────────────────────────────────────
  cmd
    .command('workflow')
    .description('Create a new workflow')
    .argument('<name>', 'Workflow name')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        const data = newScaffoldData(name, 'workflow', '', '', 'node');
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('workflow', data, outDir);
        ok(`Created workflow at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  // ── create prompt ─────────────────────────────────────────────
  cmd
    .command('prompt')
    .description('Create a new prompt')
    .argument('<name>', 'Prompt name')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        const data = newScaffoldData(name, 'prompt', '', '', '');
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('prompt', data, outDir);
        ok(`Created prompt at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  // ── create persona ────────────────────────────────────────────
  cmd
    .command('persona')
    .description('Create a new persona')
    .argument('<name>', 'Persona name')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        const data = newScaffoldData(name, 'persona', '', '', '');
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('persona', data, outDir);
        ok(`Created persona at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  // ── create context ────────────────────────────────────────────
  cmd
    .command('context')
    .description('Create a new context')
    .argument('<name>', 'Context name')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        const data = newScaffoldData(name, 'context', '', '', '');
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('context', data, outDir);
        ok(`Created context at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  // ── create template ───────────────────────────────────────────
  cmd
    .command('template')
    .description('Create a new template')
    .argument('<name>', 'Template name')
    .option('--output-dir <dir>', 'Output directory')
    .action((name, opts) => {
      try {
        validateName(name, 'name');
        const data = newScaffoldData(name, 'template', '', '', '');
        const outDir = opts.outputDir ?? join(process.cwd(), name);
        const result = generate('template', data, outDir);
        ok(`Created template at ${result.outputDir}`);
        for (const f of result.files) console.log(`  ${f}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}
