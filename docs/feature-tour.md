# AgentX Feature Tour

A hands-on walkthrough of every feature in the AgentX CLI. Each section shows what the feature does, the commands involved, and example usage.

---

## Table of Contents

1. [Project Initialization](#1-project-initialization)
2. [Type System and Catalog](#2-type-system-and-catalog)
3. [Searching the Catalog](#3-searching-the-catalog)
4. [Installing Types](#4-installing-types)
5. [Listing Installed Types](#5-listing-installed-types)
6. [Linking Types to a Project](#6-linking-types-to-a-project)
7. [AI Tool Config Generation](#7-ai-tool-config-generation)
8. [Running Skills and Workflows](#8-running-skills-and-workflows)
9. [Prompt Composition](#9-prompt-composition)
10. [Scaffolding New Types](#10-scaffolding-new-types)
11. [Extension System](#11-extension-system)
12. [Secret and Environment Management](#12-secret-and-environment-management)
13. [Configuration Profiles](#13-configuration-profiles)
14. [User Configuration](#14-user-configuration)
15. [Health Checks and Diagnostics](#15-health-checks-and-diagnostics)
16. [Self-Update](#16-self-update)
17. [Enterprise Distribution](#17-enterprise-distribution)

---

## 1. Project Initialization

AgentX has two levels of initialization: global (user-level) and project-level.

### Global Init

Creates the `~/.agentx/` directory structure that holds installed types, user data, secrets, profiles, and skill registries.

```bash
agentx init --global
```

This creates:

```
~/.agentx/
  config.yaml              # user settings
  installed/               # installed types live here
  userdata/
    env/                   # shared .env files (default.env, aws.env, ...)
    profiles/              # named config profiles
    preferences.yaml       # user-wide defaults
    skills/                # per-skill registries (tokens, config, state, output)
```

### Project Init

Creates a project-level `.agentx/project.yaml` that declares which AI tools to generate configs for.

```bash
cd my-project
agentx init --tools claude-code,copilot,augment,opencode
```

The `--tools` flag accepts a comma-separated list. Supported values: `claude-code`, `copilot`, `augment`, `opencode`. The default is all four.

**What you get:** A `.agentx/project.yaml` file in your project root, ready for linking types.

---

## 2. Type System and Catalog

AgentX manages six composable types that follow a strict one-directional dependency graph:

```
context     <- foundation, no dependencies (knowledge, docs, patterns)
persona     <- references context (agent identity, expertise, tone)
skill       <- atomic, no type dependencies (wraps one CLI/API)
workflow    <- references skills (multi-step orchestration)
prompt      <- references all of the above (composed agent instructions)

template    <- standalone (distributable starting points)
```

### The Catalog

The built-in catalog ships with sample types across these categories:

| Type | Examples |
|------|----------|
| **context** | `context/spring-boot/error-handling`, `context/spring-boot/security` |
| **persona** | `personas/senior-java-dev` |
| **skill** | `skills/scm/git/commit-analyzer` (Node, wraps git CLI), `skills/ai/token-counter` (Go, self-contained), `skills/cloud/aws/ssm-lookup` (Node, wraps aws CLI) |
| **workflow** | `workflows/code-review`, `workflows/observability/splunk/create-query` |
| **prompt** | `prompts/java-pr-review` |
| **template** | `templates/skill-readme` |

### Key Rule: One Skill = One CLI/API

Every skill wraps exactly one external dependency. If your skill needs git AND jq, it should be a workflow that chains two separate skills. This keeps skills composable and their dependency requirements predictable.

- **Wraps an external CLI** (git, mvn, aws, kubectl) -> use a **Node ESM** skill
- **Self-contained, no external dependency** -> use a **Go** skill

### Manifest Files

Every type has a `manifest.yaml` that declares its metadata, dependencies, inputs, and outputs. Here is an example skill manifest:

```yaml
name: commit-analyzer
type: skill
version: "1.0.0"
description: Analyzes git commit history for patterns and issues
tags: [git, scm, analysis]
runtime: node
topic: scm
vendor: git
cli_dependencies:
  - name: git
    min_version: "2.0.0"
inputs:
  - name: repo-path
    type: string
    required: true
    description: Path to the git repository
  - name: since
    type: string
    required: false
    default: "30d"
    description: Time period to analyze
outputs:
  format: json
registry:
  tokens: []
  config:
    max_commits: 500
    include_merges: false
  state: []
```

And here is a prompt manifest that composes multiple types:

```yaml
name: java-pr-review
type: prompt
version: "1.1.0"
description: Java PR review prompt combining persona, context, skills, and workflows
tags: [java, code-review, pr]
persona: personas/senior-java-dev
context:
  - context/spring-boot/error-handling
  - context/spring-boot/security
skills:
  - skills/scm/git/commit-analyzer
  - skills/ai/token-counter
workflows:
  - workflows/code-review
template: prompt.hbs
```

---

## 3. Searching the Catalog

Search for types across all sources (catalog, extensions, installed).

```bash
# Free-text search (matches name, description, type path)
agentx search git

# Filter by type category
agentx search --type skill

# Filter by topic (scm, cicd, cloud, ai, observability, ...)
agentx search --topic cloud

# Filter by vendor
agentx search --vendor aws

# Filter by CLI dependency
agentx search --cli git

# Filter by tags (comma-separated, matches any)
agentx search --tag java,spring-boot

# Combine filters (all filters are AND-combined)
agentx search --type skill --topic cloud --vendor aws

# JSON output for scripting
agentx search --type skill --json
```

Output is a table with TYPE, NAME, VERSION, and DESCRIPTION columns. Results come from an mtime-based cache at `~/.agentx/registry-cache.json` that automatically invalidates when source directories change.

---

## 4. Installing Types

Install a type and all its dependencies to `~/.agentx/installed/`.

```bash
# Install a prompt (pulls in persona, context, skills, workflows)
agentx install prompts/java-pr-review
```

AgentX shows an install plan before proceeding:

```
Install plan:
  context: spring-boot/error-handling
  context: spring-boot/security
  persona: senior-java-dev
  skill:   scm/git/commit-analyzer
  skill:   ai/token-counter
  workflow: code-review
  prompt:  java-pr-review

? Proceed with installation? (Y/n)
```

### Flags

```bash
# Skip dependency resolution (install only the specified type)
agentx install skills/ai/token-counter --no-deps

# Skip confirmation prompt
agentx install prompts/java-pr-review -y
```

### What Happens During Install

1. **Dependency resolution** -- walks the manifest dependency tree, deduplicates
2. **Copy files** -- copies type directories from source to `~/.agentx/installed/`
3. **npm install** -- for Node skills/workflows, runs `npm install` in the installed directory
4. **Registry initialization** -- for skills, creates the userdata registry directory with `tokens.env`, `config.yaml`, `state/`, and `output/` scaffolding

If any skills require tokens (API keys, secrets), you'll see a warning:

```
  3 skills need token configuration.
  Run `agentx doctor --check-registry` to see what's missing.
```

---

## 5. Listing Installed Types

See everything installed in `~/.agentx/installed/`.

```bash
# List all installed types
agentx list

# Filter by type
agentx list --type skill
agentx list --type workflow

# JSON output
agentx list --json
```

Output:

```
TYPE       NAME               VERSION
context    error-handling      1.0.0
context    security            1.0.0
persona    senior-java-dev     1.0.0
skill      commit-analyzer     1.0.0
skill      token-counter       1.0.0
workflow   code-review         1.0.0
prompt     java-pr-review      1.1.0
```

---

## 6. Linking Types to a Project

Linking declares which installed types are active in a specific project and generates AI tool configurations.

```bash
cd my-project

# Link individual types
agentx link add personas/senior-java-dev
agentx link add skills/scm/git/commit-analyzer
agentx link add context/spring-boot/security

# Remove a link
agentx link remove context/spring-boot/security

# Regenerate all AI tool config files
agentx link sync

# Check link status
agentx link status
```

### Link Status Output

```
  [OK] claude-code: up-to-date (.claude/CLAUDE.md)
       Symlinks: 3/3 valid
  [OK] copilot:     up-to-date (.github/copilot-instructions.md)
       Symlinks: 2/2 valid
  [!!] augment:     stale (.augment/augment-guidelines.md)
       Symlinks: 1/2 valid
  [OK] opencode:    up-to-date (AGENTS.md)
       Symlinks: 3/3 valid
```

Status values: `up-to-date`, `stale` (needs re-sync), `not-generated` (no config yet).

---

## 7. AI Tool Config Generation

When you run `agentx link sync`, AgentX calls per-tool Node.js generators that produce native configuration files.

### Claude Code

Generates:
- `.claude/CLAUDE.md` -- persona instructions and available skills
- `.claude/commands/` -- skill and workflow wrappers as slash commands
- `.claude/context/` -- symlinks to installed context documents

### GitHub Copilot

Generates:
- `.github/copilot-instructions.md` -- persona instructions with context references
- `.github/copilot-context/` -- symlinks to installed context documents

### Augment Code

Generates:
- `.augment/augment-guidelines.md` -- persona instructions with context references
- `.augment/context/` -- symlinks to installed context documents

### OpenCode

Generates:
- `AGENTS.md` -- persona instructions and available skills (lives in project root, not inside `.opencode/`)
- `.opencode/commands/` -- skill and workflow wrappers as commands (with YAML frontmatter for description, agent, model)
- `.opencode/context/` -- symlinks to installed context documents

After sync, your AI tools pick up the new configurations automatically through their native discovery mechanisms. No AgentX runtime injection required at AI-tool-use time.

---

## 8. Running Skills and Workflows

Execute installed skills or workflows directly from the CLI.

### Running a Skill

```bash
# Run a Node skill (wraps git CLI)
agentx run skills/scm/git/commit-analyzer \
  --input repo-path=. \
  --input since=7d

# Run a Go skill (self-contained)
agentx run skills/ai/token-counter \
  --input text="Hello world, this is a test." \
  --input model=gpt-4
```

AgentX validates required inputs, ensures the skill's userdata registry exists, then dispatches to the appropriate runtime (Node or Go).

### Running a Workflow

```bash
agentx run workflows/code-review \
  --input repo-path=. \
  --input branch=feature/auth
```

Workflows execute each step in sequence. Each step invokes a skill:

```
Running workflow code-review (3 steps)...

--- Step 1/3: analyze-commits (skill: scm/git/commit-analyzer) ---
--- Step 1/3 complete ---

--- Step 2/3: check-patterns (skill: scm/git/commit-analyzer) ---
--- Step 2/3 complete ---

--- Step 3/3: generate-report (skill: ai/token-counter) ---
--- Step 3/3 complete ---

Workflow code-review completed successfully.
```

### Input Handling

Inputs are passed as `--input key=value` flags (repeatable). Required inputs are validated; optional inputs use their declared defaults.

```bash
agentx run skills/cloud/aws/ssm-lookup \
  --input param-name=/app/db-host \
  --input decrypt=true
```

---

## 9. Prompt Composition

Compose a full agent prompt by resolving all referenced types (persona, context, skills, workflows) into a single markdown document.

### From a Prompt Type

```bash
# Print to stdout
agentx prompt prompts/java-pr-review

# Copy to clipboard
agentx prompt prompts/java-pr-review --copy

# Write to file
agentx prompt prompts/java-pr-review -o review-prompt.md
```

The composed output includes persona instructions, context documents, skill references, and workflow descriptions -- all stitched together via the prompt's Handlebars template (`prompt.hbs`).

### Interactive Mode

When called with no arguments, AgentX walks you through an interactive prompt builder:

```bash
agentx prompt
```

Interactive flow:
1. **Select persona** -- numbered list of installed personas
2. **Select topic** -- topics derived from installed skills
3. **Enter intent** -- free-text description (e.g., "code-review", "migration")

AgentX then auto-discovers context and skills matching the selected topic and composes a prompt on the fly.

```
Select persona:
  1) senior-java-dev
Enter number [1-1]: 1

Select topic:
  1) scm
  2) ai
  3) cloud
Enter number [1-3]: 1

Enter intent (e.g., code-review, migration, incident-triage): code-review
```

---

## 10. Scaffolding New Types

Generate boilerplate for any type with `agentx create`.

### Create a Skill

```bash
# Node skill (wraps an external CLI)
agentx create skill my-tool --topic cloud --vendor aws --runtime node

# Go skill (self-contained)
agentx create skill token-counter --topic ai --runtime go
```

Generated files for a Node skill:
```
my-tool/
  manifest.yaml    # skill manifest with inputs, outputs, registry
  index.mjs        # Node ESM entry point with registry pattern
  package.json     # npm package with @agentx/shared-node dependency
  Makefile         # build, test, clean targets
```

Generated files for a Go skill:
```
token-counter/
  manifest.yaml    # skill manifest
  main.go          # Go entry point with flag parsing and registry output
  go.mod           # Go module
  Makefile         # build, test, clean targets
```

### Create Other Types

```bash
agentx create workflow my-flow
agentx create prompt my-prompt
agentx create persona senior-devops
agentx create context k8s-patterns
agentx create template migration-report
```

Each includes next-step guidance:

```
Created skill at ./my-tool/
  manifest.yaml
  index.mjs
  package.json
  Makefile

Next steps:
  1. Edit index.mjs to add your skill logic
  2. Run 'npm install' to install dependencies
  3. Test with 'agentx run skills/cloud/aws/my-tool'
```

### Flags

```bash
# Specify output directory (default: ./<name>)
agentx create skill my-tool --output-dir catalog/skills/cloud/aws/my-tool
```

---

## 11. Extension System

Extensions let organizations bring proprietary types (context, personas, skills) as git submodules without modifying the core catalog.

### Add an Extension

```bash
agentx extension add acme-corp git@github.com:acme/agentx-types.git
agentx extension add my-types https://github.com/me/types.git --branch develop
```

This registers the extension in `project.yaml` and adds it as a git submodule under `extensions/`.

### Manage Extensions

```bash
# List all extensions with their status
agentx extension list

# Sync (git submodule update --init --recursive)
agentx extension sync

# Remove an extension
agentx extension remove acme-corp
```

### Extension List Output

```
NAME         PATH                    BRANCH   STATUS
acme-corp    extensions/acme-corp    main     ok
my-types     extensions/my-types     develop  uninitialized
```

Status values: `ok`, `uninitialized`, `modified`, `missing`.

### Resolution Order

Extensions are searched in the order declared in `project.yaml`. Later entries have higher priority and override core types with the same name:

```yaml
# project.yaml
extensions:
  - name: acme-corp
    path: extensions/acme-corp
    source: git@github.com:acme/agentx-knowledge.git
    branch: main

resolution:
  - core          # built-in catalog
  - acme-corp     # higher priority, overrides core on name conflicts
```

This means an organization can override `personas/senior-java-dev` with their own version in an extension, and all installs will use the override.

The `ext` alias works for all extension commands: `agentx ext list`, `agentx ext add`, etc.

---

## 12. Secret and Environment Management

AgentX manages environment variables and secrets through `.env` files at two levels: shared (vendor-wide) and skill-specific.

### List All Env Files

```bash
agentx env list
```

Output:

```
Shared:
  env/default.env
  env/aws.env
Skill-specific:
  skills/cloud/aws/ssm-lookup/tokens.env
  skills/scm/git/commit-analyzer/tokens.env
```

### Edit an Env File

Opens the file in your `$EDITOR` (defaults to `vi` on Unix, `notepad` on Windows). Creates the file with a template if it doesn't exist.

```bash
# Shared vendor env file
agentx env edit aws

# Skill-specific tokens
agentx env edit cloud/aws/ssm-lookup
```

### View an Env File

Values are redacted by default for safety.

```bash
agentx env show aws
```

Output:

```
# /Users/you/.agentx/userdata/env/aws.env
AWS_ACCESS_KEY_ID=AKIA****
AWS_SECRET_ACCESS_KEY=****
AWS_DEFAULT_REGION=us-east-1
```

```bash
# Show actual values
agentx env show aws --no-redact
```

### Resolution Order

When a skill runs, environment variables load in this order (later overrides earlier):

1. `env/default.env` -- global defaults
2. `env/<vendor>.env` -- vendor-specific (e.g., `env/aws.env`)
3. `skills/<path>/tokens.env` -- skill-specific (highest priority)

Use `agentx doctor --trace-env <skill>` to see the full resolution chain for a specific skill.

---

## 13. Configuration Profiles

Profiles let you switch between environments (work vs. personal, staging vs. production) with a single command.

```bash
# List all profiles
agentx profile list

# Output:
#   work (active)
#   personal
#   staging

# Switch active profile
agentx profile use personal

# View active profile
agentx profile show

# Output:
# Profile: personal
# ---
#   name:           personal
#   github_org:     my-github
#   default_branch: main
```

Profiles are stored as YAML files in `~/.agentx/userdata/profiles/`. The active profile is tracked via a symlink.

```bash
# Output as JSON or YAML
agentx profile show --json
agentx profile show --yaml
```

Profile fields include: `name`, `aws_profile`, `aws_region`, `github_org`, `splunk_host`, `default_branch`, plus arbitrary extras.

---

## 14. User Configuration

Persistent settings stored in `~/.agentx/config.yaml`.

```bash
# Set a value
agentx config set mirror https://nexus.corp.com/repository/agentx-releases

# Get a value
agentx config get mirror

# Common settings:
agentx config set editor vim
agentx config set default_profile work
```

The `mirror` setting is used by `agentx update` and `scripts/install.sh` to download binaries from an internal mirror instead of GitHub.

---

## 15. Health Checks and Diagnostics

The `doctor` command diagnoses problems with your AgentX installation.

### Run All Checks

```bash
agentx doctor
```

Runs runtime, extensions, userdata, CLI dependency, registry, and link checks in sequence.

### Individual Checks

```bash
# Verify CLI dependencies (git, aws, mvn, etc.) for all installed skills
agentx doctor --check-cli

# Verify Node and Go are available
agentx doctor --check-runtime

# Verify symlinks from link sync are intact
agentx doctor --check-links

# Verify extension submodules are initialized
agentx doctor --check-extensions

# Verify userdata directory exists with correct permissions
agentx doctor --check-userdata

# Validate skill registries (tokens.env, config.yaml) against manifest declarations
agentx doctor --check-registry

# Validate a specific manifest file against JSON Schema
agentx doctor --check-manifest catalog/skills/ai/token-counter/manifest.yaml
```

### Interactive Fix

```bash
agentx doctor --fix
```

Walks through issues and offers to fix them: creating missing registry directories, initializing missing `tokens.env` files, and scaffolding `config.yaml` with defaults.

### Environment Trace

Debug the full environment resolution chain for any skill:

```bash
agentx doctor --trace-env cloud/aws/ssm-lookup
```

Shows exactly which `.env` files load, in what order, and what values each contributes.

### Example Output

```
Runtime check:
  [ OK ] go found at /usr/local/bin/go
  [ OK ] node found at /usr/local/bin/node
  [ OK ] git found at /usr/bin/git
Extensions check:
  [ OK ] project.yaml is valid
  [ OK ] acme-corp: clean
CLI dependency check:
  [ OK ] git >= 2.0.0 (found 2.43.0)
  [MISS] aws >= 2.0.0 (not found)
Registry check:
  [ OK ] scm/git/commit-analyzer: tokens.env present, config.yaml valid
  [WARN] cloud/aws/ssm-lookup: SSM_ROLE_ARN not set in tokens.env
```

---

## 16. Self-Update

Update the AgentX binary in place.

```bash
# Update to latest version
agentx update

# Check for updates without installing
agentx update --check

# Install a specific version
agentx update --version 1.2.0

# Force reinstall current version
agentx update --force
```

The update process:
1. Fetches the latest release from GitHub (or a configured mirror)
2. Downloads the binary for your OS/architecture
3. Verifies the checksum
4. Extracts and replaces the current binary

### Mirror Support

For enterprises behind firewalls:

```bash
# Set via config
agentx config set mirror https://nexus.corp.com/repository/agentx-releases

# Or via environment variable
export AGENTX_MIRROR=https://nexus.corp.com/repository/agentx-releases
agentx update
```

### Update Banner

AgentX checks for updates in the background on every command invocation (using a cached version check). If an update is available, you'll see a non-blocking banner:

```
A new version of agentx is available: 1.1.0 -> 1.2.0
Run `agentx update` to upgrade.
```

The alias `agentx self-update` also works.

---

## 17. Enterprise Distribution

AgentX supports distribution through Sonatype Nexus for organizations that cannot use public GitHub releases.

### CI Pipeline

The `release-nexus.yaml` workflow triggers automatically after a CLI release and:
1. Downloads all release artifacts from GitHub
2. Uploads binaries + checksums to a Nexus raw repository
3. Publishes `@agentx/*` npm packages to a Nexus npm registry
4. Uploads `scripts/install.sh` and a `version.txt` marker to Nexus

### Install from Nexus

```bash
AGENTX_MIRROR=https://nexus.corp.com/repository/agentx-releases \
  curl -sSL https://nexus.corp.com/repository/agentx-releases/install.sh | bash
```

When `AGENTX_MIRROR` is set, the install script resolves the latest version from `${MIRROR}/latest/version.txt` instead of the GitHub API.

See [docs/enterprise-setup.md](enterprise-setup.md) for the full Nexus configuration guide, internal Homebrew tap setup, and mirror configuration.

---

## Version Information

```bash
# Full version info
agentx version
# Output: agentx version 1.0.0 (commit: abc1234, built: 2025-01-15)

# Version number only
agentx version --short

# JSON format
agentx version --json
```

---

## Quick Reference

```bash
# Setup
agentx init --global                    # one-time user setup
agentx init --tools claude-code,copilot,augment,opencode # per-project setup

# Discovery
agentx search git                       # find types
agentx search --type skill --topic cloud --vendor aws

# Install & manage
agentx install prompts/java-pr-review   # install with dependencies
agentx list --type skill                # see what's installed
agentx uninstall skills/ai/token-counter

# Project linking
agentx link add personas/senior-java-dev
agentx link sync                        # generate AI tool configs
agentx link status                      # check link health

# Execute
agentx run skills/scm/git/commit-analyzer --input repo-path=.
agentx prompt prompts/java-pr-review --copy

# Author
agentx create skill my-tool --topic cloud --runtime node

# Maintain
agentx doctor                           # full health check
agentx doctor --fix                     # interactive repair
agentx doctor --trace-env cloud/aws/ssm # debug env resolution
agentx update                           # self-update

# Secrets & config
agentx env edit aws                     # edit shared secrets
agentx env show cloud/aws/ssm           # view skill tokens
agentx profile use work                 # switch profiles
agentx config set mirror https://...    # set mirror URL

# Extensions
agentx extension add acme git@...       # add knowledge base
agentx extension sync                   # update submodules
```