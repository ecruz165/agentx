export const APP_NAME = 'agentx';
export const DISPLAY_NAME = 'AgentX';
export const DESCRIPTION = 'Supply chain manager for AI agent configurations';
export const HOME_DIR = '.agentx';
export const ENV_PREFIX = 'AGENTX';
export const GITHUB_REPO = 'ecruz165/agentx';
export const CATALOG_REPO_URL = 'https://github.com/ecruz165/agentx.git';
export const NPM_PACKAGE = 'agentx-skillz';

export function envVar(suffix: string): string {
  return `${ENV_PREFIX}_${suffix.toUpperCase()}`;
}
