import { join, dirname } from 'node:path';
import {
  readFileSync,
  writeFileSync,
  readdirSync,
  mkdirSync,
  existsSync,
  statSync,
} from 'node:fs';
import { fileURLToPath } from 'node:url';

export interface ScaffoldData {
  name: string;
  topic: string;
  vendor: string;
  runtime: string;
  description: string;
  version: string;
  packageName: string;
  skillPath: string;
  year: number;
}

export interface ScaffoldResult {
  outputDir: string;
  files: string[];
  warnings: string[];
}

export function newScaffoldData(
  name: string,
  typeName: string,
  topic: string,
  vendor: string,
  runtime: string,
): ScaffoldData {
  const skillPath = vendor ? `${topic}/${vendor}/${name}` : `${topic}/${name}`;
  return {
    name,
    topic,
    vendor,
    runtime,
    description: `A ${typeName} for ${name}`,
    version: '0.1.0',
    packageName: `@agentx/${typeName}-${topic}-${name}`,
    skillPath,
    year: new Date().getFullYear(),
  };
}

function templateSetName(typeName: string, runtime: string): string {
  if (typeName === 'skill') return `skill-${runtime}`;
  return typeName;
}

function manifestFileName(typeName: string): string {
  if (['skill', 'workflow', 'prompt', 'persona', 'context', 'template'].includes(typeName)) {
    return `${typeName}.yaml`;
  }
  return 'manifest.yaml';
}

function getScaffoldsDir(): string {
  // At runtime, code runs from dist/ — scaffolds live at src/scaffolds/
  const thisFile = fileURLToPath(import.meta.url);
  const projectRoot = join(dirname(thisFile), '..');
  const candidate = join(projectRoot, 'src', 'scaffolds');
  if (existsSync(candidate)) return candidate;
  throw new Error('Scaffolds directory not found');
}

function renderTemplate(
  template: string,
  data: ScaffoldData,
): string {
  let result = template;
  for (const [key, value] of Object.entries(data)) {
    const pattern = new RegExp(`\\{\\{\\s*\\.${key.charAt(0).toUpperCase() + key.slice(1)}\\s*\\}\\}`, 'g');
    result = result.replace(pattern, String(value));
  }
  return result;
}

export function generate(
  typeName: string,
  data: ScaffoldData,
  outputDir: string,
): ScaffoldResult {
  const setName = templateSetName(typeName, data.runtime);
  const scaffoldsDir = getScaffoldsDir();
  const templateDir = join(scaffoldsDir, setName);

  if (!existsSync(templateDir)) {
    throw new Error(`Template set not found: ${setName}`);
  }

  // Prevent overwriting non-empty directories
  if (existsSync(outputDir)) {
    const entries = readdirSync(outputDir);
    if (entries.length > 0) {
      throw new Error(`Output directory is not empty: ${outputDir}`);
    }
  }

  mkdirSync(outputDir, { recursive: true });
  const files: string[] = [];

  for (const entry of readdirSync(templateDir)) {
    const srcPath = join(templateDir, entry);
    if (!statSync(srcPath).isFile()) continue;

    // Strip .tmpl extension
    const outName = entry.endsWith('.tmpl') ? entry.slice(0, -5) : entry;
    const content = readFileSync(srcPath, 'utf-8');

    // .hbs files are copied verbatim, others are rendered
    const rendered = entry.endsWith('.hbs.tmpl') ? content : renderTemplate(content, data);

    const outPath = join(outputDir, outName);
    writeFileSync(outPath, rendered, 'utf-8');
    files.push(outName);
  }

  return { outputDir, files, warnings: [] };
}
