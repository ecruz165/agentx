import { defineConfig } from 'tsup';
import { execFileSync } from 'node:child_process';

const version = process.env.npm_package_version ?? 'dev';

let commit = 'unknown';
try {
  commit = execFileSync('git', ['rev-parse', '--short', 'HEAD']).toString().trim();
} catch {
  // Not in a git repo — use fallback
}

const date = new Date().toISOString();

export default defineConfig({
  entry: ['src/cli.ts'],
  format: ['esm'],
  target: 'es2022',
  outDir: 'dist',
  clean: true,
  sourcemap: true,
  define: {
    __VERSION__: JSON.stringify(version),
    __COMMIT__: JSON.stringify(commit),
    __DATE__: JSON.stringify(date),
  },
  banner: {
    js: '// agentx-skillz — Supply chain manager for AI agent configurations',
  },
});
