# Enterprise Setup Guide

This guide covers distributing AgentX internally via Sonatype Nexus. It applies to organizations that cannot use public GitHub releases.

## Overview

AgentX supports enterprise distribution through two Nexus repository types:

- **Nexus raw (hosted)** -- stores Go binaries, checksums, and the install script
- **Nexus npm (hosted)** -- stores `@agentx/*` Node.js packages

A CI workflow (`.github/workflows/release-nexus.yaml`) automatically uploads artifacts to both repositories after each release.

## Nexus Raw Repository Setup

Create a **raw (hosted)** repository in Nexus for binary distribution.

1. In Nexus, go to **Repositories > Create Repository > raw (hosted)**
2. Name: `agentx-releases`
3. Blob store: default or dedicated
4. Deployment policy: **Allow redeploy** (the `install.sh` and `latest/version.txt` files are overwritten on each release)

After setup, the repository will be available at:
```
https://nexus.corp.com/repository/agentx-releases/
```

Artifacts are organized as:
```
agentx-releases/
  install.sh                          # Always-latest install script
  latest/
    version.txt                       # Contains the latest release tag (e.g., v1.2.0)
  v1.2.0/
    agentx_linux_amd64.tar.gz
    agentx_linux_arm64.tar.gz
    agentx_darwin_amd64.tar.gz
    agentx_darwin_arm64.tar.gz
    agentx_windows_amd64.zip
    checksums.txt
```

## Nexus npm Registry Setup

Create an **npm (hosted)** repository for `@agentx/*` packages.

1. In Nexus, go to **Repositories > Create Repository > npm (hosted)**
2. Name: `npm-internal`
3. Blob store: default or dedicated

Developers configure their `.npmrc` to use this registry:
```
@agentx:registry=https://nexus.corp.com/repository/npm-internal/
//nexus.corp.com/repository/npm-internal/:_authtoken=<token>
```

## CI Secrets Configuration

The `release-nexus.yaml` workflow requires these GitHub repository settings:

**Secrets** (Settings > Secrets and variables > Actions > Secrets):

| Secret | Description |
|--------|-------------|
| `NEXUS_USER` | Nexus service account username for raw repository uploads |
| `NEXUS_TOKEN` | Nexus service account password/token for raw repository uploads |
| `NEXUS_NPM_TOKEN` | Nexus npm auth token for `@agentx/*` package publishing |

**Variables** (Settings > Secrets and variables > Actions > Variables):

| Variable | Example | Description |
|----------|---------|-------------|
| `NEXUS_URL` | `https://nexus.corp.com/repository` | Base URL for Nexus (without trailing slash) |
| `NEXUS_NPM_URL` | `https://nexus.corp.com/repository/npm-internal/` | Full URL to the npm hosted repository |

## Installing from Nexus

### Using the install script

The install script supports the `AGENTX_MIRROR` environment variable:

```bash
# Install latest from Nexus
AGENTX_MIRROR="https://nexus.corp.com/repository/agentx-releases" \
  curl -sSL https://nexus.corp.com/repository/agentx-releases/install.sh | bash

# Install a specific version
AGENTX_VERSION=1.2.0 \
AGENTX_MIRROR="https://nexus.corp.com/repository/agentx-releases" \
  curl -sSL https://nexus.corp.com/repository/agentx-releases/install.sh | bash
```

When `AGENTX_MIRROR` is set and no version is pinned, the script resolves the latest version from `<mirror>/latest/version.txt` instead of the GitHub API.

### Direct download

```bash
curl -sSL "https://nexus.corp.com/repository/agentx-releases/v1.2.0/agentx_darwin_arm64.tar.gz" \
  | tar xz -C ~/.local/bin agentx
```

## Internal Homebrew Tap

Organizations using Homebrew can create an internal tap that points to Nexus-hosted binaries.

1. Create a Git repository (e.g., `github.corp.com:platform-team/homebrew-tools.git`)
2. Add a formula at `Formula/agentx.rb`:

```ruby
class Agentx < Formula
  desc "Developer infrastructure for AI agent configurations"
  homepage "https://github.com/jefelabs/agentx"
  version "1.2.0"

  on_macos do
    if Hardware::CPU.arm?
      url "https://nexus.corp.com/repository/agentx-releases/v1.2.0/agentx_darwin_arm64.tar.gz"
      sha256 "<sha256>"
    else
      url "https://nexus.corp.com/repository/agentx-releases/v1.2.0/agentx_darwin_amd64.tar.gz"
      sha256 "<sha256>"
    end
  end

  on_linux do
    if Hardware::CPU.arm?
      url "https://nexus.corp.com/repository/agentx-releases/v1.2.0/agentx_linux_arm64.tar.gz"
      sha256 "<sha256>"
    else
      url "https://nexus.corp.com/repository/agentx-releases/v1.2.0/agentx_linux_amd64.tar.gz"
      sha256 "<sha256>"
    end
  end

  def install
    bin.install "agentx"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/agentx version")
  end
end
```

3. Update the formula version, URLs, and sha256 hashes after each release
4. Developers install via:

```bash
brew tap corp/tools git@github.corp.com:platform-team/homebrew-tools.git
brew install agentx
```

## Mirror Configuration for Self-Update

The `agentx update` command checks for new versions. To point it at Nexus instead of GitHub:

```yaml
# ~/.agentx/config.yaml
mirror: https://nexus.corp.com/repository/agentx-releases
```

With this setting, `agentx update` downloads new versions from the Nexus mirror. Without it, it defaults to GitHub Releases.
