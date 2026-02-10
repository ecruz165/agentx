#!/usr/bin/env bash
# E2E integration test for the OpenCode tool integration.
# Verifies: build, unit tests, generate, status, and artifact correctness.
#
# Usage: ./scripts/e2e-opencode.sh
# Exit code: 0 on success, 1 on any failure.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PASS=0
FAIL=0
CLEANUP_DIRS=()

cleanup() {
  for d in "${CLEANUP_DIRS[@]}"; do
    rm -rf "$d"
  done
}
trap cleanup EXIT

check() {
  local desc="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    echo "  [PASS] $desc"
    PASS=$((PASS + 1))
  else
    echo "  [FAIL] $desc"
    FAIL=$((FAIL + 1))
  fi
}

check_contains() {
  local desc="$1"
  local file="$2"
  local pattern="$3"
  if grep -q "$pattern" "$file" 2>/dev/null; then
    echo "  [PASS] $desc"
    PASS=$((PASS + 1))
  else
    echo "  [FAIL] $desc (pattern '$pattern' not found in $file)"
    FAIL=$((FAIL + 1))
  fi
}

check_starts_with() {
  local desc="$1"
  local file="$2"
  local expected="$3"
  if head -1 "$file" 2>/dev/null | grep -q "^${expected}"; then
    echo "  [PASS] $desc"
    PASS=$((PASS + 1))
  else
    echo "  [FAIL] $desc"
    FAIL=$((FAIL + 1))
  fi
}

check_symlink_valid() {
  local desc="$1"
  local path="$2"
  if [ -L "$path" ] && [ -e "$path" ]; then
    echo "  [PASS] $desc"
    PASS=$((PASS + 1))
  else
    echo "  [FAIL] $desc"
    FAIL=$((FAIL + 1))
  fi
}

# ============================================================
# Phase 1: Build
# ============================================================
echo "=== Phase 1: Build ==="

echo "--- Building Go CLI binary ---"
CLI_BIN=$(mktemp /tmp/agentx-e2e-XXXXXX)
CLEANUP_DIRS+=("$CLI_BIN")
(cd "$REPO_ROOT/packages/cli" && go build -o "$CLI_BIN" .) 2>&1
check "Go CLI binary builds" test -x "$CLI_BIN"
echo "  Binary: $CLI_BIN"

# ============================================================
# Phase 2: Unit tests
# ============================================================
echo ""
echo "=== Phase 2: Unit Tests ==="

echo "--- Go unit tests ---"
GO_TEST_OUTPUT=$(cd "$REPO_ROOT/packages/cli" && go test ./... 2>&1) || true
if echo "$GO_TEST_OUTPUT" | grep -q "FAIL"; then
  echo "  [FAIL] Go unit tests"
  echo "$GO_TEST_OUTPUT" | grep "FAIL"
  FAIL=$((FAIL + 1))
else
  echo "  [PASS] Go unit tests (all packages)"
  PASS=$((PASS + 1))
fi

echo "--- Node opencode-cli unit tests ---"
NODE_TEST_OUTPUT=$(cd "$REPO_ROOT/packages/opencode-cli" && node --test test/**/*.test.mjs 2>&1) || true
NODE_FAIL_COUNT=$(echo "$NODE_TEST_OUTPUT" | grep "^# fail" | awk '{print $3}')
if [ "${NODE_FAIL_COUNT:-0}" = "0" ]; then
  NODE_PASS_COUNT=$(echo "$NODE_TEST_OUTPUT" | grep "^# pass" | awk '{print $3}')
  echo "  [PASS] Node opencode-cli tests ($NODE_PASS_COUNT passed)"
  PASS=$((PASS + 1))
else
  echo "  [FAIL] Node opencode-cli tests ($NODE_FAIL_COUNT failures)"
  FAIL=$((FAIL + 1))
fi

echo "--- Go integrations registry tests ---"
INTEG_OUTPUT=$(cd "$REPO_ROOT/packages/cli" && go test ./internal/integrations/ 2>&1) || true
if echo "$INTEG_OUTPUT" | grep -q "^ok"; then
  echo "  [PASS] Go integrations registry tests"
  PASS=$((PASS + 1))
else
  echo "  [FAIL] Go integrations registry tests"
  FAIL=$((FAIL + 1))
fi

# ============================================================
# Phase 3: E2E — OpenCode generate + status via bin scripts
# ============================================================
echo ""
echo "=== Phase 3: E2E Integration (generate + status) ==="

# Create a temp directory with mock installed types
TMPDIR=$(mktemp -d /tmp/agentx-e2e-opencode-XXXXXX)
CLEANUP_DIRS+=("$TMPDIR")

INSTALLED="$TMPDIR/installed"
PROJECT="$TMPDIR/project"
mkdir -p "$PROJECT"

# --- Set up mock installed types ---

# Persona
PERSONA_DIR="$INSTALLED/personas/test-persona"
mkdir -p "$PERSONA_DIR"
cat > "$PERSONA_DIR/manifest.yaml" <<'YAML'
name: test-persona
type: persona
version: "1.0.0"
description: You are a senior test engineer with deep testing expertise.
tone: direct, pragmatic
conventions:
  - Always write tests before implementation
  - Prefer integration tests over mocks
context:
  - context/testing/best-practices
YAML

# Context
CTX_DIR="$INSTALLED/context/testing/best-practices"
mkdir -p "$CTX_DIR"
cat > "$CTX_DIR/manifest.yaml" <<'YAML'
name: best-practices
type: context
version: "1.0.0"
description: Testing best practices documentation
format: markdown
sources:
  - patterns.md
YAML
cat > "$CTX_DIR/patterns.md" <<'MD'
# Testing Best Practices
Always verify edge cases and error paths.
MD

# Skill
SKILL_DIR="$INSTALLED/skills/testing/coverage-reporter"
mkdir -p "$SKILL_DIR"
cat > "$SKILL_DIR/manifest.yaml" <<'YAML'
name: coverage-reporter
type: skill
version: "1.0.0"
description: Reports test coverage metrics for the project
runtime: node
topic: testing
inputs:
  - name: path
    type: string
    required: true
    description: Path to the project
  - name: threshold
    type: number
    default: 80
    description: Minimum coverage percentage
YAML

# Workflow
WF_DIR="$INSTALLED/workflows/test-and-report"
mkdir -p "$WF_DIR"
cat > "$WF_DIR/manifest.yaml" <<'YAML'
name: test-and-report
type: workflow
version: "1.0.0"
description: Runs tests and generates coverage report
runtime: node
inputs:
  - name: projectPath
    type: string
    required: true
YAML

# --- Build the generate input JSON ---
GENERATE_INPUT=$(cat <<'JSON'
{
  "projectConfig": {
    "tools": ["opencode"],
    "active": {
      "personas": ["personas/test-persona"],
      "context": ["context/testing/best-practices"],
      "skills": ["skills/testing/coverage-reporter"],
      "workflows": ["workflows/test-and-report"]
    }
  },
  "installedPath": "INSTALLED_PLACEHOLDER",
  "projectPath": "PROJECT_PLACEHOLDER"
}
JSON
)
GENERATE_INPUT=$(echo "$GENERATE_INPUT" | sed "s|INSTALLED_PLACEHOLDER|$INSTALLED|g" | sed "s|PROJECT_PLACEHOLDER|$PROJECT|g")

echo "--- Running generate via bin/generate.mjs ---"
GENERATE_OUTPUT=$(echo "$GENERATE_INPUT" | node "$REPO_ROOT/packages/opencode-cli/bin/generate.mjs" 2>&1)
GENERATE_EXIT=$?

check "generate.mjs exits successfully" test "$GENERATE_EXIT" -eq 0
echo "  Output: $GENERATE_OUTPUT"

# --- Verify AGENTS.md ---
echo ""
echo "--- Verifying AGENTS.md ---"
check "AGENTS.md exists in project root" test -f "$PROJECT/AGENTS.md"
check "AGENTS.md NOT inside .opencode/" test ! -f "$PROJECT/.opencode/AGENTS.md"
check_contains "AGENTS.md contains persona description" "$PROJECT/AGENTS.md" "senior test engineer"
check_contains "AGENTS.md contains persona tone" "$PROJECT/AGENTS.md" "direct, pragmatic"
check_contains "AGENTS.md contains skill name" "$PROJECT/AGENTS.md" "coverage-reporter"
check_contains "AGENTS.md contains workflow name" "$PROJECT/AGENTS.md" "test-and-report"
check_contains "AGENTS.md references .opencode/context/" "$PROJECT/AGENTS.md" ".opencode/context/"

# --- Verify command files ---
echo ""
echo "--- Verifying .opencode/commands/ ---"
check ".opencode/commands/ directory exists" test -d "$PROJECT/.opencode/commands"
check "coverage-reporter.md command file exists" test -f "$PROJECT/.opencode/commands/coverage-reporter.md"
check "test-and-report.md command file exists" test -f "$PROJECT/.opencode/commands/test-and-report.md"
check_starts_with "coverage-reporter.md has YAML frontmatter" "$PROJECT/.opencode/commands/coverage-reporter.md" "---"
check_starts_with "test-and-report.md has YAML frontmatter" "$PROJECT/.opencode/commands/test-and-report.md" "---"
check_contains "coverage-reporter.md contains agentx run" "$PROJECT/.opencode/commands/coverage-reporter.md" "agentx run skills/testing/coverage-reporter"
check_contains "coverage-reporter.md contains input placeholders" "$PROJECT/.opencode/commands/coverage-reporter.md" "{{path}}"
check_contains "test-and-report.md contains agentx run" "$PROJECT/.opencode/commands/test-and-report.md" "agentx run workflows/test-and-report"

# --- Verify context symlinks ---
echo ""
echo "--- Verifying .opencode/context/ ---"
check ".opencode/context/ directory exists" test -d "$PROJECT/.opencode/context"
check_symlink_valid "context symlink exists and is valid" "$PROJECT/.opencode/context/testing-best-practices"

# --- Run status check ---
echo ""
echo "--- Running status via bin/status.mjs ---"

# status needs .agentx/project.yaml to exist for staleness checks
mkdir -p "$PROJECT/.agentx"
cat > "$PROJECT/.agentx/project.yaml" <<'YAML'
tools:
  - opencode
active:
  personas:
    - personas/test-persona
  context:
    - context/testing/best-practices
  skills:
    - skills/testing/coverage-reporter
  workflows:
    - workflows/test-and-report
YAML
# Make project.yaml older than AGENTS.md so status is up-to-date
touch -t 202501010000 "$PROJECT/.agentx/project.yaml"

STATUS_INPUT=$(cat <<JSON
{"projectPath": "$PROJECT"}
JSON
)

STATUS_OUTPUT=$(echo "$STATUS_INPUT" | node "$REPO_ROOT/packages/opencode-cli/bin/status.mjs" 2>&1)
STATUS_EXIT=$?

check "status.mjs exits successfully" test "$STATUS_EXIT" -eq 0
echo "  Output: $STATUS_OUTPUT"

# Parse status output
check_contains "status reports tool=opencode" <(echo "$STATUS_OUTPUT") '"tool":"opencode"'
check_contains "status reports up-to-date" <(echo "$STATUS_OUTPUT") '"status":"up-to-date"'
check_contains "status reports valid symlinks" <(echo "$STATUS_OUTPUT") '"valid":1'

# --- Verify idempotency: run generate again ---
echo ""
echo "--- Verifying idempotency (second generate) ---"
GENERATE_OUTPUT2=$(echo "$GENERATE_INPUT" | node "$REPO_ROOT/packages/opencode-cli/bin/generate.mjs" 2>&1)
check "second generate exits successfully" test $? -eq 0
check_contains "second generate reports updated (not created) for AGENTS.md" <(echo "$GENERATE_OUTPUT2") '"updated":\[.*AGENTS.md'

# ============================================================
# Summary
# ============================================================
echo ""
echo "========================================="
echo "  E2E Results: $PASS passed, $FAIL failed"
echo "========================================="

if [ "$FAIL" -gt 0 ]; then
  exit 1
fi
exit 0
