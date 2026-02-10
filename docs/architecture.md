# AgentX Architecture

This document covers the internal architecture of AgentX, including manifest schemas, the registry pattern, dependency resolution, and troubleshooting.

---

## Manifest Reference

Every type has a YAML manifest file at its root. The filename matches the type (`skill.yaml`, `persona.yaml`, etc.). All manifests share a set of common fields.

### Common Fields

```yaml
name: string                    # unique identifier (pattern: [a-z0-9][a-z0-9-]*)
type: skill | workflow | prompt | persona | context | template
version: string                 # semver (e.g., 1.0.0)
description: string             # human-readable description
tags: [string]                  # searchable tags
author: string                  # author name
```

### Context Manifest -- `context.yaml`

```yaml
name: spring-boot-error-handling
type: context
version: 1.0.0
description: Standard error handling patterns for Spring Boot services
tags: [spring-boot, java, error-handling]
author: edwin
format: markdown
tokens: 2400                     # estimated token count for AI consumption
sources:
  - patterns.md                  # referenced content files (relative paths)
  - examples.md
```

### Persona Manifest -- `persona.yaml`

```yaml
name: senior-java-dev
type: persona
version: 1.2.0
description: Senior Java developer with Spring Boot expertise
tags: [java, spring-boot, microservices]
author: edwin
expertise:
  - spring-boot
  - maven
tone: direct, pragmatic, opinionated
conventions:
  - prefers constructor injection over field injection
  - enforces test coverage for public methods
context:                         # references to context types
  - context/spring-boot/error-handling
  - context/spring-boot/testing
template: persona.md             # the persona instruction file
```

### Skill Manifest -- `skill.yaml`

```yaml
name: commit-analyzer
type: skill
version: 2.1.0
description: Analyzes git commit history for team metrics
tags: [git, analytics]
author: edwin
runtime: node                    # node | go
vendor: null                     # aws, github, harness, splunk, etc.
topic: scm                       # cloud, scm, cicd, java, observability, etc.
cli_dependencies:
  - name: git
    min_version: "2.30"
inputs:
  - name: repoPath
    type: string
    required: true
    description: Path to the git repository
  - name: days
    type: number
    default: 30
    description: Number of days to analyze
outputs:
  format: json
  schema: ./output-schema.json
registry:                        # declares what the skill expects in userdata
  tokens: []                     # env vars needed in tokens.env
  config:
    days_default: 30             # default values for config.yaml
    include_merge_commits: true
  state:
    - last-run.json              # files the skill may create in state/
  output:
    schema: ./output-schema.json # JSON schema for output/latest.json
```

### Workflow Manifest -- `workflow.yaml`

```yaml
name: deploy-verify
type: workflow
version: 1.0.0
description: Verifies deployment by checking commits, pipeline status, and config
tags: [deploy, verification, cicd]
author: edwin
runtime: node
steps:
  - id: analyze-commits
    skill: skills/scm/git/commit-analyzer
    inputs:
      repoPath: ${inputs.repoPath}
      days: 7
  - id: check-pipeline
    skill: skills/cicd/harness/deploy-status
    inputs:
      pipelineId: ${inputs.pipelineId}
  - id: verify-config
    skill: skills/cloud/aws/ssm-lookup
    inputs:
      paramName: ${steps.check-pipeline.outputs.envConfig}
inputs:
  - name: repoPath
    type: string
    required: true
  - name: pipelineId
    type: string
    required: true
outputs:
  format: json
```

### Prompt Manifest -- `prompt.yaml`

```yaml
name: java-pr-review
type: prompt
version: 1.0.0
description: Comprehensive Java PR review with dependency and pattern analysis
tags: [code-review, java, pr]
author: edwin
persona: personas/senior-java-dev
context:
  - context/spring-boot/error-handling
  - context/spring-boot/security
skills:
  - skills/scm/git/commit-analyzer
  - skills/java/maven/dependency-analyzer
workflows:
  - workflows/deploy-verify
template: prompt.hbs             # Handlebars template for rendering
```

### Template Manifest -- `template.yaml`

```yaml
name: error-spike
type: template
version: 1.0.0
description: Reusable SPL query for detecting error spikes
tags: [splunk, observability]
variables:
  - name: index
    default: main
  - name: threshold
    default: 100
```

---

## Registry Pattern

Each installed skill has a corresponding registry folder in `~/.agentx/userdata/skills/<skill-path>/`. This folder contains everything about that skill's user-specific state: secrets, configuration, cached state, output, and templates.

### Structure

```
~/.agentx/userdata/skills/<topic>/<vendor>/<name>/
  tokens.env          <- skill-specific secrets (chmod 600)
  config.yaml         <- skill-specific configuration
  state/              <- internal persisted state
    last-run.json
    cache.json
  output/             <- reusable output consumed by other skills
    latest.json       <- most recent run (always overwritten)
    2026-02-07T14-30.json  <- timestamped history
  templates/          <- graduated outputs saved for reuse
    param-report.hbs
```

### Shared Resources

In addition to the per-skill registry, skills also load shared resources:

```
~/.agentx/userdata/
  env/
    default.env        <- shared global env vars (loaded by all skills)
    aws.env            <- shared vendor-specific env vars
    github.env
    splunk.env
  profiles/
    work.yaml          <- named configuration profiles
    personal.yaml
    active -> work.yaml  <- symlink to active profile
  preferences.yaml     <- user-wide defaults (output_format, color, etc.)
```

### Environment Resolution Order

Skills load environment variables in this order. Later sources override earlier ones:

1. `env/default.env` -- shared global
2. `env/<vendor>.env` -- shared vendor-specific
3. `skills/<skill-path>/tokens.env` -- skill-specific (highest priority)

Skills resolve the userdata root via the environment variable:

```bash
AGENTX_USERDATA="${AGENTX_USERDATA:-$HOME/.agentx/userdata}"
```

This means skills work both inside and outside AgentX. No import of AgentX is required -- just read the known paths.

### The "Graduate Output" Pattern

Skills produce output to `output/latest.json`. When a workflow or skill produces something the user approves (a query, a report format), it can be saved as a reusable template in `templates/`. This is the key workflow: **produce -> review -> save -> reuse**.

### Cross-Skill Output Consumption

Skills consume each other's output through the `output/latest.json` convention at known paths. No import chain is needed. A workflow runs skill A, then skill B reads A's output from the well-known path.

---

## Dependency Resolution

### Type Resolution Across Sources

When `agentx` looks up a type, it searches sources in the resolution order defined in `project.yaml`:

1. Extensions (in declared order, later extensions have higher priority)
2. Core types (`catalog/`)

```bash
agentx install personas/acme-java-dev

# Resolution order:
# 1. extensions/another-team/personas/acme-java-dev/  (highest priority)
# 2. extensions/acme-corp/personas/acme-java-dev/
# 3. catalog/personas/acme-java-dev/                  (core)
```

### Source Discovery

The CLI discovers type sources in this order:

1. If `AGENTX_HOME` is set, use `$AGENTX_HOME/catalog/` as the core source and scan `$AGENTX_HOME/extensions/` for extension sources
2. Otherwise, look for `catalog/` relative to the executable

### Dependency Tree Resolution

When installing a type with dependencies (e.g., a prompt that references personas, context, skills, and workflows), AgentX:

1. Reads the manifest and discovers all references
2. Resolves each reference through the source resolution order
3. Walks references recursively, building a dependency tree
4. Deduplicates types that appear multiple times in the tree
5. Checks which types are already installed and skips them
6. Presents the install plan to the user for confirmation

### Installation Flow

```
agentx install prompts/code-review/java-pr-review

1. Parse prompt.yaml -> discover persona, context, skills, workflows
2. Resolve each reference -> search sources in order
3. Build dependency tree -> walk recursively, deduplicate
4. Show install plan + prompt for confirmation
5. Copy types to ~/.agentx/installed/ in dependency order
6. Install Node dependencies for skills/workflows (npm install)
7. Initialize skill registries in ~/.agentx/userdata/skills/
8. Report: summary, missing CLI deps, required tokens
```

---

## Userdata Directory Structure

The full userdata hierarchy:

```
~/.agentx/
  config.yaml                    <- user-level settings (mirror URL, etc.)
  bin/                           <- compiled Go skill binaries
  installed/                     <- installed types
    context/
    personas/
    prompts/
    skills/
    workflows/
    templates/
  userdata/
    env/
      default.env                <- shared global env
      aws.env                    <- vendor-specific
      github.env
    profiles/
      work.yaml
      personal.yaml
      active -> work.yaml        <- symlink to active profile
    preferences.yaml             <- output_format, color, verbose, etc.
    skills/                      <- per-skill registries
      scm/git/commit-analyzer/
        tokens.env
        config.yaml
        state/
        output/
        templates/
  registry-cache.json            <- cached catalog index
```

### Security

- `~/.agentx/userdata/env/` -- `chmod 700` (shared secrets)
- `~/.agentx/userdata/skills/*/tokens.env` -- `chmod 600` (skill-specific secrets)
- `~/.agentx/userdata/profiles/` -- restricted permissions (may contain sensitive data)
- `agentx init --global` sets correct permissions on creation
- `agentx doctor --check-userdata` verifies permissions
- `agentx doctor --check-registry` checks all `tokens.env` file permissions

---

## AI Tool Integration Architecture

### How Link Sync Works

`agentx link sync` reads `.agentx/project.yaml` and delegates config generation to per-tool CLI packages:

```
agentx link sync
  |
  +-- @agentx/claudecode-cli   -> generates .claude/CLAUDE.md, commands/, context/
  +-- @agentx/copilot-cli      -> generates .github/copilot-instructions.md
  +-- @agentx/augment-cli      -> generates augment config files
  +-- @agentx/opencode-cli     -> generates AGENTS.md (project root), .opencode/commands/, .opencode/context/
```

Each per-tool package is a standalone Node.js ESM module that exports `generate()` and `status()` functions. The Go CLI dispatches to these packages through `internal/integrations/dispatcher.go`.

### Symlink Strategy

- **Static content** (context, personas) -> symlinked from the project to `~/.agentx/installed/`
- **Generated files** (CLAUDE.md, copilot-instructions.md) -> generated by `agentx link sync`
- **Executable skills** -> invoked through `agentx run`, not symlinked

On Windows, if symlinks are not available (requires developer mode), AgentX falls back to copying files with a `.target` sidecar.

---

## Troubleshooting

### `agentx doctor`

Run `agentx doctor` with no flags to execute all diagnostic checks. Use specific flags to run individual checks.

```bash
# Full health check
agentx doctor

# Check only CLI dependencies
agentx doctor --check-cli

# Validate skill registries (tokens, config, permissions)
agentx doctor --check-registry

# Debug environment variable resolution for a skill
agentx doctor --trace-env skills/cloud/aws/ssm-lookup

# Fix common issues interactively
agentx doctor --fix

# Validate a manifest file
agentx doctor --check-manifest path/to/skill.yaml
```

### Common Issues

**"No type sources found"**
- Set `AGENTX_HOME` to the AgentX repository root
- Or ensure the `catalog/` directory is relative to the `agentx` binary

**Symlinks broken after reinstall**
- Run `agentx link sync` to regenerate all project configs
- Run `agentx doctor --check-links` to see which symlinks need repair

**Skill fails with "required input missing"**
- Check `skill.yaml` for the required inputs
- Pass inputs via `--input key=value` flags

**Tokens not configured**
- Run `agentx doctor --check-registry` to see which skills need tokens
- Run `agentx env edit <skill-path>` to configure tokens
- Run `agentx doctor --trace-env <skill-path>` to see the resolution chain

**Extensions not showing types**
- Run `agentx extension sync` to initialize submodules
- Run `agentx doctor --check-extensions` to verify status
- Check `project.yaml` resolution order

**Self-update fails**
- Check network connectivity to GitHub Releases (or your Nexus mirror)
- Set `mirror` in `~/.agentx/config.yaml` for enterprise environments
- Try `agentx update --version <specific-version>` to target a known release