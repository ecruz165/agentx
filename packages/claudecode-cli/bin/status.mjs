#!/usr/bin/env node
import { status } from '../src/index.mjs';

let input = '';
process.stdin.setEncoding('utf8');
process.stdin.on('data', (chunk) => { input += chunk; });
process.stdin.on('end', async () => {
  try {
    const { projectPath } = JSON.parse(input);
    const result = await status(projectPath);
    process.stdout.write(JSON.stringify(result));
  } catch (err) {
    process.stderr.write(JSON.stringify({ error: err.message }));
    process.exit(1);
  }
});
