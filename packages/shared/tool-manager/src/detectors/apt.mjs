import { BaseDetector } from './base.mjs';

/**
 * Detects whether apt-get is available on Debian/Ubuntu Linux.
 */
export class AptDetector extends BaseDetector {
  get name() {
    return 'apt';
  }

  async isAvailable() {
    const result = await this._shell('which', ['apt-get']);
    if (result.exitCode === 0) {
      return { success: true };
    }
    return { success: false, error: 'apt-get is not available' };
  }
}
