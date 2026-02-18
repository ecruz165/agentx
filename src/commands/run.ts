import type { Command } from 'commander';
import { join } from 'node:path';
import { existsSync, readFileSync } from 'node:fs';
import yaml from 'js-yaml';
import { getInstalledRoot } from '../core/userdata.js';
import { runSkill } from '../core/runtime.js';
import { parseInputArgs, validateInputs } from '../utils/input-parser.js';
import { fail } from '../ui/output.js';
import type { SkillManifest, WorkflowManifest } from '../types/manifest.js';

export function registerRun(program: Command): void {
  program
    .command('run')
    .description('Execute a skill or workflow')
    .argument('<type-path>', 'Path to installed skill or workflow')
    .option('-i, --input <key=value...>', 'Input key=value pairs', collectInputs, [])
    .action(async (typePath, opts) => {
      try {
        const installedRoot = getInstalledRoot();
        const typeDir = join(installedRoot, typePath);

        if (!existsSync(typeDir)) {
          fail(`Type not installed: ${typePath}. Run \`agentx install ${typePath}\` first.`);
          process.exit(1);
        }

        // Find and parse manifest
        const manifestPath = findManifest(typeDir);
        if (!manifestPath) {
          fail(`No manifest found in: ${typeDir}`);
          process.exit(1);
        }

        const raw = readFileSync(manifestPath, 'utf-8');
        const data = yaml.load(raw) as { type: string };
        const inputs = parseInputArgs(opts.input);

        if (data.type === 'skill') {
          const manifest = data as unknown as SkillManifest;

          // Validate inputs
          if (manifest.inputs) {
            const errors = validateInputs(inputs, manifest.inputs);
            if (errors.length > 0) {
              for (const e of errors) fail(e);
              process.exit(1);
            }
          }

          const result = await runSkill(typeDir, manifest, inputs);
          if (result.stdout) process.stdout.write(result.stdout);
          if (result.stderr) process.stderr.write(result.stderr);
          process.exit(result.exitCode);
        } else if (data.type === 'workflow') {
          const manifest = data as unknown as WorkflowManifest;
          // Run workflow steps sequentially
          for (const step of manifest.steps) {
            const skillDir = join(installedRoot, step.skill);
            if (!existsSync(skillDir)) {
              fail(`Workflow step skill not installed: ${step.skill}`);
              process.exit(1);
            }
            const skillManifestPath = findManifest(skillDir);
            if (!skillManifestPath) {
              fail(`No manifest for workflow step: ${step.skill}`);
              process.exit(1);
            }
            const skillRaw = readFileSync(skillManifestPath, 'utf-8');
            const skillManifest = yaml.load(skillRaw) as SkillManifest;
            const stepInputs = step.inputs
              ? Object.fromEntries(
                  Object.entries(step.inputs).map(([k, v]) => [k, String(v)]),
                )
              : {};
            // Merge workflow-level inputs
            const mergedInputs = { ...inputs, ...stepInputs };
            const result = await runSkill(skillDir, skillManifest, mergedInputs);
            if (result.stdout) process.stdout.write(result.stdout);
            if (result.stderr) process.stderr.write(result.stderr);
            if (result.exitCode !== 0) {
              process.exit(result.exitCode);
            }
          }
        } else {
          fail(`Cannot run type: ${data.type}. Only skills and workflows are runnable.`);
          process.exit(1);
        }
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}

function collectInputs(value: string, previous: string[]): string[] {
  return [...previous, value];
}

function findManifest(dir: string): string | null {
  for (const name of ['manifest.yaml', 'manifest.json', 'skill.yaml', 'workflow.yaml']) {
    const path = join(dir, name);
    if (existsSync(path)) return path;
  }
  return null;
}
