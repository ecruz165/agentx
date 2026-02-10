import { BaseDetector } from './base.mjs';

/**
 * Detects whether Homebrew is available on macOS.
 */
export class HomebrewDetector extends BaseDetector {
  get name() {
    return 'homebrew';
  }

  async isAvailable() {
    const result = await this._shell('which', ['brew']);
    if (result.exitCode === 0) {
      return { success: true };
    }
    return { success: false, error: 'Homebrew is not installed' };
  }
}
