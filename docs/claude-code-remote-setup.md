# lazycurl — Claude Code Remote Session Setup Guide

This document describes how to set up the `lazycurl` repository for development in a **Claude Code remote session**.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Cloning the repository](#cloning-the-repository)
3. [Starting a Claude Code remote session](#starting-a-claude-code-remote-session)
4. [Project-specific setup](#project-specific-setup)
5. [Installing and configuring openspec](#installing-and-configuring-openspec)
6. [Development workflow](#development-workflow)
7. [Project structure overview](#project-structure-overview)
8. [Custom slash commands (/opsx)](#custom-slash-commands-opsx)
9. [Notes and troubleshooting](#notes-and-troubleshooting)

---

## Prerequisites

You'll need the following tools and accounts.

### Required tools

| Tool | Recommended version | Check command |
|--------|--------------|------------|
| Go | 1.24.7 or later | `go version` |
| curl | 7.70 or later | `curl --version` |
| git | any | `git --version` |
| Claude Code CLI | latest | `claude --version` |
| Node.js / npm | LTS | `node --version` |

> **About the curl version**: lazycurl uses the `-w '%{json}'` option to fetch response metadata. On curl versions below 7.70, a warning is shown and some features are limited.

### Required accounts

- An **Anthropic account** (to use Claude Code)
- GitHub access (to clone the repository)

---

## Cloning the repository

```sh
git clone https://github.com/asunaro276/lazycurl.git
cd lazycurl
```

---

## Starting a Claude Code remote session

### 1. Install the Claude Code CLI (if not already installed)

```sh
npm install -g @anthropic-ai/claude-code
```

### 2. Authenticate

```sh
claude auth login
```

A browser window opens and asks you to authenticate with your Anthropic account.

### 3. Start the remote session

Run the following from the repository root.

```sh
cd ~/sandbox/lazycurl
claude
```

> **Running under WSL**: If you're using this as a remote session into a WSL environment from Windows, run the command above inside the WSL terminal. Claude Code handles a Linux environment on WSL natively.

---

## Project-specific setup

### Installing Go dependencies

```sh
go mod download
```

This downloads all dependencies (Bubble Tea, lipgloss, bubbles, etc.).

### Verifying the build

```sh
go build -o lazycurl ./cmd/lazycurl
./lazycurl
```

### About `.claude/settings.local.json`

`.claude/settings.local.json` is a per-user local permission settings file. By Claude Code convention, it is not committed to the repository (and is not present in this repository either). If you don't want to be asked to approve commands like `openspec` every session, create the following file yourself.

```json
{
  "permissions": {
    "allow": [
      "Bash(openspec *)"
    ]
  }
}
```

You can also add permissions such as `WebFetch(domain:github.com)` or file-management commands like `mv` / `rmdir` as needed.

---

## Installing and configuring openspec

`openspec` is the CLI tool used by this project's **spec-driven development** workflow. It works together with Claude Code to generate specs, proposals, and tasks automatically.

### Installation

The npm package name `openspec` is taken by an unrelated project, so make sure to install the scoped package name instead.

```sh
npm install -g @fission-ai/openspec
```

> After installing, confirm the version with `openspec --version`.

### Verifying it works

```sh
# Check the openspec configuration in this repository
openspec doctor

# List current changes
openspec list
```

### Layout of the `openspec/` directory

```
openspec/
├── config.yaml          # Project context and rule definitions
├── specs/                 # Finalized feature specs (spec.md)
│   ├── tui-shell/
│   ├── collection-storage/
│   ├── curl-execution/
│   ├── environment-variables/
│   └── request-editor/
└── changes/               # In-progress and completed changes (proposal/design/tasks)
    ├── adhoc-request-mode/
    ├── stream-response-body/
    └── archive/            # Archived changes
```

For the current contents of each directory, check the repository directly (`ls openspec/specs` / `ls openspec/changes`).

### Overview of `openspec/config.yaml`

```yaml
schema: spec-driven
context: |
  Language: Japanese
  All artifacts (proposal, tasks, spec, etc.) are written in Japanese
  Technical terms (API, REST, HTTP, TUI, etc.), code, and file paths stay in English

  Tech stack:
    - Language: Go
    - TUI: Bubble Tea + lipgloss + bubbles
rules:
  proposal:
    - Keep it concise
```

---

## Development workflow

### Commonly used commands

```sh
# Build
go build -o lazycurl ./cmd/lazycurl

# Run
./lazycurl

# Test
go test ./...

# Test a specific package
go test ./internal/tui/...

# Sync go.sum
go mod tidy
```

> **About lint**: no additional lint tool such as `golangci-lint` is set up. Only `go vet` and `gofmt` are checked (same in CI).

### Standard feature workflow (via openspec)

1. **Explore the idea** (inside a Claude Code session)

   ```
   /opsx:explore
   ```

2. **Create a change proposal**

   ```
   /opsx:propose
   ```

   `proposal.md`, `design.md`, and `tasks.md` are generated automatically under `openspec/changes/<name>/`.

3. **Implement**

   ```
   /opsx:apply
   ```

   Implements the tasks in `tasks.md` in order.

4. **Archive the change once complete**

   ```
   /opsx:archive
   ```

---

## Project structure overview

```
lazycurl/
├── cmd/
│   └── lazycurl/
│       ├── main.go          # Entry point
│       ├── app.go           # Application core
│       └── app_test.go
├── internal/
│   ├── collection/          # Collection management (.http files)
│   ├── config/              # Configuration file management
│   ├── curlexec/            # curl subprocess execution
│   ├── environment/         # Environments and variable expansion
│   ├── httpfile/            # .http file parser
│   └── tui/                 # Bubble Tea TUI components
│       ├── form/
│       ├── shell/
│       └── styles/
├── openspec/                # Spec management (for the openspec CLI)
│   ├── config.yaml
│   ├── specs/
│   └── changes/
├── .claude/
│   ├── settings.local.json  # Claude Code permission settings (create yourself, not committed)
│   ├── commands/opsx/       # Custom slash command definitions
│   └── skills/              # openspec-related skill definitions
├── go.mod
├── go.sum
└── README.md
```

---

## Custom slash commands (/opsx)

Project-specific commands available inside a Claude Code session.

| Command | Description |
|---------|------|
| `/opsx:propose` | Create a change proposal for a new feature (auto-generates proposal.md, design.md, tasks.md) |
| `/opsx:apply` | Implement a change based on its tasks.md |
| `/opsx:explore` | Thinking-partner mode. No implementation; focused on exploring problems and clarifying requirements |
| `/opsx:archive` | Move a completed change into the archive |

> All of these commands require the `openspec` CLI. Use them after installing it.

---

## Notes and troubleshooting

### `curl` not found / version too old

```sh
# Ubuntu/Debian
sudo apt-get update && sudo apt-get install -y curl

# macOS
brew install curl

# Check the version
curl --version
```

### Go version too old

```sh
# Install the latest version from the official Go site
# https://go.dev/dl/

# Or use a version manager such as mise / asdf
mise use go@1.24.7
```

### `openspec` command not found

```sh
# Check whether npm's global path is included in PATH
npm config get prefix
# Example output: /home/user/.npm-global

# Add it to .zshrc or .bashrc
export PATH="$HOME/.npm-global/bin:$PATH"
source ~/.zshrc
```

### Claude Code keeps asking to approve the openspec command every session

See the [Project-specific setup](#project-specific-setup) section on `.claude/settings.local.json` and create a config that allows `Bash(openspec *)`.

### PATH issues under WSL

WSL inherits the Windows PATH by default. If Go or Node.js are installed on the Windows side, it's recommended to install them separately inside WSL as well.

```sh
# Install Go inside WSL (example using mise)
curl https://mise.run | sh
mise use go@1.24.7 node@lts
```

### Errors from `go mod tidy`

```sh
# When fetching modules over the network
export GOPROXY=https://proxy.golang.org,direct
go mod tidy
```

---

## References

- [lazycurl GitHub repository](https://github.com/asunaro276/lazycurl)
- [Claude Code documentation](https://docs.anthropic.com/claude-code)
- [Bubble Tea documentation](https://github.com/charmbracelet/bubbletea)
- [openspec CLI](https://www.npmjs.com/package/@fission-ai/openspec)
