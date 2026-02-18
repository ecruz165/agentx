export type ToolName = 'claude-code' | 'copilot' | 'augment' | 'opencode';

export const ALL_TOOLS: ToolName[] = [
  'claude-code',
  'copilot',
  'augment',
  'opencode',
];

export function parseToolName(s: string): ToolName | null {
  return ALL_TOOLS.includes(s as ToolName) ? (s as ToolName) : null;
}

export interface GenerateResult {
  tool: ToolName;
  created: string[];
  updated: string[];
  symlinked: string[];
  warnings: string[];
}

export interface StatusResult {
  tool: string;
  status: string;
  files: string[];
  symlinks: {
    total: number;
    valid: number;
  };
}
