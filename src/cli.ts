import { Command } from 'commander';
import { APP_NAME, DESCRIPTION, DISPLAY_NAME } from './config/branding.js';
import {
  registerVersion,
  registerInit,
  registerInstall,
  registerUninstall,
  registerSearch,
  registerList,
  registerCatalog,
  registerLink,
  registerRun,
  registerCreate,
  registerDoctor,
  registerProfile,
  registerEnv,
  registerExtension,
  registerConfig,
  registerPrompt,
  registerUpdate,
  registerRebrand,
} from './commands/index.js';

const program = new Command()
  .name(APP_NAME)
  .description(
    `${DISPLAY_NAME} manages the installation, linking, and discovery of reusable types\n` +
      '(skills, workflows, prompts, personas, context) that power AI coding assistants.',
  )
  .enablePositionalOptions()
  .showHelpAfterError(true);

// Register all commands
registerVersion(program);
registerInit(program);
registerInstall(program);
registerUninstall(program);
registerSearch(program);
registerList(program);
registerCatalog(program);
registerLink(program);
registerRun(program);
registerCreate(program);
registerDoctor(program);
registerProfile(program);
registerEnv(program);
registerExtension(program);
registerConfig(program);
registerPrompt(program);
registerUpdate(program);
registerRebrand(program);

program.parse();
