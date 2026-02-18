export * from './manifest.js';
export * from './userdata.js';

// Selective re-exports to avoid name conflicts
export {
  resolveType,
  discoverTypes,
  discoverByCategory,
  discoverAll,
  discoverAllCached,
  buildDependencyTree,
  flattenTree,
  buildInstallPlan,
  installType,
  installNodeDeps,
  removeType as removeInstalledType,
  initSkillRegistry,
  categoryFromPath,
  nameFromPath,
  printTree,
  defaultCachePath,
} from './registry.js';

export {
  loadProject,
  saveProject,
  initProject,
  projectConfigPath,
  addType,
  removeType as unlinkType,
  sync,
  status,
} from './linker.js';

export { compose, render } from './compose.js';
export { runSkill } from './runtime.js';

export {
  clone as cloneCatalog,
  update as updateCatalog,
  isStale,
  readFreshnessMarker,
  repoURL,
} from './catalog.js';

export { generate as generateScaffold, newScaffoldData } from './scaffold.js';

export {
  addExtension,
  removeExtension,
  listExtensions,
  syncExtensions,
  buildSources,
} from './extension.js';

export {
  checkForUpdate,
  update as updateCli,
  currentVersion,
} from './updater.js';
