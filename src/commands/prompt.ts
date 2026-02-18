import type { Command } from 'commander';
import { writeFileSync } from 'node:fs';
import { execFileSync } from 'node:child_process';
import { getInstalledRoot } from '../core/userdata.js';
import { compose, render } from '../core/compose.js';
import { ok, fail } from '../ui/output.js';

export function registerPrompt(program: Command): void {
  program
    .command('prompt')
    .description('Compose a prompt from installed types')
    .argument('[prompt-type-path]', 'Path to installed prompt type')
    .option('--copy', 'Copy output to clipboard')
    .option('-o, --output <file>', 'Write output to file')
    .action((promptPath, opts) => {
      try {
        if (!promptPath) {
          console.log('Interactive mode not yet implemented. Provide a prompt type path.');
          process.exit(1);
        }

        const installedRoot = getInstalledRoot();
        const composed = compose(promptPath, installedRoot);
        const output = render(composed);

        if (composed.warnings.length) {
          for (const w of composed.warnings) {
            console.error(`⚠ ${w}`);
          }
        }

        if (opts.output) {
          writeFileSync(opts.output, output, 'utf-8');
          ok(`Written to: ${opts.output}`);
        } else if (opts.copy) {
          copyToClipboard(output);
          ok('Copied to clipboard.');
        } else {
          console.log(output);
        }
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });
}

function copyToClipboard(text: string): void {
  const platform = process.platform;
  let cmd: string;
  let args: string[];

  if (platform === 'darwin') {
    cmd = 'pbcopy';
    args = [];
  } else if (platform === 'win32') {
    cmd = 'clip';
    args = [];
  } else {
    cmd = 'xclip';
    args = ['-selection', 'clipboard'];
  }

  execFileSync(cmd, args, { input: text });
}
