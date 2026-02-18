import type { Command } from 'commander';
import { APP_NAME } from '../config/branding.js';

declare const __VERSION__: string;
declare const __COMMIT__: string;
declare const __DATE__: string;

export function registerVersion(program: Command): void {
  program
    .command('version')
    .description('Print version information')
    .option('--short', 'Print version number only')
    .option('--json', 'Print version info as JSON')
    .action((opts) => {
      const version = typeof __VERSION__ !== 'undefined' ? __VERSION__ : 'dev';
      const commit = typeof __COMMIT__ !== 'undefined' ? __COMMIT__ : 'unknown';
      const date = typeof __DATE__ !== 'undefined' ? __DATE__ : 'unknown';

      if (opts.short) {
        console.log(version);
        return;
      }

      if (opts.json) {
        console.log(JSON.stringify({ version, commit, date }, null, 2));
        return;
      }

      console.log(`${APP_NAME} version ${version} (commit: ${commit}, built: ${date})`);
    });
}
