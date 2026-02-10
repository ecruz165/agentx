# Contributing to AgentX

This guide covers development setup, coding standards, and the process for contributing to the AgentX project.

---

## Development Setup

### Prerequisites

| Tool | Version | Notes |
|------|---------|-------|
| Go | 1.25+ | See `.go-version` for the exact version |
| Node.js | 20+ | Required for skill wrappers and integrations |
| pnpm | 9+ | Mono-repo workspace manager |
| Git | 2.30+ | Submodule support for extensions |

### Getting Started

```bash
# Clone the repository
git clone https://github.com/agentx-labs/agentx.git
cd agentx

# Install workspace dependencies
pnpm install

# Build all packages
make build

# Run all tests
make test

# Set AGENTX_HOME for development (so the CLI finds the catalog)
export AGENTX_HOME=$(pwd)
```

### Development Workflow

```bash
# Build just the CLI
cd packages/cli && go build -o ../../dist/agentx .

# Run Go tests
cd packages/cli && go test ./...

# Run a specific test
cd packages/cli && go test ./internal/registry/ -run TestDiscover

# Build a specific package
pnpm --filter @agentx/cli run build

# Run all package tests via pnpm
pnpm run test
```

---

## Repository Structure

```
agentx/
  catalog/                     <- Composable types (what AgentX manages)
    context/                   <- Knowledge and documentation (static files)
    personas/                  <- Agent identity definitions (static files)
    skills/                    <- CLI/API wrappers (Node or Go packages)
    workflows/                 <- Multi-skill orchestrations (Node or Go)
    prompts/                   <- Agent-facing instructions (static + templates)
    templates/                 <- Distributable starting points (static)

  packages/                    <- Tooling (what builds/runs AgentX)
    cli/                       <- Go CLI source (the core engine)
      internal/
        cli/                   <- Cobra command definitions
        compose/               <- Prompt composition engine
        config/                <- User configuration management
        extension/             <- Extension submodule management
        integrations/          <- AI tool integration dispatcher
        linker/                <- Symlink and project config management
        manifest/              <- Manifest parser and validator
        platform/              <- Cross-platform filesystem operations
        registry/              <- Type discovery, indexing, and installation
        runtime/               <- Skill execution dispatchers (Node, Go)
        scaffold/              <- Type scaffolding from templates
        updater/               <- Self-update mechanism
        userdata/              <- User data directory management
    schema/                    <- JSON Schema definitions (@agentx/schema)
    shared/
      node/                    <- Shared Node utilities (@agentx/shared-node)
      tool-manager/            <- CLI dependency detection (@agentx/tool-manager)
    claudecode-cli/            <- Claude Code config generator
    copilot-cli/               <- GitHub Copilot config generator
    augment-cli/               <- Augment Code config generator
    opencode-cli/              <- OpenCode config generator
    vscode-copilot-chat-ext/   <- VS Code extension (future)

  extensions/                  <- Git submodules for custom knowledge bases
  scripts/                     <- Build and install scripts
  docs/                        <- Additional documentation
  .github/workflows/           <- CI/CD workflows
```

### Build System

Every buildable folder has its own `Makefile` as the canonical build interface. `package.json` scripts delegate to `make` for pnpm workspace orchestration. The chain is:

```
pnpm -r run build  ->  each package.json  ->  make build  ->  native tooling
```

The direction is strictly **pnpm calls make, never the reverse**. Node Makefiles use `npm` (not `pnpm`) so any package can build standalone without the workspace.

---

## How to Add a New Type

### Adding a Skill

1. **Scaffold** using the CLI:
   ```bash
   agentx create skill my-tool --topic cloud --vendor aws --runtime node
   ```

2. **Structure**: The scaffold creates:
   ```
   my-tool/
     Makefile          <- make build, test, clean
     package.json      <- @agentx/skill-cloud-my-tool
     skill.yaml        <- manifest with inputs, outputs, registry declaration
     index.mjs         <- skill logic with registry pattern boilerplate
   ```

3. **Manifest** (`skill.yaml`): Define metadata, CLI dependencies, inputs/outputs, and the registry declaration:
   ```yaml
   name: my-tool
   type: skill
   version: 0.1.0
   description: Short description of what this skill does
   tags: [cloud, aws]
   author: your-name
   runtime: node
   vendor: aws
   topic: cloud
   cli_dependencies:
     - name: aws
       min_version: "2.0.0"
   inputs:
     - name: paramName
       type: string
       required: true
       description: The parameter to look up
   outputs:
     format: json
   registry:
     tokens:
       - name: AWS_PROFILE
         required: false
     config:
       cache_ttl: 30
     state:
       - cache.json
     output:
       schema: ./output-schema.json
   ```

4. **Implement** the skill logic in `index.mjs` (Node) or `main.go` (Go). The scaffolded boilerplate includes the registry pattern for loading user data.

5. **Place** the skill in the correct catalog location: `catalog/skills/<topic>/<vendor>/<name>/`

### Adding a Context Type

1. Create a directory under `catalog/context/<topic>/`
2. Add a `context.yaml` manifest:
   ```yaml
   name: my-context
   type: context
   version: 1.0.0
   description: Description of this context
   tags: [relevant, tags]
   format: markdown
   sources:
     - patterns.md
     - examples.md
   ```
3. Add the referenced markdown source files

### Adding a Persona

1. Create a directory under `catalog/personas/<name>/`
2. Add a `persona.yaml` manifest referencing context types
3. Add a `persona.md` template file

### Adding a Workflow

1. Scaffold: `agentx create workflow my-flow`
2. Define steps in `workflow.yaml`, referencing existing skills
3. Implement orchestration logic in `index.mjs`

### Adding a Prompt

1. Scaffold: `agentx create prompt my-prompt`
2. Configure `prompt.yaml` to reference persona, context, skills, and workflows
3. Edit `prompt.hbs` with the Handlebars template

---

## How to Add an AI Tool Integration

Each AI tool integration is a standalone Node.js package under `packages/`. The Go CLI dispatches to these packages through `internal/integrations/`.

1. **Create the package** at `packages/<tool>-cli/`:
   ```
   packages/mytool-cli/
     Makefile
     package.json      <- @agentx/mytool-cli
     src/
       index.mjs       <- exports generate() and status() functions
   ```

2. **Implement the generator**: The `generate()` function receives the project config and installed types, and produces tool-specific config files (instruction files, commands, symlinks).

3. **Register the tool** in `packages/cli/internal/integrations/registry.go` so the dispatcher routes `agentx link sync` to your generator.

4. **Add the package** to `pnpm-workspace.yaml`:
   ```yaml
   packages:
     - 'packages/mytool-cli'
   ```

---

## Coding Standards

### Go Code

- Follow standard Go idioms and `go fmt`
- Wrap errors with context: `fmt.Errorf("doing something: %w", err)`
- Use structured logging where applicable
- Package-level documentation in `doc.go` files
- Tests in `*_test.go` files alongside the code they test

### Node.js (Skill Wrappers and Integrations)

- ESM modules only (`"type": "module"` in `package.json`)
- Minimal dependencies -- one external CLI/API per skill
- Type definitions for all exports where applicable

### Manifest Files

- Use YAML for all manifest files (`<type>.yaml`)
- Follow the schema definitions in `packages/schema/`
- Names must match the pattern `[a-z0-9][a-z0-9-]*`

---

## Testing

### Go Tests

```bash
# Run all Go tests
cd packages/cli && go test ./...

# Run tests for a specific package
cd packages/cli && go test ./internal/manifest/

# Run a specific test
cd packages/cli && go test ./internal/registry/ -run TestResolve

# Run with verbose output
cd packages/cli && go test -v ./...
```

### Integration Tests

```bash
# Run all tests across all packages
make test

# Run integration tests (if available)
make test-integration
```

### Writing Tests

- Go tests live alongside source files as `*_test.go`
- Use test fixtures in `testdata/` directories
- Integration tests should clean up after themselves
- Skill tests should mock external CLI dependencies

---

## Pull Request Process

### Branch Naming

Use the pattern: `task/<id>-<short-description>`

Examples:
- `task/12-add-search-filters`
- `task/7-scaffold-templates`

### Before Submitting

1. **Run tests**: `make test` (or `go test ./...` for Go-only changes)
2. **Format code**: `go fmt ./...` for Go files
3. **Validate manifests**: `agentx doctor --check-manifest <path>` for any new type manifests
4. **Update documentation** if your change affects CLI commands, manifest schemas, or the type system

### PR Requirements

- Tests pass
- Code is formatted (`go fmt`)
- New manifest files validate against the JSON Schema
- Meaningful commit messages describing the "why" rather than the "what"
- One concern per PR -- avoid mixing unrelated changes

---

## Architecture Reference

For detailed architecture documentation including manifest schemas, registry patterns, dependency resolution, and troubleshooting, see [docs/architecture.md](docs/architecture.md).