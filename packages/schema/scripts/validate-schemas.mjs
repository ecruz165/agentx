/**
 * Validates that all JSON Schema files are well-formed and compilable by AJV.
 * Used by `make build` to catch schema errors early.
 */
import { readFileSync } from 'node:fs';
import { resolve, dirname } from 'node:path';
import { fileURLToPath } from 'node:url';
import Ajv2020 from 'ajv/dist/2020.js';
import addFormats from 'ajv-formats';

const __dirname = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dirname, '..');

const schemas = [
  'manifest.schema.json',
  'skill-output.schema.json',
  'skill-error.schema.json',
];

const ajv = new Ajv2020.default({ strict: true, allErrors: true });
addFormats.default(ajv);

let failed = false;

for (const file of schemas) {
  const path = resolve(root, file);
  try {
    const schema = JSON.parse(readFileSync(path, 'utf8'));
    ajv.compile(schema);
    console.log(`  OK  ${file}`);
  } catch (err) {
    console.error(`FAIL  ${file}: ${err.message}`);
    failed = true;
  }
}

if (failed) {
  process.exit(1);
}

console.log('\nAll schemas valid.');
