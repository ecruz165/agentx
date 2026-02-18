import type { Command } from 'commander';
import * as settings from '../config/settings.js';
import { getConfigPath } from '../core/userdata.js';

export function registerConfig(program: Command): void {
  const cmd = program
    .command('config')
    .description('Manage user settings');

  cmd
    .command('set')
    .description('Set a config value')
    .argument('<key>', 'Config key')
    .argument('<value>', 'Config value')
    .action((key, value) => {
      settings.init(getConfigPath());
      settings.set(key, value);
      console.log(`Set ${key} = ${value}`);
    });

  cmd
    .command('get')
    .description('Get a config value')
    .argument('<key>', 'Config key')
    .action((key) => {
      settings.init(getConfigPath());
      const value = settings.get(key);
      if (value) {
        console.log(value);
      }
    });
}
