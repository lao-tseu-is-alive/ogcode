<div align="center">

# Ogcode

**One binary. Zero cloud. An agentic coding workbench that plans, remembers, and ships in parallel.**

[![Discord](https://img.shields.io/discord/1373677337985056828?label=Discord&logo=discord&logoColor=white&color=5865F2)](https://discord.gg/JQP9t8y2Zv)
[![Release](https://img.shields.io/github/v/release/prasenjeet-symon/ogcode?label=Release&style=flat)](https://github.com/prasenjeet-symon/ogcode/releases)
[![License](https://img.shields.io/github/license/prasenjeet-symon/ogcode?label=License&color=green)](LICENSE)
[![Stars](https://img.shields.io/github/stars/prasenjeet-symon/ogcode?style=social)](https://github.com/prasenjeet-symon/ogcode)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)](https://react.dev)
[![MIT License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

</div>

![Ogcode Demo](assets/ogcode_intro.gif)

> **One binary. Zero cloud dependencies. Your codebase, your models, your rules.**

Ogcode is an agentic coding assistant that runs entirely on your machine — a single Go binary with an embedded React web UI. It doesn't just suggest code; it **understands your entire codebase**, **plans complex features with you**, and **executes them across parallel git branches** — all while keeping your code local and your data private.

Unlike IDE-locked assistants (Cursor, Copilot) or cloud-only services (Claude Code), Ogcode is **browser-native, self-hosted, and model-agnostic**. Use Claude, GPT, OpenRouter, or local Ollama models — switch anytime from the UI. No subscriptions. No vendor lock-in. No code leaving your machine except to your chosen LLM provider.

---

## What Makes Ogcode Different

| | Ogcode | Cursor | Claude Code | Copilot | Aider |
|---|---|---|---|---|---|
| **Interface** | Web UI (any editor) | VS Code fork | Terminal | IDE extension | Terminal |
| **Self-Hosted** | Single binary, zero deps | Cloud-required | Cloud-only | Cloud-only | Open source |
| **Parallel Tasks** | Git worktrees, auto-PRs | Cloud agents | Subagents | Single agent | Sequential |
| **Plan Mode** | Kanban + effort estimates | Agents window | Architect mode | Prompt-based | /architect |
| **Persistent Memory** | Knowledge graph + Call graph | Session-only | CLAUDE.md | None | None |
| **Model Choice** | Claude, GPT, OpenRouter, Ollama | Built-in + custom | Claude only | MS-managed | Any endpoint |
| **Cost** | BYOK (tokens only) | $20–$40/mo | $20–$100/mo | $19–$39/mo | Free (BYOK) |
| **License** | **MIT** | Proprietary | Proprietary | Proprietary | Apache-2.0 |

**Ogcode is the only agentic coding assistant that combines:**

- **Browser-native UI** — works with Vim, Emacs, VS Code, JetBrains, or any editor
- **Plan Mode with Kanban** — collaboratively plan features, lock them into tasks, watch them execute in parallel
- **Git-native parallel execution** — each task gets its own isolated branch. Auto-commits. Auto-PRs. Zero merge conflicts.
- **Agentic Knowledge Graph** — persistent Topic→Concept→Fact memory with ~70% token savings and infinite context
- **Deep Research Agent** — built-in web search that fetches docs, changelogs, and security advisories for your agent
- **Single binary deployment** — one `ogcode` executable. No Docker. No Node server. No external DB.

---

## Features

- **Build Mode + Plan Mode** — Chat with a coding agent in real-time, or collaboratively plan complex features with effort estimates and dependency graphs
- **Parallel Task Execution** — Multiple independent tasks run simultaneously across isolated git worktree branches. Ship entire features in parallel.
- **Agentic Session Memory** — Infinite context via a persistent knowledge graph. ~70% token savings on long sessions. Never lose context.
- **Knowledge Graph + Call Graph** — Semantic memory of your codebase (Topic→Concept→Fact) plus function-level call relationships for intelligent navigation
- **Multi-Provider LLM Support** — Anthropic Claude, OpenAI GPT, OpenRouter, or local Ollama models. Switch anytime from the UI.
- **Deep Research Agent (v0.8.0)** — Built-in `deep_search` tool searches the web, fetches pages, and synthesizes cited research for your agent
- **Kanban Board** — Visual task board with S/M/L/XL effort estimates, complexity scores, and dependency chains
- **Permission-Based Safety** — Destructive operations (write, edit, bash) require explicit approval per-tool; read-only tools auto-approve
- **PDF Support** — Read and index PDF documentation directly in the agent
- **Codebase Map** — Semantic index of your entire project for intelligent file discovery
- **Single Binary, Self-Contained** — One Go binary with embedded React frontend. Zero external server dependencies.
- **MCP Extensibility** — Model Context Protocol support for custom tools and integrations

---

## Table of Contents

- [Quick Start](#quick-start)
- [Why Ogcode?](#why-ogcode)
- [System Requirements](#system-requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Usage](#usage)
- [Remote Deployment](#remote-deployment)
- [The Plan Mode Workflow](#the-plan-mode-workflow)
- [Agentic Session Memory](#agentic-session-memory)
- [Architecture](#architecture)
- [Roadmap](#roadmap)
- [Community](#community)
- [Contributing](#contributing)
- [Security](#security)
- [License](#license)

---

## Quick Start

```bash
# macOS / Linux — one-line install
curl -fsSL http://ogcode.xyz/install.sh | sh

# Set your API key (or use Ollama for local models)
export ANTHROPIC_API_KEY=sk-ant-...

# Start coding
ogcode
```

Opens at `http://localhost:8080`. That's it. No config files. No Docker. No IDE extension.

---

## Why Ogcode?

### "I already use Cursor / Copilot / Claude Code..."

**Cursor** is great — but it's a fork of VS Code. If you use Vim, Emacs, or JetBrains, you're out of luck. And its parallel agents run in the cloud, not on your machine.

**Claude Code** is powerful — but it's terminal-only, Anthropic-only, and cloud-only. No web UI. No plan mode. No persistent knowledge graph.

**GitHub Copilot** is everywhere — but it's a Microsoft service. Your code analysis happens in the cloud. No parallel execution. No formal planning.

**Aider** is excellent — but it's terminal-only, sequential-only, and has no persistent memory graph or visual planning board.

**Ogcode gives you what none of the above do:**

- A **web UI** that works with *any* editor
- **Formal Plan Mode** with visual Kanban and parallel execution
- **Persistent knowledge graph** that survives across sessions
- **Single-binary self-hosting** with zero cloud dependencies
- **Full model freedom** — Claude today, GPT tomorrow, local Llama next week

---

## System Requirements

| Platform | Minimum |
|----------|---------|
| **Operating System** | macOS, Linux, or Windows |
| **Go** | 1.22+ (for `go install` only) |
| **Git** | 2.34+ (required for worktree support) |
| **CPU** | Any modern x86_64 or arm64 processor |
| **Memory** | 512 MB free RAM |

> **Note:** An LLM API key or local Ollama installation is required. See [Configuration](#configuration).

---

## Installation

### macOS / Linux

**Via Homebrew (recommended):**

```bash
brew tap prasenjeet-symon/ogcode
brew install ogcode
```

**Via curl (one-liner):**

```bash
curl -fsSL http://ogcode.xyz/install.sh | sh
```

Auto-detects platform, downloads the latest release, and installs to `/usr/local/bin`.

### Windows

```powershell
irm http://ogcode.xyz/install.ps1 | iex
```

Downloads the latest release, extracts to `%LOCALAPPDATA%\ogcode`, and adds to PATH.

**Via winget:**

```powershell
winget install prasenjeet-symon.ogcode
```

### Go Install

```bash
go install github.com/prasenjeet-symon/ogcode@latest
```

### Docker

```bash
docker run -p 8080:8080 -v $(pwd):/workspace -w /workspace ghcr.io/prasenjeet-symon/ogcode:latest
```

---

## Configuration

Ogcode auto-detects available AI providers from environment variables. No config files required.

### Required: AI Provider

Set at least one API key (or use Ollama):

| Variable | Provider |
|----------|----------|
| `ANTHROPIC_API_KEY` | Anthropic (Claude) |
| `OPENAI_API_KEY` | OpenAI (GPT) |
| `OPENROUTER_API_KEY` | OpenRouter |
| `OLLAMA_BASE_URL` | Ollama (local / cloud URL) |

#### Ollama (local models — free, private)

```bash
# macOS / Linux — auto-detected if ollama is installed
ollama serve
ogcode

# Or explicit on any OS:
export OLLAMA_BASE_URL=http://localhost:11434/v1
ogcode
```

Available models: `qwen3`, `codellama`, `llama3.1`, `deepseek-coder-v2`, `mistral`, and any model you've pulled.

### Optional: Agentic Memory

Enable infinite-context memory across sessions:

```bash
export OGCODE_AGENTIC_MEMORY_MODE=true
```

### Optional: Web Search (v0.8.0)

Give your agent the ability to research current documentation:

```bash
export OGCODE_SEARCH_ENABLED=true
```

---

## Usage

### Start in Build Mode (default)

```bash
ogcode
```

Chat with the agent, ask it to read files, write code, run commands, or search the codebase.

### Start in Plan Mode

```bash
ogcode plan
```

Describe what you want to build. The planning agent reads your codebase, discusses the approach, and breaks it into tasks with dependencies and effort estimates. Lock the plan → tasks become a Kanban board → execute in parallel.

### Custom port

```bash
ogcode -p 3000
ogcode plan -p 3000
```

---

## Remote Deployment

Ogcode is just an HTTP server — host it on a remote machine and access it from anywhere via a browser.

### Quick Start — Expose the Port

```bash
docker run -p 8080:8080 \
  -v ~/.ogcode:/root/.ogcode \
  -v $(pwd):/workspace -w /workspace \
  ghcr.io/prasenjeet-symon/ogcode:latest
```

Then open `http://<your-server-ip>:8080` from any browser.

### Production — Behind a Reverse Proxy with HTTPS

```nginx
# nginx config
server {
    listen 443 ssl;
    server_name ogcode.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Upgrade $http_upgrade;     # WebSocket support
        proxy_set_header Connection "upgrade";
    }
}
```

Access via `https://ogcode.yourdomain.com` — encrypted and clean.

### Docker Compose (Production-Friendly)

```yaml
services:
  ogcode:
    image: ghcr.io/prasenjeet-symon/ogcode:latest
    volumes:
      - ~/.ogcode:/root/.ogcode
    ports:
      - "127.0.0.1:8080:8080"   # only localhost — nginx handles public access
    restart: unless-stopped
```

### ⚠️ Security Considerations

Ogcode is a coding agent that can execute shell commands, read/write files, and modify your system. **Never expose it to the public internet without authentication.**

| Risk | Mitigation |
|------|-----------|
| Anyone can hit port 8080 | Bind to `127.0.0.1` + use a reverse proxy |
| No auth on the web UI | Add HTTP Basic Auth in nginx or use a VPN |
| Full shell access via the agent | Run in a restricted environment (Docker, VM) |

Recommended approaches:

1. **SSH tunnel** — Most secure, zero config:
   ```bash
   # On your laptop:
   ssh -L 8080:localhost:8080 user@your-server
   # Then open http://localhost:8080 in your browser
   ```
2. **nginx + HTTP Basic Auth** — Simple password gate:
   ```bash
   htpasswd -c /etc/nginx/.htpasswd your_username
   ```
3. **Cloudflare Tunnel** — Zero open ports, add Cloudflare Access for auth
4. **VPN (WireGuard / Tailscale)** — Private network, no public exposure

---

## The Plan Mode Workflow

```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  1. Describe │ → │  2. Lock    │ → │  3. Review  │ → │  4. Execute  │
│   your goal  │    │   the plan  │    │  Kanban board│    │   in parallel│
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                                          │
                    ┌─────────────┐    ┌─────────────┐     ▼
                    │  6. Retry   │ ← │  5. Complete│ ← ┌─────────────┐
                    │  if needed  │    │  auto-PR    │   │  Task runs  │
                    └─────────────┘    └─────────────┘   │  in isolated │
                                                          │  git branch  │
                                                          └─────────────┘
```

1. **Describe** — Open a new plan and describe your goal. The planning agent reads your codebase and refines the approach with you.
2. **Lock** — When ready, lock the plan. The agent generates a structured task breakdown with effort (S/M/L/XL), complexity, and dependencies.
3. **Review** — View tasks in the Kanban board. Drag, reorder, or start tasks individually.
4. **Execute** — Start tasks. Each one gets its own git branch and isolated agent session. Independent tasks run in **parallel**.
5. **Complete** — Finished tasks auto-commit and auto-create pull requests.
6. **Retry** — If a task fails, retry it. The stale branch is removed and the task starts fresh.

Plans are archived as markdown in `.ogcode/archives/` once complete.

---

## Agentic Session Memory

![Agentic Memory Demo](assets/agentic_memory.gif)

Traditional assistants send the *entire conversation history* to the LLM every turn — expensive and quickly hitting token limits. Ogcode's **Agentic Session Memory** extracts, stores, and retrieves only the relevant context for each query.

### Key Benefits

| Feature | Impact |
|---------|--------|
| **~70% Token Savings** | Drastically reduced API costs on long sessions |
| **Infinite Context** | No practical limit on session length or codebase size |
| **Higher Accuracy** | Only relevant memories are retrieved per query |

### Token Savings Example

| Session Length | Traditional | With Agentic Memory | Savings |
|----------------|-------------|---------------------|---------|
| 50 messages | ~25K tokens | ~8K tokens | **68%** |
| 200 messages | ~100K tokens | ~28K tokens | **72%** |
| 1000 messages | ~500K tokens | ~120K tokens | **76%** |

### How It Works

Ogcode maintains a persistent **Topic → Concept → Fact** hierarchy with vector embeddings, plus a **function-level Call Graph** for codebase navigation. This knowledge graph survives across sessions — your agent remembers your codebase structure, your conventions, and your past decisions.

```
Topic: "Ogcode Authentication"
  └─ Concept: "JWT Middleware"
     └─ Fact: "Token validation lives in internal/auth/jwt.go:47"
     └─ Fact: "Refresh tokens expire after 7 days (config: AUTH_REFRESH_TTL)"
  └─ Concept: "OAuth Flow"
     └─ Fact: "GitHub OAuth uses PKCE, implemented in internal/auth/oauth.go"
```

Enable it with:

```bash
export OGCODE_AGENTIC_MEMORY_MODE=true
```

---

## Architecture

Ogcode is a single Go binary that embeds a React web UI and runs its own HTTP server.

```
┌─────────────┐     REST + SSE      ┌──────────────┐
│  Web UI     │ ◄────────────────► │  Go Server   │
│  (React)    │                    │  (port 8080) │
└─────────────┘                    └──────┬───────┘
                                          │
                    ┌─────────────────────┼─────────────────────┐
                    ▼                     ▼                     ▼
            ┌─────────────┐    ┌─────────────┐    ┌──────────────┐
            │ Agent Loop  │    │  SQLite DB  │    │ LLM Provider │
            │ (Claude,    │    │ (workspace  │    │ (Anthropic,  │
            │  GPT, etc.) │    │  + config)  │    │ OpenAI, ...) │
            └─────────────┘    └─────────────┘    └──────────────┘
                    │
                    ▼
            ┌─────────────┐    ┌─────────────┐    ┌──────────────┐
            │  Knowledge  │    │   Call      │    │   Search    │
            │   Graph     │    │   Graph     │    │   Bridge    │
            │  (Memory)   │    │ (Code Rel)  │    │  (Web/JS)   │
            └─────────────┘    └─────────────┘    └──────────────┘
```

Key components:

- **Agent Loop** — Streaming LLM chat with tool execution (bash, read, write, edit, glob, grep, memory_recall, callgraph, deep_search)
- **Session Store** — SQLite database for conversations, plans, tasks, and permissions
- **Git Worktrees** — Each task gets an isolated branch so multiple agents work in parallel
- **Knowledge Graph** — Persistent semantic memory with vector embeddings
- **Call Graph** — Function-level code relationship tracking
- **Search Bridge** — Playwright-based headless Chrome for web research (v0.8.0)

---

## Roadmap

- [ ] **Advanced Task Planning & Parallel Execution** — Enhanced plan decomposition with manual/automatic agent assignment to tasks
- [ ] **AI Daily Standups** — Voice-enabled meetings where agents report progress and discuss their work
- [ ] **Ogland Integration** — Connect external services (Slack, Email, Jira) for planning-phase use
- [ ] **Agentic Deployment** — End-to-end agentic deployment for major cloud providers, starting with AWS

---

## Community

Join the Ogcode community on Discord:

- Ask questions and get help
- Share feedback and feature ideas
- Stay up to date with releases and announcements

[**Join us on Discord →**](https://discord.gg/JQP9t8y2Zv)

**Star us on GitHub** — it helps more developers discover Ogcode!

---

## Contributing

We welcome contributions! Whether it's bug fixes, features, or documentation:

1. **Fork** the repository
2. Create a feature branch: `git checkout -b feature/my-improvement`
3. Make your changes and add tests if applicable
4. Submit a pull request

Please ensure your code follows the existing Go style and passes `go test ./...`.

---

## Security

- **Destructive operations require approval** — The agent cannot write files or run shell commands without your explicit permission. Approve once or always per tool.
- **Git worktree isolation** — Each task runs in a separate git worktree, preventing accidental contamination of your working branch.
- **No data sent to third parties** — Your codebase is analyzed locally. Only conversation text is sent to your chosen LLM provider.
- **Local-first** — All data (sessions, plans, memory) is stored in local SQLite databases.

For security concerns, please open an issue or reach out on Discord.

---

## License

MIT License — see [LICENSE](LICENSE) for details.

---

<div align="center">

**Made with care by the Ogcode team and contributors**

[Star on GitHub](https://github.com/prasenjeet-symon/ogcode) · [Discord](https://discord.gg/JQP9t8y2Zv) · [Docs](docs/OUTLINE.md)

</div>
