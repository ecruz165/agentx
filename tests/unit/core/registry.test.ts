import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { join } from 'node:path';
import { tmpdir } from 'node:os';
import {
  resolveType,
  discoverTypes,
  buildDependencyTree,
  flattenTree,
  buildInstallPlan,
  categoryFromPath,
  nameFromPath,
} from '../../../src/core/registry.js';
import type { Source } from '../../../src/types/registry.js';

function makeManifest(dir: string, content: string): void {
  mkdirSync(dir, { recursive: true });
  writeFileSync(join(dir, 'manifest.yaml'), content);
}

describe('registry', () => {
  let testDir: string;
  let catalogDir: string;
  let installedDir: string;
  let sources: Source[];

  beforeEach(() => {
    testDir = join(tmpdir(), `agentx-test-${Date.now()}`);
    catalogDir = join(testDir, 'catalog');
    installedDir = join(testDir, 'installed');
    mkdirSync(catalogDir, { recursive: true });
    mkdirSync(installedDir, { recursive: true });
    sources = [{ name: 'catalog', basePath: catalogDir }];
  });

  afterEach(() => {
    rmSync(testDir, { recursive: true, force: true });
  });

  describe('categoryFromPath', () => {
    it('maps plural to singular', () => {
      expect(categoryFromPath('skills/scm/git/commit-analyzer')).toBe('skill');
      expect(categoryFromPath('personas/senior-java-dev')).toBe('persona');
      expect(categoryFromPath('context/spring-boot')).toBe('context');
      expect(categoryFromPath('workflows/code-review')).toBe('workflow');
      expect(categoryFromPath('prompts/java-review')).toBe('prompt');
      expect(categoryFromPath('templates/readme')).toBe('template');
    });
  });

  describe('nameFromPath', () => {
    it('strips category prefix', () => {
      expect(nameFromPath('skills/scm/git/commit-analyzer')).toBe('scm/git/commit-analyzer');
      expect(nameFromPath('personas/senior-java-dev')).toBe('senior-java-dev');
    });
  });

  describe('resolveType', () => {
    it('finds type with manifest.yaml', () => {
      makeManifest(join(catalogDir, 'skills/scm/git/commit-analyzer'), `
name: commit-analyzer
type: skill
version: "1.0.0"
description: test
runtime: node
topic: scm
`);
      const result = resolveType('skills/scm/git/commit-analyzer', sources);
      expect(result).not.toBeNull();
      expect(result!.typePath).toBe('skills/scm/git/commit-analyzer');
      expect(result!.category).toBe('skill');
      expect(result!.sourceName).toBe('catalog');
    });

    it('returns null for missing type', () => {
      const result = resolveType('skills/nonexistent', sources);
      expect(result).toBeNull();
    });

    it('resolves from first source (priority)', () => {
      const ext = join(testDir, 'ext');
      makeManifest(join(catalogDir, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "1.0.0"
description: from catalog
runtime: node
topic: test
`);
      makeManifest(join(ext, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "2.0.0"
description: from extension
runtime: node
topic: test
`);
      const result = resolveType('skills/test/basic-skill', [
        ...sources,
        { name: 'ext', basePath: ext },
      ]);
      expect(result!.sourceName).toBe('catalog');
    });
  });

  describe('discoverTypes', () => {
    it('discovers types across categories', () => {
      makeManifest(join(catalogDir, 'skills/scm/git/commit-analyzer'), `
name: commit-analyzer
type: skill
version: "1.0.0"
description: test
runtime: node
topic: scm
`);
      makeManifest(join(catalogDir, 'personas/senior-java-dev'), `
name: senior-java-dev
type: persona
version: "1.0.0"
description: test
`);
      const types = discoverTypes(sources);
      expect(types.length).toBe(2);
      expect(types.map((t) => t.category).sort()).toEqual(['persona', 'skill']);
    });

    it('deduplicates across sources', () => {
      const ext = join(testDir, 'ext');
      makeManifest(join(catalogDir, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "1.0.0"
description: test
runtime: node
topic: test
`);
      makeManifest(join(ext, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "2.0.0"
description: test override
runtime: node
topic: test
`);
      const types = discoverTypes([...sources, { name: 'ext', basePath: ext }]);
      expect(types.length).toBe(1);
      expect(types[0].sourceName).toBe('catalog');
    });
  });

  describe('buildDependencyTree', () => {
    it('builds tree with no dependencies', () => {
      makeManifest(join(catalogDir, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "1.0.0"
description: test
runtime: node
topic: test
`);
      const tree = buildDependencyTree('skills/test/basic-skill', sources, installedDir);
      expect(tree.typePath).toBe('skills/test/basic-skill');
      expect(tree.children.length).toBe(0);
    });

    it('builds tree with persona context dependency', () => {
      makeManifest(join(catalogDir, 'context/spring-boot'), `
name: spring-boot
type: context
version: "1.0.0"
description: test
format: markdown
sources:
  - content.md
`);
      makeManifest(join(catalogDir, 'personas/java-dev'), `
name: java-dev
type: persona
version: "1.0.0"
description: test
context:
  - context/spring-boot
`);
      const tree = buildDependencyTree('personas/java-dev', sources, installedDir);
      expect(tree.children.length).toBe(1);
      expect(tree.children[0].typePath).toBe('context/spring-boot');
    });

    it('marks already-installed types', () => {
      makeManifest(join(catalogDir, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "1.0.0"
description: test
runtime: node
topic: test
`);
      mkdirSync(join(installedDir, 'skills/test/basic-skill'), { recursive: true });
      const tree = buildDependencyTree('skills/test/basic-skill', sources, installedDir);
      expect(tree.installed).toBe(true);
    });
  });

  describe('flattenTree', () => {
    it('returns types in topological order (deps first)', () => {
      makeManifest(join(catalogDir, 'context/spring-boot'), `
name: spring-boot
type: context
version: "1.0.0"
description: test
format: markdown
sources:
  - content.md
`);
      makeManifest(join(catalogDir, 'personas/java-dev'), `
name: java-dev
type: persona
version: "1.0.0"
description: test
context:
  - context/spring-boot
`);
      const tree = buildDependencyTree('personas/java-dev', sources, installedDir);
      const flat = flattenTree(tree);
      expect(flat.length).toBe(2);
      expect(flat[0].typePath).toBe('context/spring-boot');
      expect(flat[1].typePath).toBe('personas/java-dev');
    });

    it('skips installed types', () => {
      makeManifest(join(catalogDir, 'skills/test/basic-skill'), `
name: basic-skill
type: skill
version: "1.0.0"
description: test
runtime: node
topic: test
`);
      mkdirSync(join(installedDir, 'skills/test/basic-skill'), { recursive: true });
      const tree = buildDependencyTree('skills/test/basic-skill', sources, installedDir);
      const flat = flattenTree(tree);
      expect(flat.length).toBe(0);
    });
  });

  describe('buildInstallPlan', () => {
    it('builds plan with counts', () => {
      makeManifest(join(catalogDir, 'context/spring-boot'), `
name: spring-boot
type: context
version: "1.0.0"
description: test
format: markdown
sources:
  - content.md
`);
      makeManifest(join(catalogDir, 'personas/java-dev'), `
name: java-dev
type: persona
version: "1.0.0"
description: test
context:
  - context/spring-boot
`);
      const plan = buildInstallPlan('personas/java-dev', sources, installedDir);
      expect(plan.allTypes.length).toBe(2);
      expect(plan.counts['context']).toBe(1);
      expect(plan.counts['persona']).toBe(1);
    });

    it('respects --no-deps', () => {
      makeManifest(join(catalogDir, 'personas/java-dev'), `
name: java-dev
type: persona
version: "1.0.0"
description: test
context:
  - context/spring-boot
`);
      const plan = buildInstallPlan('personas/java-dev', sources, installedDir, true);
      expect(plan.allTypes.length).toBe(1);
      expect(plan.root.children.length).toBe(0);
    });
  });
});
