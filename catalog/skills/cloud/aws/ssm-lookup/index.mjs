// ssm-lookup -- AgentX Skill (Node)
// Looks up AWS SSM Parameter Store values with caching.
//
// This skill wraps the AWS CLI to retrieve SSM parameters. It demonstrates
// the full AgentX registry pattern: tokens, config, state (caching), and output.
//
// Usage: node index.mjs --param-name=/app/config/db-host [--decrypt]

import { execFileSync } from 'child_process';
import { readFileSync, writeFileSync, mkdirSync, existsSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';
import { parseArgs } from 'util';

// Skill identity
const SKILL_TOPIC  = 'cloud';
const SKILL_VENDOR = 'aws';
const SKILL_NAME   = 'ssm-lookup';
const SKILL_PATH   = `${SKILL_TOPIC}/${SKILL_VENDOR}/${SKILL_NAME}`;

// Resolve userdata root
const USERDATA = process.env.AGENTX_USERDATA
  || join(homedir(), '.agentx', 'userdata');

// Registry paths
const registry = {
  root:    join(USERDATA, 'skills', SKILL_PATH),
  tokens:  join(USERDATA, 'skills', SKILL_PATH, 'tokens.env'),
  config:  join(USERDATA, 'skills', SKILL_PATH, 'config.yaml'),
  state:   join(USERDATA, 'skills', SKILL_PATH, 'state'),
  output:  join(USERDATA, 'skills', SKILL_PATH, 'output'),
};

// Parse CLI arguments
const { values } = parseArgs({
  options: {
    'param-name': { type: 'string' },
    'decrypt':    { type: 'boolean', default: true },
  },
  strict: false,
});

const paramName = values['param-name'];
const decrypt   = values['decrypt'] !== false;

if (!paramName) {
  console.error('error: --param-name is required');
  process.exit(1);
}

// Load tokens from tokens.env (simple KEY=VALUE parser)
function loadTokens() {
  const tokens = {};
  if (existsSync(registry.tokens)) {
    const lines = readFileSync(registry.tokens, 'utf8').split('\n');
    for (const line of lines) {
      const trimmed = line.trim();
      if (!trimmed || trimmed.startsWith('#')) continue;
      const eq = trimmed.indexOf('=');
      if (eq > 0) {
        tokens[trimmed.slice(0, eq).trim()] = trimmed.slice(eq + 1).trim();
      }
    }
  }
  return tokens;
}

// Load config defaults
function loadConfig() {
  const defaults = { cache_ttl: 300, max_results: 10 };
  if (existsSync(registry.config)) {
    try {
      const raw = readFileSync(registry.config, 'utf8');
      const parsed = {};
      for (const line of raw.split('\n')) {
        const trimmed = line.trim();
        if (!trimmed || trimmed.startsWith('#')) continue;
        const match = trimmed.match(/^(\w+):\s*(.+)$/);
        if (match) parsed[match[1]] = isNaN(match[2]) ? match[2] : Number(match[2]);
      }
      return { ...defaults, ...parsed };
    } catch {
      return defaults;
    }
  }
  return defaults;
}

// Cache management
function getCachePath() {
  return join(registry.state, 'cache.json');
}

function readCache() {
  const cachePath = getCachePath();
  if (existsSync(cachePath)) {
    try {
      return JSON.parse(readFileSync(cachePath, 'utf8'));
    } catch {
      return {};
    }
  }
  return {};
}

function writeCache(cache) {
  mkdirSync(registry.state, { recursive: true });
  writeFileSync(getCachePath(), JSON.stringify(cache, null, 2));
}

function getCachedValue(paramName, cacheTtl) {
  const cache = readCache();
  const entry = cache[paramName];
  if (entry) {
    const age = (Date.now() - entry.timestamp) / 1000;
    if (age < cacheTtl) {
      return entry.value;
    }
  }
  return null;
}

function setCachedValue(paramName, value) {
  const cache = readCache();
  cache[paramName] = { value, timestamp: Date.now() };
  writeCache(cache);
}

// Save output
function saveOutput(data) {
  mkdirSync(registry.output, { recursive: true });
  const payload = JSON.stringify(data, null, 2);
  writeFileSync(join(registry.output, 'latest.json'), payload);
}

// Call AWS CLI
function ssmGetParameter(paramName, decrypt, region) {
  const args = [
    'ssm', 'get-parameters',
    '--names', paramName,
    '--output', 'json',
  ];
  if (decrypt) {
    args.push('--with-decryption');
  }

  const env = { ...process.env };
  if (region) {
    env.AWS_DEFAULT_REGION = region;
  }

  const raw = execFileSync('aws', args, {
    encoding: 'utf8',
    env,
    timeout: 30000,
  });

  return JSON.parse(raw);
}

// Main
const tokens = loadTokens();
const config = loadConfig();

// Check for cached value
const cached = getCachedValue(paramName, config.cache_ttl);
if (cached) {
  const result = {
    timestamp: new Date().toISOString(),
    skill: SKILL_NAME,
    status: 'ok',
    source: 'cache',
    data: cached,
  };
  saveOutput(result);
  console.log(JSON.stringify(result, null, 2));
  process.exit(0);
}

// Look up the parameter via AWS CLI
const region = tokens.AWS_REGION || process.env.AWS_REGION || 'us-east-1';

let awsResult;
try {
  awsResult = ssmGetParameter(paramName, decrypt, region);
} catch (err) {
  const errorResult = {
    timestamp: new Date().toISOString(),
    skill: SKILL_NAME,
    status: 'error',
    error: `Failed to call aws ssm: ${err.message}`,
  };
  saveOutput(errorResult);
  console.log(JSON.stringify(errorResult, null, 2));
  process.exit(1);
}

// Process results
const parameters = awsResult.Parameters || [];
const invalidParameters = awsResult.InvalidParameters || [];

const data = {
  param_name: paramName,
  region,
  parameters: parameters.map((p) => ({
    name: p.Name,
    value: p.Value,
    type: p.Type,
    version: p.Version,
    last_modified: p.LastModifiedDate,
  })),
  invalid_parameters: invalidParameters,
};

// Cache the result
if (parameters.length > 0) {
  setCachedValue(paramName, data);
}

const result = {
  timestamp: new Date().toISOString(),
  skill: SKILL_NAME,
  status: 'ok',
  source: 'aws',
  data,
};

saveOutput(result);
console.log(JSON.stringify(result, null, 2));
