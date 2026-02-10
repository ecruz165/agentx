// code-review — AgentX Workflow
// Composes the commit-analyzer skill into an automated code review pipeline.

import { readFileSync, existsSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';
import { parseArgs } from 'util';

const USERDATA = process.env.AGENTX_USERDATA
  || join(homedir(), '.agentx', 'userdata');

// Parse CLI arguments
const { values } = parseArgs({
  options: {
    'repo-path': { type: 'string' },
  },
  strict: false,
});

const repoPath = values['repo-path'] || process.cwd();

// Step definitions
const steps = [
  {
    id: 'analyze-commits',
    skill: 'skills/scm/git/commit-analyzer',
    run: async () => {
      // Read the commit-analyzer skill's latest output from the registry
      const outputPath = join(
        USERDATA, 'skills', 'scm', 'git', 'commit-analyzer', 'output', 'latest.json'
      );
      if (existsSync(outputPath)) {
        return JSON.parse(readFileSync(outputPath, 'utf8'));
      }
      return { status: 'pending', message: 'Run commit-analyzer skill first' };
    },
  },
];

// Workflow runner
async function run() {
  const results = {};
  for (const step of steps) {
    console.log(`Running step: ${step.id} (${step.skill})`);
    results[step.id] = await step.run();
  }
  console.log(JSON.stringify(results, null, 2));
}

run().catch((err) => {
  console.error(err);
  process.exit(1);
});
