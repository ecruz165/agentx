import { BaseDetector } from './base.mjs';

/**
 * Detects whether winget is available on Windows.
 */
export class WingetDetector extends BaseDetector {
  get name() {
    return 'winget';
  }

  async isAvailable() {
    const result = await this._shell('where', ['winget']);
    if (result.exitCode === 0) {
      return { success: true };
    }
    return { success: false, error: 'winget is not installed' };
  }
}
