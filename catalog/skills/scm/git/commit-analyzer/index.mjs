// commit-analyzer — AgentX Skill (Node)
// Analyzes git commit history for patterns and issues.
//
// This skill invokes the git CLI to collect commit data and produce
// a JSON summary. It follows the AgentX self-contained skill pattern
// (no AgentX runtime dependency).
//
// Usage: node index.mjs --repo-path=/path/to/repo [--since=30d]

import { execFileSync } from 'child_process';
import { writeFileSync, mkdirSync } from 'fs';
import { join } from 'path';
import { homedir } from 'os';
import { parseArgs } from 'util';

// Skill identity
const SKILL_TOPIC  = 'scm';
const SKILL_VENDOR = 'git';
const SKILL_NAME   = 'commit-analyzer';
const SKILL_PATH   = `${SKILL_TOPIC}/${SKILL_VENDOR}/${SKILL_NAME}`;

// Resolve userdata root
const USERDATA = process.env.AGENTX_USERDATA
  || join(homedir(), '.agentx', 'userdata');

// Registry paths
const registry = {
  root:   join(USERDATA, 'skills', SKILL_PATH),
  config: join(USERDATA, 'skills', SKILL_PATH, 'config.yaml'),
  output: join(USERDATA, 'skills', SKILL_PATH, 'output'),
};

// Parse CLI arguments
const { values } = parseArgs({
  options: {
    'repo-path': { type: 'string' },
    'since':     { type: 'string', default: '30d' },
  },
  strict: false,
});

const repoPath = values['repo-path'] || process.cwd();
const since    = values['since'] || '30d';

// Save output helper
function saveOutput(data) {
  mkdirSync(registry.output, { recursive: true });
  const payload = JSON.stringify(data, null, 2);
  writeFileSync(join(registry.output, 'latest.json'), payload);
}

// Convert a period like "30d" to a git --since argument value
function parseSincePeriod(period) {
  const match = period.match(/^(\d+)d$/);
  if (match) {
    return `${match[1]} days ago`;
  }
  return period;
}

// Main analysis
function analyzeCommits(repoPath, since) {
  const sinceValue = parseSincePeriod(since);
  const format = '{"hash":"%H","author":"%an","date":"%aI","subject":"%s"}';

  const args = [
    'log',
    `--since=${sinceValue}`,
    `--format=${format}`,
    '--no-merges',
  ];

  let raw;
  try {
    raw = execFileSync('git', args, {
      cwd: repoPath,
      encoding: 'utf8',
      maxBuffer: 10 * 1024 * 1024,
    });
  } catch (err) {
    return { error: `Failed to run git log: ${err.message}`, commits: [] };
  }

  const commits = raw
    .trim()
    .split('\n')
    .filter(Boolean)
    .slice(0, 500)
    .map((line) => {
      try { return JSON.parse(line); }
      catch { return null; }
    })
    .filter(Boolean);

  // Basic pattern analysis
  const authors = {};
  const daily = {};

  for (const c of commits) {
    authors[c.author] = (authors[c.author] || 0) + 1;
    const day = c.date.slice(0, 10);
    daily[day] = (daily[day] || 0) + 1;
  }

  return {
    total_commits: commits.length,
    period: since,
    top_authors: Object.entries(authors)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 10)
      .map(([name, count]) => ({ name, count })),
    daily_activity: Object.entries(daily)
      .sort((a, b) => a[0].localeCompare(b[0]))
      .map(([date, count]) => ({ date, count })),
    recent_commits: commits.slice(0, 10),
  };
}

// Run
const result = {
  timestamp: new Date().toISOString(),
  skill: SKILL_NAME,
  status: 'ok',
  data: analyzeCommits(repoPath, since),
};

saveOutput(result);
console.log(JSON.stringify(result, null, 2));
