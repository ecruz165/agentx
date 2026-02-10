# AgentX: Supply Chain Manager for AI Agents

> A technical overview for engineering leadership

---

## The Problem

AI coding agents are only as good as the context, skills, and instructions they receive. Today, that knowledge is **trapped in individual projects and individual developers' heads**:

- **No shared catalog** — Teams build the same git analysis scripts, the same AWS lookups, the same code review patterns independently. There's no central repo of reusable agent skills that benefit everyone.
- **No layered ownership** — A platform team wants to provide baseline agent context (coding standards, security patterns, deployment checklists), but individual teams need to **override and extend** for their domain without forking.
- **No tool flexibility** — Developers have preferences. Some use Claude Code, others Copilot, others Augment or OpenCode. The company approves all four, but agent knowledge is locked to one tool's config format.
- **No composability** — A senior Java developer's persona, Spring Boot error handling context, a git commit analyzer skill, and a code review workflow should snap together like building blocks — not be copy-pasted into each project's README.

**The goal:** A company-wide **agent context repository** that delivers reusable skills, personas, and knowledge to every developer, on every team, in whatever AI tool they prefer — with the flexibility for teams to extend and override.

---

## What AgentX Does

AgentX is the **supply chain** between your organization's agent knowledge and the AI tools developers use daily. It manages a shared catalog of reusable types that any team can consume, extend, or override.

**Three layers of the solution:**

1. **Shared Catalog** — A company-wide repo of skills, personas, context, workflows, and prompts that provide value across all teams
2. **Extension System** — Teams override or extend the shared catalog with their own private knowledge, without forking
3. **Multi-Tool Delivery** — Developers choose their preferred approved AI tool; AgentX generates native configs for all of them

```
Company Catalog (shared skills, personas, context)
     +  Team Extensions (domain-specific overrides)
     +  Developer's Tool Choice (Claude Code / Copilot / Augment / OpenCode)
     ──────────────────────────────────
     =  Fully configured AI agent, native to each tool
```

AgentX is **plumbing, not porcelain**. The human-facing experience is the AI agent itself — AgentX ensures the agent has the right tools and instructions. No runtime injection. AI tools discover configs through their native mechanisms.

---

## The Type System

AgentX defines **six composable types** — each with a single responsibility:

| Type | Purpose | Runtime |
|------|---------|---------|
| **Context** | Knowledge docs, patterns, examples | Static files |
| **Persona** | Agent identity — expertise, tone, conventions | Static files |
| **Skill** | Atomic CLI/API wrapper (one dependency max) | Node or Go |
| **Workflow** | Multi-step orchestration chaining skills | Node or Go |
| **Prompt** | Composed agent instructions from all types | Template |
| **Template** | Distributable starting points (queries, reports) | Static files |

Every type has a `manifest.yaml` declaring metadata, dependencies, inputs, outputs, and runtime requirements.

---

## Dependency Graph

Types follow a **strict one-directional dependency chain**:

```
context          (foundation — no dependencies)
   ↓
persona          (references context)
   ↓
skill            (atomic — no type dependencies)
   ↓
workflow         (chains multiple skills)
   ↓
prompt           (composes everything above)
```

**Key rule:** One skill = one CLI/API dependency. Always. Multi-tool logic belongs in workflows.

This means installing a single prompt automatically resolves the full tree:

```
agentx install prompts/java-pr-review

Install plan:
  context:  spring-boot/error-handling
  context:  spring-boot/security
  persona:  senior-java-dev
  skill:    scm/git/commit-analyzer
  skill:    ai/token-counter
  workflow: code-review
  prompt:   java-pr-review
```

---

## The Developer Flow

Three commands get a project fully configured:

```bash
# 1. Initialize — declare which AI tools you use
agentx init --tools claude-code,copilot,augment,opencode

# 2. Link types — pick what your agents should know and do
agentx link add personas/senior-java-dev
agentx link add skills/scm/git/commit-analyzer
agentx link add context/spring-boot/error-handling

# 3. Sync — generate native config files for each tool
agentx link sync
```

That's it. Your AI tools now have consistent instructions, available skills, and shared context — without any manual file copying.

---

## AI Tool Config Generation

`agentx link sync` generates **native** config files for each tool:

### Claude Code
```
.claude/CLAUDE.md              ← persona + skills + context refs
.claude/commands/*.md          ← skill wrappers as slash commands
.claude/context/               ← symlinks to installed context
```

### GitHub Copilot
```
.github/copilot-instructions.md   ← persona + context refs
.github/copilot-context/          ← symlinks to installed context
```

### Augment Code
```
.augment/augment-guidelines.md   ← persona + context refs
.augment/context/                ← symlinks to installed context
```

### OpenCode
```
AGENTS.md                        ← persona + skills (project root)
.opencode/commands/*.md          ← commands with YAML frontmatter
.opencode/context/               ← symlinks to installed context
```

Each tool gets exactly what it expects, in the format it expects, at the path it expects.

---

## Live Demo: The Full Flow

```bash
# Install a full prompt (auto-resolves 7 types)
$ agentx install prompts/java-pr-review
Install plan:
  context:  spring-boot/error-handling, spring-boot/security
  persona:  senior-java-dev
  skill:    scm/git/commit-analyzer, ai/token-counter
  workflow: code-review
  prompt:   java-pr-review
? Proceed? (Y/n) Y
✓ Installed 7 types

# Link to project and sync
$ cd my-project
$ agentx init --tools claude-code,opencode
$ agentx link add prompts/java-pr-review
$ agentx link sync
  [OK] claude-code: generated .claude/CLAUDE.md + 3 commands + 2 context links
  [OK] opencode:    generated AGENTS.md + 3 commands + 2 context links

# Check health
$ agentx link status
  [OK] claude-code: up-to-date   Symlinks: 2/2 valid
  [OK] opencode:    up-to-date   Symlinks: 2/2 valid

# Run a skill directly
$ agentx run skills/scm/git/commit-analyzer --input repo-path=. --input since=7d
```

---

## Search, Install & Stay Current

The catalog is a **living repository** — new skills, context, and workflows are added frequently. AgentX makes discovery and adoption frictionless:

```bash
# Search the catalog by keyword, type, topic, vendor, or tag
$ agentx search "git commit" --type skill
  skills/scm/git/commit-analyzer    Analyzes git commit history for team metrics
  skills/scm/git/branch-status      Shows branch comparison and merge readiness

# Install with full dependency resolution
$ agentx install skills/scm/git/commit-analyzer
Install plan:
  skill: scm/git/commit-analyzer
? Proceed? (Y/n) Y
✓ Installed 1 type
```

### Automatic update notifications

Every `agentx` command checks for updates in the background (cached, non-blocking). When new versions are available — including new catalog types — developers see:

```
A new version of agentx is available: 1.1.0 -> 1.2.0
Run `agentx update` to upgrade.
```

No disruption, no forced updates — just a nudge to stay current.

---

## Skill Architecture

### Two runtimes, one rule

| | Node Skills | Go Skills |
|---|------------|-----------|
| **When** | Wraps external CLI (git, aws, mvn) | Self-contained, no dependencies |
| **Entry** | `index.mjs` (ESM) | `main.go` |
| **Deps** | `package.json` + `npm install` | Single compiled binary |
| **Example** | `commit-analyzer` (wraps git) | `token-counter` (pure computation) |

### Hard rule: One skill = one CLI/API

- Need git AND jq? That's a **workflow**, not a skill.
- This keeps skills composable and dependency requirements predictable.

### The Registry Pattern

Every installed skill gets a **userdata registry** for state management:

```
~/.agentx/userdata/skills/scm/git/commit-analyzer/
  tokens.env       ← secrets (chmod 600)
  config.yaml      ← configuration
  state/           ← persisted state between runs
  output/          ← latest.json (consumed by other skills)
  templates/       ← graduated outputs saved for reuse
```

Skills read from well-known paths — no AgentX imports required. They work standalone.

---

## Extension System

Organizations bring **proprietary knowledge** as git submodules — private personas, context, and skills that extend the open-source catalog:

```bash
agentx extension add acme-corp git@github.com:acme/agentx-knowledge.git
```

### Resolution order

```yaml
# project.yaml
resolution:
  - core           # built-in catalog
  - acme-corp      # higher priority — wins on name conflicts
```

An enterprise can override `personas/senior-java-dev` with their own version. All installs and syncs use the override automatically.

### Extension status

```
$ agentx extension list
NAME         BRANCH   STATUS
acme-corp    main     ok
team-tools   develop  uninitialized
```

---

## Secret & Profile Management

### Environment resolution (layered, predictable)

```
1. env/default.env           ← global defaults (lowest priority)
2. env/aws.env               ← vendor-specific
3. skills/<path>/tokens.env  ← skill-specific (highest priority)
```

Debug with: `agentx doctor --trace-env cloud/aws/ssm-lookup`

### Profiles (switch environments instantly)

```bash
$ agentx profile list
  work (active)
  personal
  staging

$ agentx profile use staging
```

Profiles store org-specific settings: `aws_profile`, `github_org`, `splunk_host`, `default_branch`.

### Security

- `tokens.env` files: `chmod 600`
- `env/` directory: `chmod 700`
- `agentx doctor --check-userdata` verifies permissions

---

## Health Checks & Diagnostics

`agentx doctor` is the single command for diagnosing any AgentX issue:

```
$ agentx doctor
Runtime check:
  [OK]   go found at /usr/local/bin/go
  [OK]   node found at /usr/local/bin/node
Extensions check:
  [OK]   acme-corp: clean
CLI dependency check:
  [OK]   git >= 2.0.0 (found 2.43.0)
  [MISS] aws >= 2.0.0 (not found)
Registry check:
  [OK]   scm/git/commit-analyzer: tokens present, config valid
  [WARN] cloud/aws/ssm-lookup: SSM_ROLE_ARN not set
Link check:
  [OK]   claude-code: up-to-date, 3/3 symlinks valid
  [!!]   augment: stale (needs re-sync)
```

### Individual checks and auto-fix

```bash
agentx doctor --check-cli          # verify CLI dependencies
agentx doctor --check-registry     # validate skill registries
agentx doctor --check-links        # verify symlinks
agentx doctor --fix                # interactive repair
agentx doctor --trace-env <skill>  # debug env resolution
```

---

## Enterprise Distribution

For organizations behind firewalls, AgentX supports **Sonatype Nexus** distribution:

### What ships

| Artifact | Destination |
|----------|-------------|
| Go binary (all OS/arch) | Nexus raw repository |
| `@agentx/*` npm packages | Nexus npm registry |
| `install.sh` + `version.txt` | Nexus raw repository |
| SHA256 checksums | Nexus raw repository |

### Air-gapped install

```bash
AGENTX_MIRROR=https://nexus.corp.com/repository/agentx-releases \
  curl -sSL $AGENTX_MIRROR/install.sh | bash
```

### Self-update from mirror

```bash
agentx config set mirror https://nexus.corp.com/repository/agentx-releases
agentx update
```

Also supports internal **Homebrew taps** for macOS teams.

---

## Architecture

### Mono-repo structure

```
agentx/
  catalog/          ← composable types (what AgentX manages)
    skills/         ← Node and Go skill implementations
    workflows/      ← multi-step orchestrations
    personas/       ← agent identities
    context/        ← knowledge documents
    prompts/        ← composed agent instructions
    templates/      ← distributable starting points

  packages/         ← tooling (what builds/runs AgentX)
    cli/            ← Go CLI (Cobra commands, core engine)
    schema/         ← JSON Schema for manifest validation
    shared/node/    ← @agentx/shared-node utilities
    claudecode-cli/ ← Claude Code config generator
    copilot-cli/    ← Copilot config generator
    augment-cli/    ← Augment config generator
    opencode-cli/   ← OpenCode config generator

  extensions/       ← enterprise knowledge (git submodules)
```

### Tech stack

- **Go CLI** — core engine (Cobra commands, dependency resolution, dispatching)
- **Node ESM** — per-tool config generators + skill wrappers
- **pnpm workspace** — monorepo package management
- **GoReleaser** — cross-platform binary distribution

---

## What's Next

| Phase | Focus | Status |
|-------|-------|--------|
| 1. Foundation | Schema, CLI skeleton, build system, sample types | Done |
| 2. Install & Run | Userdata, registries, tool-manager, install/run | Done |
| 3. AI Tool Integration | Link sync, per-tool generators (4 tools) | Done |
| 4. Search & Polish | Search, self-update, enterprise distribution | Done |
| **5. VS Code Extension** | **Visual type browser, inline skill execution** | **Planned** |
| **6. Marketplace** | **Public registry, versioned publishing, discovery** | **Planned** |

### Current metrics

- **35 tasks** completed across 4 implementation phases
- **4 AI tools** supported (Claude Code, Copilot, Augment, OpenCode)
- **6 composable types** with full dependency resolution
- **17 CLI commands** covering the full lifecycle
- Enterprise-ready with Nexus distribution and extension system
