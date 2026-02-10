// create-query -- AgentX Workflow (Node)
// Multi-step workflow that analyzes logs and creates reusable Splunk SPL queries.
//
// Demonstrates:
//   - Multi-step orchestration (4 steps)
//   - Cross-skill output consumption (reads from registry output paths)
//   - Graduated template pattern (save/load reusable .spl templates)
//   - Template reuse via --template flag
//
// Usage:
//   node index.mjs --log-pattern="error 500 in payment service"
//   node index.mjs --template=payment-errors
//   node index.mjs --log-pattern="timeout in auth" --save-as=auth-timeouts

import { readFileSync, writeFileSync, mkdirSync, existsSync, readdirSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';
import { parseArgs } from 'util';

const WORKFLOW_NAME = 'create-query';
const WORKFLOW_PATH = 'observability/splunk/create-query';

// Resolve userdata root
const USERDATA = process.env.AGENTX_USERDATA
  || join(homedir(), '.agentx', 'userdata');

// Registry paths for this workflow
const registry = {
  root:      join(USERDATA, 'workflows', WORKFLOW_PATH),
  output:    join(USERDATA, 'workflows', WORKFLOW_PATH, 'output'),
  templates: join(USERDATA, 'workflows', WORKFLOW_PATH, 'templates'),
  state:     join(USERDATA, 'workflows', WORKFLOW_PATH, 'state'),
};

// Parse CLI arguments
const { values } = parseArgs({
  options: {
    'log-pattern': { type: 'string' },
    'template':    { type: 'string' },
    'index':       { type: 'string', default: 'main' },
    'save-as':     { type: 'string' },
  },
  strict: false,
});

const logPattern   = values['log-pattern'];
const templateName = values['template'];
const splunkIndex  = values['index'] || 'main';
const saveAs       = values['save-as'];

if (!logPattern && !templateName) {
  console.error('error: --log-pattern or --template is required');
  process.exit(1);
}

// Save output helper
function saveOutput(data) {
  mkdirSync(registry.output, { recursive: true });
  const payload = JSON.stringify(data, null, 2);
  writeFileSync(join(registry.output, 'latest.json'), payload);
}

// Template management -- graduated template pattern
function loadTemplate(name) {
  const templatePath = join(registry.templates, `${name}.spl`);
  if (existsSync(templatePath)) {
    return readFileSync(templatePath, 'utf8');
  }
  return null;
}

function saveTemplate(name, spl) {
  mkdirSync(registry.templates, { recursive: true });
  writeFileSync(join(registry.templates, `${name}.spl`), spl);
}

function listTemplates() {
  if (!existsSync(registry.templates)) return [];
  return readdirSync(registry.templates)
    .filter((f) => f.endsWith('.spl'))
    .map((f) => f.replace('.spl', ''));
}

// Read another skill's output from the registry
function readSkillOutput(skillPath) {
  const outputPath = join(USERDATA, 'skills', skillPath, 'output', 'latest.json');
  if (existsSync(outputPath)) {
    try {
      return JSON.parse(readFileSync(outputPath, 'utf8'));
    } catch {
      return null;
    }
  }
  return null;
}

// Step 1: Analyze logs -- parse the log pattern description
function analyzeLogs(logPattern) {
  console.log(`Step 1/4: Analyzing log pattern: "${logPattern}"`);

  // Extract key terms from the pattern description
  const keywords = logPattern.toLowerCase().split(/\s+/).filter((w) => w.length > 2);
  const hasError = /error|fail|exception|crash/i.test(logPattern);
  const hasTimeout = /timeout|slow|latency|delay/i.test(logPattern);
  const hasHTTP = /\b[45]\d{2}\b|http|request|response/i.test(logPattern);

  // Try to read related skill outputs for enrichment
  const commitData = readSkillOutput('scm/git/commit-analyzer');

  return {
    step: 'analyze-logs',
    status: 'ok',
    analysis: {
      keywords,
      patterns: {
        error: hasError,
        timeout: hasTimeout,
        http: hasHTTP,
      },
      enrichment: {
        recent_commits: commitData ? 'available' : 'not available',
      },
    },
  };
}

// Step 2: Create Splunk query -- generate SPL from analysis or load template
function createSplunkQuery(analysis, templateName, index) {
  console.log('Step 2/4: Creating Splunk query');

  // If a template is specified, load it
  if (templateName) {
    const spl = loadTemplate(templateName);
    if (spl) {
      console.log(`  Loaded template: ${templateName}`);
      return {
        step: 'create-splunk-query',
        status: 'ok',
        source: 'template',
        template_name: templateName,
        spl,
      };
    }
    console.log(`  Template "${templateName}" not found, generating new query`);
  }

  // Generate SPL from the analysis
  const { keywords, patterns } = analysis.analysis;
  const parts = [`index=${index}`];

  if (patterns.error) {
    parts.push('(level=ERROR OR level=FATAL)');
  }
  if (patterns.timeout) {
    parts.push('(timeout OR "connection timed out" OR "read timed out")');
  }
  if (patterns.http) {
    parts.push('(status>=400)');
  }

  // Add keyword search terms
  const searchTerms = keywords
    .filter((k) => !['error', 'fail', 'timeout', 'the', 'and', 'for', 'from'].includes(k))
    .slice(0, 5);
  if (searchTerms.length > 0) {
    parts.push(`(${searchTerms.map((t) => `"${t}"`).join(' OR ')})`);
  }

  // Build the full SPL query
  const searchClause = parts.join(' ');
  const spl = [
    searchClause,
    '| stats count by source, host, level',
    '| sort -count',
    '| head 100',
  ].join('\n');

  return {
    step: 'create-splunk-query',
    status: 'ok',
    source: 'generated',
    spl,
  };
}

// Step 3: Search Splunk -- execute the query (simulated)
function searchSplunk(queryResult) {
  console.log('Step 3/4: Executing Splunk search');

  // In a real implementation, this would call the Splunk REST API
  // or invoke the splunk CLI. For this sample, we simulate the result.
  return {
    step: 'search-splunk',
    status: 'ok',
    note: 'Simulated result -- connect to Splunk REST API for real execution',
    query: queryResult.spl,
    result: {
      total_events: 0,
      message: 'Connect splunk skill or configure SPLUNK_HOST to execute queries',
    },
  };
}

// Step 4: Save query template -- graduated template pattern
function saveQueryTemplate(queryResult, saveAs) {
  console.log('Step 4/4: Template management');

  const saved = [];
  if (saveAs && queryResult.spl) {
    saveTemplate(saveAs, queryResult.spl);
    saved.push(saveAs);
    console.log(`  Saved template: ${saveAs}.spl`);
  }

  const available = listTemplates();

  return {
    step: 'save-query-template',
    status: 'ok',
    saved_templates: saved,
    available_templates: available,
  };
}

// Workflow runner
async function run() {
  const stepResults = {};

  // Step 1: Analyze logs
  if (logPattern) {
    stepResults['analyze-logs'] = analyzeLogs(logPattern);
  }

  // Step 2: Create query (from analysis or template)
  stepResults['create-splunk-query'] = createSplunkQuery(
    stepResults['analyze-logs'] || { analysis: { keywords: [], patterns: {} } },
    templateName,
    splunkIndex,
  );

  // Step 3: Search Splunk
  stepResults['search-splunk'] = searchSplunk(stepResults['create-splunk-query']);

  // Step 4: Save template if requested
  stepResults['save-query-template'] = saveQueryTemplate(
    stepResults['create-splunk-query'],
    saveAs,
  );

  // Build final output
  const result = {
    timestamp: new Date().toISOString(),
    workflow: WORKFLOW_NAME,
    status: 'ok',
    steps: stepResults,
  };

  saveOutput(result);
  console.log('\n--- Workflow Result ---');
  console.log(JSON.stringify(result, null, 2));
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
