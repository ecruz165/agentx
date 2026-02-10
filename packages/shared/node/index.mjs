// @agentx/shared-node - Public API
// Shared utilities for Node-based skills and CLI

// Output formatting
export { formatOutput, formatSuccess, formatError, formatTable } from './src/output-formatter.mjs';

// Error handling
export { AgentXError, handleError, wrapAsync } from './src/error-handler.mjs';

// CLI runner
export { runCommand, parseOutput } from './src/cli-runner.mjs';

// Registry helpers
export {
  getUserdataRoot,
  getSkillRegistryPath,
  loadEnvChain,
  readState,
  writeState,
  saveOutput,
  loadTemplate,
  saveTemplate,
  listTemplates,
  readConfig,
} from './src/registry-helpers.mjs';

// Link helpers (AI tool integration)
export {
  loadManifest,
  createSymlink,
  flattenRef,
  isStale,
  ensureDir,
  validateSymlinks,
} from './src/link-helpers.mjs';
