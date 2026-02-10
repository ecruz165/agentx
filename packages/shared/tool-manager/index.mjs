// @agentx/tool-manager - Public API
// CLI dependency detection and guided installation

export { ToolManager } from './src/tool-manager.mjs';

export { BaseDetector } from './src/detectors/base.mjs';
export { HomebrewDetector } from './src/detectors/homebrew.mjs';
export { WingetDetector } from './src/detectors/winget.mjs';
export { AptDetector } from './src/detectors/apt.mjs';
export { ManualDetector } from './src/detectors/manual.mjs';

export { BaseInstaller } from './src/installers/base.mjs';
export { HomebrewInstaller } from './src/installers/homebrew.mjs';
export { WingetInstaller } from './src/installers/winget.mjs';
export { AptInstaller } from './src/installers/apt.mjs';
export { ManualInstaller } from './src/installers/manual.mjs';

export { loadRegistry } from './src/registry-loader.mjs';
export { createShellExecutor } from './src/shell.mjs';
export { promptInstall } from './src/prompt-install.mjs';
