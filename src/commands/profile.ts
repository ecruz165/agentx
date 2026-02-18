import type { Command } from 'commander';
import yaml from 'js-yaml';
import {
  listProfiles,
  activeProfileName,
  loadProfile,
  switchProfile,
} from '../core/userdata.js';
import { ok, fail } from '../ui/output.js';

export function registerProfile(program: Command): void {
  const cmd = program
    .command('profile')
    .description('Manage user configuration profiles');

  cmd
    .command('list')
    .description('List available profiles')
    .action(() => {
      const profiles = listProfiles();
      const active = activeProfileName();
      if (profiles.length === 0) {
        console.log('No profiles found. Run `agentx init --global` first.');
        return;
      }
      for (const name of profiles) {
        const marker = name === active ? ' (active)' : '';
        console.log(`  ${name}${marker}`);
      }
    });

  cmd
    .command('use')
    .description('Switch to a profile')
    .argument('<name>', 'Profile name')
    .action((name) => {
      try {
        switchProfile(name);
        ok(`Switched to profile: ${name}`);
      } catch (err) {
        fail(String(err));
        process.exit(1);
      }
    });

  cmd
    .command('show')
    .description('Show current profile')
    .option('--yaml', 'Output as YAML')
    .option('--json', 'Output as JSON')
    .action((opts) => {
      const profile = loadProfile();
      if (!profile) {
        console.log('No active profile.');
        return;
      }
      if (opts.json) {
        console.log(JSON.stringify(profile, null, 2));
      } else if (opts.yaml) {
        console.log(yaml.dump(profile));
      } else {
        for (const [key, value] of Object.entries(profile)) {
          if (value != null && value !== '') {
            console.log(`  ${key}: ${value}`);
          }
        }
      }
    });
}
