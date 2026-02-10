# AgentX

Supply chain manager for AI agent configurations. Install, link, compose, and orchestrate reusable skills, workflows, prompts, and personas across Claude Code, GitHub Copilot, Augment Code, and OpenCode.

AgentX is plumbing, not porcelain. The human-facing experience is the AI agent itself; AgentX ensures the agent has the right tools and instructions. It manages the installation, linking, and discovery of reusable types that power AI coding assistants.

---

## Quick Start

```bash
# 1. Install AgentX
curl -sSL https://raw.githubusercontent.com/agentx-labs/agentx/main/scripts/install.sh | bash

# 2. Initialize global userdata (creates ~/.agentx/userdata/)
agentx init --global

# 3. Install a prompt (resolves the full dependency tree automatically)
agentx install prompts/code-review/java-pr-review

# 4. Link types to your project (generates AI tool config files)
cd my-project
agentx init --tools claude-code,copilot,augment,opencode
agentx link add personas/senior-java-dev
agentx link add skills/scm/git/commit-analyzer
agentx link sync

# 5. Run a skill (typically invoked by agents, not humans)
agentx run skills/scm/git/commit-analyzer --input repoPath=. --input days=30
```

After `agentx link sync`, your AI tools discover the linked configurations through their native mechanisms -- no AgentX runtime injection required.

---

## Architecture Overview

### Type System

AgentX defines six composable types. Five follow a strict one-directional dependency graph, plus templates as a standalone type:

```
context     <- foundation, no dependencies
persona     <- references context
skill       <- atomic, no type dependencies (one CLI/API only)
workflow    <- references skills
prompt      <- references all of the above

template    <- standalone: distributable starting points (queries, reports, formats)
```

| Type | Purpose | Runtime |
|------|---------|---------|
| **context** | Knowledge, documentation, patterns, examples (tokenized for AI consumption) | None (static files) |
| **persona** | Reusable agent identity, expertise, tone, conventions | None (static files) |
| **skill** | Atomic CLI/API wrapper -- one external dependency max | Node (CLI wrappers) or Go (self-contained) |
| **workflow** | Step-by-step orchestration chaining multiple skills | Node or Go |
| **prompt** | Agent-facing instructions -- composes all other types | None (templates + manifest) |
| **template** | Distributable starting-point templates -- queries, reports, migration scripts | None (static files) |

### Key Rules

- **One skill = one CLI/API dependency** -- hard rule. Multi-tool orchestration belongs in workflows.
- **Wraps an external CLI/API** (git, mvn, aws, kubectl) -> **JS ESM** skill. **Self-contained, no external dependency** -> **Go** skill.
- **Dependency direction** is strictly: context -> persona -> skill -> workflow -> prompt.

### Platform Architecture

AgentX is a single mono-repo with three top-level concerns:

```
agentx/
  catalog/          <- composable types (what AgentX manages)
  packages/         <- tooling (what builds/runs AgentX)
  extensions/       <- custom knowledge bases (git submodules)
```

The Go CLI is the source of truth. AI agents discover and consume configurations through their native mechanisms -- AgentX ensures the right files are in the right places.

---

## CLI Command Reference

| Command | Description |
|---------|-------------|
| `agentx init` | Initialize AgentX in a project (`--global` for user-level config) |
| `agentx install <type-path>` | Install a type and its dependencies to `~/.agentx/installed/` |
| `agentx uninstall <type-path>` | Remove an installed type |
| `agentx list` | List installed types (filter with `--type`) |
| `agentx search [query]` | Search the registry/catalog (filter with `--type`, `--topic`, `--vendor`, `--tag`, `--cli`) |
| `agentx run <type-path>` | Execute an installed skill or workflow |
| `agentx prompt [type-path]` | Compose a prompt from installed types (interactive if no args) |
| `agentx create <type> <name>` | Scaffold a new type from a template |
| `agentx link add <type-path>` | Link a type to the current project |
| `agentx link remove <type-path>` | Unlink a type from the current project |
| `agentx link sync` | Regenerate all AI tool configurations |
| `agentx link status` | Show status of linked configurations |
| `agentx doctor` | Health check (use `--check-cli`, `--check-registry`, `--check-links`, `--fix`) |
| `agentx update` | Self-update the agentx binary (`--check` to check only) |
| `agentx config set/get` | Manage user settings in `~/.agentx/config.yaml` |
| `agentx profile list/use/show` | Manage user configuration profiles |
| `agentx env list/edit/show` | Manage `.env` secret files (shared and per-skill) |
| `agentx extension add/remove/list/sync` | Manage knowledge base git submodule extensions |
| `agentx version` | Print version information |

### Install Flags

```
--no-deps    Install only the specified type, skip dependencies
--yes (-y)   Skip confirmation prompt
```

### Search Flags

```
--type       Filter by type (skill, workflow, prompt, persona, context, template)
--topic      Filter by topic (scm, cicd, cloud, java, observability, ...)
--vendor     Filter by vendor (aws, github, harness, splunk, ...)
--tag        Filter by tags (comma-separated)
--cli        Filter by CLI dependency (git, aws, mvn, ...)
--json       Output as JSON
```

### Doctor Flags

```
--check-cli         Verify all CLI dependencies for installed skills
--check-runtime     Verify Node/Go are available
--check-links       Verify symlinks are intact
--check-extensions  Verify submodules initialized and synced
--check-userdata    Verify userdata directory exists with correct permissions
--check-registry    Validate skill registries against skill.yaml declarations
--check-manifest <path>  Validate a manifest file
--fix               Interactively install missing tools and initialize registries
--trace-env <skill> Show env resolution order for a specific skill
```

---

## AI Tool Integrations

AgentX generates configuration files for each supported AI tool through `agentx link sync`. The generated files reference installed types via symlinks and generated instruction files.

### Claude Code

`agentx link sync` generates:
- `.claude/CLAUDE.md` -- persona instructions and available skills
- `.claude/commands/` -- skill and workflow wrappers as commands
- `.claude/context/` -- symlinks to installed context

### GitHub Copilot

`agentx link sync` generates:
- `.github/copilot-instructions.md` -- persona instructions with context references
- `.github/copilot-context/` -- symlinks to installed context

### Augment Code

`agentx link sync` generates:
- `.augment/augment-guidelines.md` -- persona instructions with context references
- `.augment/context/` -- symlinks to installed context

### OpenCode

`agentx link sync` generates:
- `AGENTS.md` -- persona instructions and available skills (project root, not inside `.opencode/`)
- `.opencode/commands/` -- skill and workflow wrappers as commands (with YAML frontmatter)
- `.opencode/context/` -- symlinks to installed context

### Adding a New AI Tool

Each tool integration lives in its own package under `packages/` (e.g., `packages/claudecode-cli/`). The Go CLI dispatches to these per-tool packages through `internal/integrations/`. See [CONTRIBUTING.md](CONTRIBUTING.md) for details on adding new tool integrations.

---

## Extension System

Organizations can bring proprietary types (context, personas, skills, etc.) as git submodules mounted under `extensions/`. This keeps proprietary knowledge in private repos while leveraging the open-source AgentX core.

```bash
# Add a knowledge base extension
agentx extension add acme-corp git@github.com:acme/agentx-knowledge.git

# List extensions and their status
agentx extension list

# Sync all extension submodules
agentx extension sync
```

Extensions follow the same type directory conventions as core types. Resolution order is configured in `project.yaml`:

```yaml
# project.yaml
extensions:
  - name: acme-corp
    path: extensions/acme-corp
    source: git@github.com:acme/agentx-knowledge.git
    branch: main

resolution:
  - core           # built-in types (catalog/)
  - acme-corp      # higher priority -- wins on name conflicts
```

When AgentX looks up a type, it searches in resolution order. Extension types can reference both core types and types within the same extension.

---

## User Data and Skill Registry

Installed types live in `~/.agentx/installed/`. User-specific configuration, secrets, and state live in `~/.agentx/userdata/`:

```
~/.agentx/
  config.yaml                    <- user-level settings
  installed/                     <- installed types (mirroring catalog structure)
  userdata/
    env/                         <- shared .env files (default.env, aws.env, ...)
    profiles/                    <- named config profiles (work.yaml, personal.yaml)
    preferences.yaml             <- user-wide defaults
    skills/                      <- per-skill registries
      <topic>/<vendor>/<name>/
        tokens.env               <- skill-specific secrets
        config.yaml              <- skill-specific configuration
        state/                   <- internal persisted state
        output/                  <- reusable output (latest.json)
        templates/               <- graduated output templates
```

Skills resolve user data via the `AGENTX_USERDATA` environment variable (defaults to `~/.agentx/userdata`). Environment variables load in resolution order: `env/default.env` -> `env/<vendor>.env` -> `skills/<path>/tokens.env` (highest priority).

Use `agentx doctor --trace-env <skill>` to debug environment resolution.

---

## Enterprise Distribution

AgentX supports enterprise distribution through Sonatype Nexus (raw + npm repositories), internal Homebrew taps, and the `AGENTX_MIRROR` environment variable for air-gapped environments.

See [docs/enterprise-setup.md](docs/enterprise-setup.md) for full setup instructions.

---

## Development

### Prerequisites

- Go 1.25+ (see `.go-version`)
- Node.js 20+
- pnpm 9+

### Building

```bash
pnpm install          # Install workspace dependencies
make build            # Build all packages
make test             # Run all tests
```

### Testing

```bash
# Go tests
cd packages/cli && go test ./...

# All package tests via pnpm
pnpm run test
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for full development setup instructions.

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, coding standards, how to add new types or tool integrations, and the PR process.

---

## License

See [LICENSE](LICENSE) for details.