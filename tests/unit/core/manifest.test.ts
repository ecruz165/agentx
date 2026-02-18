import { describe, it, expect } from 'vitest';
import { parseManifest, detectType, parseBase } from '../../../src/core/manifest.js';

describe('manifest', () => {
  describe('detectType', () => {
    it('detects context type', () => {
      expect(detectType('type: context\nname: test')).toBe('context');
    });

    it('detects skill type', () => {
      expect(detectType('type: skill\nname: test')).toBe('skill');
    });

    it('returns null for unknown type', () => {
      expect(detectType('type: unknown\nname: test')).toBeNull();
    });

    it('returns null for missing type field', () => {
      expect(detectType('name: test')).toBeNull();
    });
  });

  describe('parseBase', () => {
    it('extracts base fields', () => {
      const raw = `name: test-context
type: context
version: "1.0.0"
description: A test context
tags:
  - test
  - example
author: test-author`;
      const base = parseBase(raw);
      expect(base.name).toBe('test-context');
      expect(base.type).toBe('context');
      expect(base.version).toBe('1.0.0');
      expect(base.description).toBe('A test context');
      expect(base.tags).toEqual(['test', 'example']);
      expect(base.author).toBe('test-author');
    });
  });

  describe('parseManifest', () => {
    it('parses valid context manifest', () => {
      const raw = `name: spring-boot
type: context
version: "1.0.0"
description: Spring Boot context
format: markdown
sources:
  - content.md`;
      const manifest = parseManifest(raw);
      expect(manifest.type).toBe('context');
      if (manifest.type === 'context') {
        expect(manifest.format).toBe('markdown');
        expect(manifest.sources).toEqual(['content.md']);
      }
    });

    it('parses valid skill manifest', () => {
      const raw = `name: commit-analyzer
type: skill
version: "1.0.0"
description: Analyzes commits
runtime: node
topic: scm`;
      const manifest = parseManifest(raw);
      expect(manifest.type).toBe('skill');
      if (manifest.type === 'skill') {
        expect(manifest.runtime).toBe('node');
        expect(manifest.topic).toBe('scm');
      }
    });

    it('throws on invalid manifest', () => {
      expect(() => parseManifest('not valid yaml: [[')).toThrow();
    });
  });
});
