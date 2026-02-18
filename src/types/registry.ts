import type { ManifestType } from '../config/schema.js';

export interface Source {
  name: string;
  basePath: string;
}

export interface ResolvedType {
  typePath: string;
  manifestPath: string;
  sourceDir: string;
  sourceName: string;
  category: ManifestType;
}

export interface DependencyNode {
  typePath: string;
  category: string;
  resolved: ResolvedType | null;
  children: DependencyNode[];
  deduped: boolean;
  installed: boolean;
}

export interface CLIDepStatus {
  name: string;
  available: boolean;
}

export interface InstallPlan {
  root: DependencyNode;
  allTypes: ResolvedType[];
  counts: Record<string, number>;
  cliDeps: CLIDepStatus[];
  skipCount: number;
}

export interface InstallResult {
  installed: number;
  skipped: number;
  warnings: string[];
}

export interface DiscoveredType extends ResolvedType {
  version: string;
  description: string;
  tags: string[];
}
