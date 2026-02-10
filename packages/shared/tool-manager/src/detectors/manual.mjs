import { BaseDetector } from './base.mjs';

/**
 * Manual fallback detector. Always reports as available since manual
 * installation instructions are always an option.
 */
export class ManualDetector extends BaseDetector {
  get name() {
    return 'manual';
  }

  async isAvailable() {
    return { success: true };
  }
}
