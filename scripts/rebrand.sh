#!/bin/bash
set -euo pipefail

# rebrand.sh — Apply branding.yaml changes across the entire repository.
#
# Forkers: edit branding.yaml at the repo root, then run this script.
# It updates Go source, markdown docs, agent commands, install scripts,
# GoReleaser config, and YAML configs.
#
# Usage:
#   ./scripts/rebrand.sh              # apply current branding.yaml
#   ./scripts/rebrand.sh --dry-run    # show what would change

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
BRANDING_FILE="$REPO_ROOT/branding.yaml"
CLI_DIR="$REPO_ROOT/packages/cli"
DRY_RUN=false

if [[ "${1:-}" == "--dry-run" ]]; then
  DRY_RUN=true
  echo "[DRY RUN] Showing changes without applying them."
  echo ""
fi

# --- Parse branding.yaml ---

parse_yaml_value() {
  local key="$1"
  grep "^${key}:" "$BRANDING_FILE" | sed -E "s/^${key}: *([^ #]+).*/\1/" | tr -d "'\""
}

NEW_CLI_NAME="$(parse_yaml_value cli_name)"
NEW_DISPLAY_NAME="$(parse_yaml_value display_name)"
NEW_HOME_DIR="$(parse_yaml_value home_dir)"
NEW_ENV_PREFIX="$(parse_yaml_value env_prefix)"
NEW_GO_MODULE="$(parse_yaml_value go_module)"
NEW_GITHUB_REPO="$(parse_yaml_value github_repo)"

# Current values (these are what the script replaces FROM).
CURRENT_GO_MODULE="$(head -1 "$CLI_DIR/go.mod" | awk '{print $2}')"
CURRENT_CLI_NAME="agentx"
CURRENT_DISPLAY_NAME="AgentX"
CURRENT_HOME_DIR=".agentx"
CURRENT_ENV_PREFIX="AGENTX"

echo "Rebrand configuration:"
echo "  CLI name:      ${CURRENT_CLI_NAME} -> ${NEW_CLI_NAME}"
echo "  Display name:  ${CURRENT_DISPLAY_NAME} -> ${NEW_DISPLAY_NAME}"
echo "  Home dir:      ${CURRENT_HOME_DIR} -> ${NEW_HOME_DIR}"
echo "  Env prefix:    ${CURRENT_ENV_PREFIX} -> ${NEW_ENV_PREFIX}"
echo "  Go module:     ${CURRENT_GO_MODULE} -> ${NEW_GO_MODULE}"
echo "  GitHub repo:   ${NEW_GITHUB_REPO}"
echo ""

# --- Helper: portable sed in-place ---

sedi() {
  sed -i.bak "$1" "$2" && rm -f "${2}.bak"
}

# --- Helper: find-and-replace across files matching a glob ---
# Usage: replace_pattern <old> <new> <find_args...>
# Example: replace_pattern "agentx" "mycli" "$CLI_DIR" -name "*.go"

replace_pattern() {
  local old="$1"
  local new="$2"
  shift 2

  if [[ "$old" == "$new" ]]; then
    return
  fi

  local count=0
  while IFS= read -r f; do
    if grep -q "$old" "$f" 2>/dev/null; then
      if $DRY_RUN; then
        echo "    $f"
      else
        sedi "s|${old}|${new}|g" "$f"
      fi
      count=$((count + 1))
    fi
  done < <(find "$@" -type f 2>/dev/null)

  if ! $DRY_RUN && [[ $count -gt 0 ]]; then
    echo "  Replaced in $count files."
  fi
}

# --- Step 1: Update Go module path ---

echo "Step 1: Go module path..."
if [[ "$CURRENT_GO_MODULE" != "$NEW_GO_MODULE" ]]; then
  if $DRY_RUN; then
    echo "  Would replace '${CURRENT_GO_MODULE}' -> '${NEW_GO_MODULE}':"
  fi
  replace_pattern "$CURRENT_GO_MODULE" "$NEW_GO_MODULE" \
    "$CLI_DIR" \( -name "*.go" -o -name "go.mod" \)
else
  echo "  Unchanged, skipping."
fi

# --- Step 2: Sync branding.yaml into Go embed location ---

echo "Step 2: Sync branding.yaml..."
if $DRY_RUN; then
  echo "  Would copy branding.yaml -> packages/cli/internal/branding/branding.yaml"
else
  cp "$BRANDING_FILE" "$CLI_DIR/internal/branding/branding.yaml"
  echo "  Synced."
fi

# --- Step 3: Update install.sh ---

echo "Step 3: install.sh..."
INSTALL_SCRIPT="$REPO_ROOT/scripts/install.sh"
if [[ -f "$INSTALL_SCRIPT" ]]; then
  if $DRY_RUN; then
    echo "  Would update REPO, binary name, and next-steps references."
  else
    sedi "s|^REPO=.*|REPO=\"${NEW_GITHUB_REPO}\"|" "$INSTALL_SCRIPT"
    sedi "s|/${CURRENT_CLI_NAME}_|/${NEW_CLI_NAME}_|g" "$INSTALL_SCRIPT"
    sedi "s|\"${CURRENT_CLI_NAME}\"|\"${NEW_CLI_NAME}\"|g" "$INSTALL_SCRIPT"
    sedi "s|/${CURRENT_CLI_NAME}\$|/${NEW_CLI_NAME}|g" "$INSTALL_SCRIPT"
    sedi "s|${CURRENT_CLI_NAME} v|${NEW_CLI_NAME} v|g" "$INSTALL_SCRIPT"
    sedi "s|${CURRENT_CLI_NAME} init|${NEW_CLI_NAME} init|g" "$INSTALL_SCRIPT"
    sedi "s|${CURRENT_CLI_NAME} search|${NEW_CLI_NAME} search|g" "$INSTALL_SCRIPT"
    echo "  Updated."
  fi
else
  echo "  Not found, skipping."
fi

# --- Step 4: Update .goreleaser.yaml ---

echo "Step 4: .goreleaser.yaml..."
GORELEASER="$REPO_ROOT/.goreleaser.yaml"
if [[ -f "$GORELEASER" ]]; then
  if $DRY_RUN; then
    echo "  Would update binary and project_name references."
  else
    sedi "s|binary: ${CURRENT_CLI_NAME}|binary: ${NEW_CLI_NAME}|g" "$GORELEASER"
    sedi "s|project_name: ${CURRENT_CLI_NAME}|project_name: ${NEW_CLI_NAME}|g" "$GORELEASER"
    echo "  Updated."
  fi
else
  echo "  Not found, skipping."
fi

# --- Step 5: Update markdown documentation ---
#
# Covers: README.md, CONTRIBUTING.md, CLAUDE.md, docs/*.md, .claude/commands/*.md
# Skips: .plans/*.md (historical dev artifacts)

echo "Step 5: Markdown files (docs, README, CLAUDE.md, agent commands)..."

if $DRY_RUN; then
  echo "  Patterns:"
  echo "    '${CURRENT_DISPLAY_NAME}' -> '${NEW_DISPLAY_NAME}'"
  echo "    '${CURRENT_ENV_PREFIX}_'  -> '${NEW_ENV_PREFIX}_'"
  echo "    '~/${CURRENT_HOME_DIR}'   -> '~/${NEW_HOME_DIR}'"
  echo "    '.${CURRENT_CLI_NAME}/'   -> '.${NEW_CLI_NAME}/'"
  echo "    CLI command references    -> '${NEW_CLI_NAME}'"
  echo "  Files:"
fi

# Find all .md files except .plans/ and node_modules/
MD_FIND_ARGS=("$REPO_ROOT" -name "*.md" -not -path "*/.plans/*" -not -path "*/node_modules/*" -not -path "*/.git/*")

# Replace display name first (e.g., "AgentX" before "agentx" to avoid partial matches).
replace_pattern "$CURRENT_DISPLAY_NAME" "$NEW_DISPLAY_NAME" "${MD_FIND_ARGS[@]}"
replace_pattern "${CURRENT_ENV_PREFIX}_" "${NEW_ENV_PREFIX}_" "${MD_FIND_ARGS[@]}"
replace_pattern "~/${CURRENT_HOME_DIR}" "~/${NEW_HOME_DIR}" "${MD_FIND_ARGS[@]}"
replace_pattern "\\.${CURRENT_CLI_NAME}/" ".${NEW_CLI_NAME}/" "${MD_FIND_ARGS[@]}"

# Replace CLI name in command invocations and inline code.
if [[ "$CURRENT_CLI_NAME" != "$NEW_CLI_NAME" ]]; then
  while IFS= read -r f; do
    if grep -q "${CURRENT_CLI_NAME}" "$f" 2>/dev/null; then
      if $DRY_RUN; then
        echo "    $f (cli_name)"
      else
        sedi "s|\`${CURRENT_CLI_NAME}\`|\`${NEW_CLI_NAME}\`|g" "$f"
        sedi "s|${CURRENT_CLI_NAME} |${NEW_CLI_NAME} |g" "$f"
        sedi "s| ${CURRENT_CLI_NAME}\$| ${NEW_CLI_NAME}|g" "$f"
      fi
    fi
  done < <(find "${MD_FIND_ARGS[@]}" -type f 2>/dev/null)
fi

if ! $DRY_RUN; then
  echo "  Done."
fi

# --- Summary ---

echo ""
if $DRY_RUN; then
  echo "Dry run complete. Run without --dry-run to apply changes."
else
  echo "Rebranding complete. Next steps:"
  echo "  1. Review changes:  git diff"
  echo "  2. Build:           make build"
  echo "  3. Test:            make test"
  echo "  4. Commit:          git add -A && git commit -m 'Rebrand to ${NEW_CLI_NAME}'"
fi
