<div align="center">

# Ogcode

**The self-hosted agentic coding workbench.**

It plans with you, remembers your codebase, and ships features in parallel — from a single binary that never leaves your machine.

<br/>

[![Discord](https://img.shields.io/discord/1373677337985056828?label=Discord&logo=discord&logoColor=white&color=5865F2)](https://discord.gg/JQP9t8y2Zv)
[![Release](https://img.shields.io/github/v/release/prasenjeet-symon/ogcode?label=Release&style=flat)](https://github.com/prasenjeet-symon/ogcode/releases)
[![License](https://img.shields.io/github/license/prasenjeet-symon/ogcode?label=License&color=green)](LICENSE)
[![Stars](https://img.shields.io/github/stars/prasenjeet-symon/ogcode?style=social)](https://github.com/prasenjeet-symon/ogcode)

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white)](https://go.dev)
[![SolidJS](https://img.shields.io/badge/SolidJS-1.9-2F2E82?logo=solidjs&logoColor=white)](https://www.solidjs.com)
[![MIT License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

<br/>

![Ogcode Demo](assets/ogcode_intro.gif)

<br/>

[Quick Start](#quick-start) · [Why Ogcode](#why-ogcode) · [Documentation](docs/OUTLINE.md) · [Discord](https://discord.gg/JQP9t8y2Zv)

</div>

---

Ogcode is an agentic coding assistant that runs entirely on your machine — a single Go binary with an embedded SolidJS web UI. It doesn't just suggest code. It **understands your whole codebase**, **plans complex features with you**, and **executes them across parallel git branches** — while your code stays local and private.

Unlike IDE-locked assistants (Cursor, Copilot) or cloud-only services (Claude Code), Ogcode is **browser-native, self-hosted, and model-agnostic**. Use Claude, GPT, OpenRouter, or local Ollama models, and switch anytime from the UI. No subscriptions. No vendor lock-in. Nothing leaves your machine except the prompts you send to your chosen provider.

```bash
curl -fsSL http://ogcode.xyz/install.sh | sh && ogcode
```

---

## How Ogcode Compares

|                       | Ogcode                        | Cursor          | Claude Code   | Copilot       | Aider          |
| --------------------- | ----------------------------- | --------------- | ------------- | ------------- | -------------- |
| **Interface**         | Web UI (any editor)           | VS Code fork    | Terminal      | IDE extension | Terminal       |
| **Self-Hosted**       | Single binary, zero deps      | Cloud-required  | Cloud-only    | Cloud-only    | Open source    |
| **Parallel Tasks**    | Git worktrees, auto-PRs       | Cloud agents    | Subagents     | Single agent  | Sequential     |
| **Plan Mode**         | Kanban + effort estimates     | Agents window   | Architect mode| Prompt-based  | `/architect`   |
| **Persistent Memory** | Knowledge graph + Call graph  | Session-only    | CLAUDE.md     | None          | None           |
| **Model Choice**      | Claude, GPT, OpenRouter, Ollama | Built-in + custom | Claude only | MS-managed    | Any endpoint   |
| **Cost**              | BYOK (tokens only)            | $20–$40/mo      | $20–$100/mo   | $19–$39/mo    | Free (BYOK)    |
| **License**           | **MIT**                       | Proprietary     | Proprietary   | Proprietary   | Apache-2.0     |

Ogcode is the only agentic coding assistant that combines a **browser-native UI** (works with Vim, Emacs, VS Code, JetBrains, or any editor), a **formal Plan Mode** with a visual Kanban board, **git-native parallel execution** that gives every task its own isolated branch with auto-commits and auto-PRs, a **persistent knowledge graph** for long-term memory, and **single-binary self-hosting** with zero cloud dependencies.

---

## Features

- **Build Mode + Plan Mode** — Chat with a coding agent in real time, or collaboratively plan complex features with effort estimates and dependency graphs.
- **Parallel Task Execution** — Independent tasks run simultaneously across isolated git worktree branches. Ship entire features in parallel.
- **Agentic Session Memory** — Infinite context via a persistent knowledge graph, with ~70% token savings on long sessions.
- **Knowledge Graph + Call Graph** — Semantic memory of your codebase (Topic → Concept → Fact) plus function-level call relationships for intelligent navigation.
- **Multi-Provider LLM Support** — Anthropic Claude, OpenAI GPT, OpenRouter, or local Ollama models. Switch anytime from the UI.
- **Deep Research Agent** — A built-in `deep_search` tool searches the web, fetches pages, and synthesizes cited research for your agent.
- **Kanban Board** — Visual task board with S/M/L/XL effort estimates, complexity scores, and dependency chains.
- **Permission-Based Safety** — Destructive operations (write, edit, bash) require explicit approval per tool; read-only tools auto-approve.
- **PDF Support** — Read and index PDF documentation directly in the agent.
- **Codebase Map** — A semantic index of your entire project for intelligent file discovery.
- **Single Binary, Self-Contained** — One Go binary with an embedded SolidJS frontend. Zero external server dependencies.
- **MCP Extensibility** — Model Context Protocol support for custom tools and integrations.
- **Rich Visual Output** — Agents render Mermaid diagrams, LaTeX math, Plotly charts, and full HTML/CSS/JS interactive content directly in the chat, with viewport-aware responsive design.

---

## Table of Contents

- [Quick Start](#quick-start)
- [Why Ogcode](#why-ogcode)
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

Opens at `http://localhost:9595`. That's it — no config files, no Docker, no IDE extension.

---

## Why Ogcode

**Cursor** is great, but it's a fork of VS Code. If you use Vim, Emacs, or JetBrains, you're out of luck — and its parallel agents run in the cloud, not on your machine.

**Claude Code** is powerful, but it's terminal-only, Anthropic-only, and cloud-only. No web UI, no plan mode, no persistent knowledge graph.

**GitHub Copilot** is everywhere, but it's a Microsoft service. Your code analysis happens in the cloud, with no parallel execution and no formal planning.

**Aider** is excellent, but it's terminal-only and sequential, with no persistent memory graph or visual planning board.

Ogcode gives you what none of the above do:

- A **web UI** that works with *any* editor
- **Formal Plan Mode** with a visual Kanban board and parallel execution
- A **persistent knowledge graph** that survives across sessions
- **Single-binary self-hosting** with zero cloud dependencies
- **Full model freedom** — Claude today, GPT tomorrow, local Llama next week

---

## System Requirements

| Platform             | Minimum                                   |
| -------------------- | ----------------------------------------- |
| **Operating System** | macOS, Linux, or Windows                  |
| **Go**               | 1.22+ (for `go install` only)             |
| **Git**              | 2.34+ (required for worktree support)     |
| **CPU**              | Any modern x86_64 or arm64 processor      |
| **Memory**           | 512 MB free RAM                           |

An LLM API key or a local Ollama installation is required. See [Configuration](#configuration).

---

## Installation

### macOS / Linux

Via Homebrew (recommended):

```bash
brew tap prasenjeet-symon/ogcode
brew install ogcode
```

Via curl (one-liner):

```bash
curl -fsSL http://ogcode.xyz/install.sh | sh
```

Auto-detects your platform, downloads the latest release, and installs to `/usr/local/bin`.

### Windows

```powershell
irm http://ogcode.xyz/install.ps1 | iex
```

Downloads the latest release, extracts to `%LOCALAPPDATA%\ogcode`, and adds it to your PATH.

Via winget:

```powershell
winget install prasenjeet-symon.ogcode
```

### Go Install

```bash
go install github.com/prasenjeet-symon/ogcode@latest
```

### Docker

```bash
docker run -p 9595:9595 -v $(pwd):/workspace -w /workspace ghcr.io/prasenjeet-symon/ogcode:latest
```

---

## Configuration

Ogcode auto-detects available AI providers from environment variables. No config files required.

### AI Provider

Set at least one API key (or use Ollama):

| Variable             | Provider                  |
| -------------------- | ------------------------- |
| `ANTHROPIC_API_KEY`  | Anthropic (Claude)        |
| `OPENAI_API_KEY`     | OpenAI (GPT)              |
| `OPENROUTER_API_KEY` | OpenRouter                |
| `OLLAMA_BASE_URL`    | Ollama (local / cloud URL)|

### Ollama (local models — free, private)

```bash
# macOS / Linux — auto-detected if ollama is installed
ollama serve
ogcode

# Or explicit on any OS:
export OLLAMA_BASE_URL=http://localhost:11434/v1
ogcode
```

Available models: `qwen3`, `codellama`, `llama3.1`, `deepseek-coder-v2`, `mistral`, and any model you've pulled.

### Agentic Memory (optional)

Enable infinite-context memory across sessions:

```bash
export OGCODE_AGENTIC_MEMORY_MODE=true
```

### Web Search (optional)

Give your agent the ability to research current documentation:

```bash
export OGCODE_SEARCH_ENABLED=true
```

---

## Usage

### Build Mode (default)

```bash
ogcode
```

Chat with the agent — ask it to read files, write code, run commands, or search the codebase.

### Plan Mode

```bash
ogcode plan
```

Describe what you want to build. The planning agent reads your codebase, discusses the approach, and breaks it into tasks with dependencies and effort estimates. Lock the plan, and the tasks become a Kanban board you can execute in parallel.

### Custom port

```bash
ogcode -p 3000
ogcode plan -p 3000
```

---

## Remote Deployment

Ogcode is just an HTTP server — host it on a remote machine and reach it from any browser.

### Expose the port

```bash
docker run -p 9595:9595 \
  -v ~/.ogcode:/root/.ogcode \
  -v $(pwd):/workspace -w /workspace \
  ghcr.io/prasenjeet-symon/ogcode:latest
```

Then open `http://<your-server-ip>:9595` from any browser.

### Behind a reverse proxy with HTTPS

```nginx
# nginx config
server {
    listen 443 ssl;
    server_name ogcode.yourdomain.com;

    location / {
        proxy_pass http://127.0.0.1:9595;
        proxy_set_header Upgrade $http_upgrade;     # WebSocket support
        proxy_set_header Connection "upgrade";
    }
}
```

Access via `https://ogcode.yourdomain.com` — encrypted and clean.

### Docker Compose

```yaml
services:
  ogcode:
    image: ghcr.io/prasenjeet-symon/ogcode:latest
    volumes:
      - ~/.ogcode:/root/.ogcode
    ports:
      - "127.0.0.1:9595:9595"   # only localhost — nginx handles public access
    restart: unless-stopped
```

### Security considerations

Ogcode is a coding agent that can execute shell commands, read and write files, and modify your system. **Never expose it to the public internet without authentication.**

| Risk                          | Mitigation                                   |
| ----------------------------- | -------------------------------------------- |
| Anyone can hit port 9595      | Bind to `127.0.0.1` + use a reverse proxy    |
| No auth on the web UI         | Add HTTP Basic Auth in nginx, or use a VPN   |
| Full shell access via the agent | Run in a restricted environment (Docker, VM) |

Recommended approaches:

1. **SSH tunnel** — most secure, zero config:
   ```bash
   # On your laptop:
   ssh -L 9595:localhost:9595 user@your-server
   # Then open http://localhost:9595 in your browser
   ```
2. **nginx + HTTP Basic Auth** — a simple password gate:
   ```bash
   htpasswd -c /etc/nginx/.htpasswd your_username
   ```
3. **Cloudflare Tunnel** — zero open ports; add Cloudflare Access for auth.
4. **VPN (WireGuard / Tailscale)** — a private network with no public exposure.

### Rich output

Agents can produce rich visual content directly in the chat — not just plain text:

| Format               | Syntax                  | Use For                                                  |
| -------------------- | ----------------------- | -------------------------------------------------------- |
| **Mermaid diagrams** | ` ```mermaid `          | Flows, architectures, sequences, ER diagrams             |
| **LaTeX math**       | `$...$` or `$$...$$`     | Mathematical formulas and equations                      |
| **Plotly charts**    | ` ```plotly `           | Bar, line, scatter, pie, heatmap, and more               |
| **Rough diagrams**   | ` ```rough `            | Hand-drawn style 2D diagrams                             |
| **HTML/CSS/JS**      | ` ```html `             | Interactive dashboards, styled tables, animations        |

HTML blocks render in a **sandboxed iframe** — scripts run in isolation with no access to the parent page. The agent is given your viewport dimensions so it can design responsive content that fits your screen.

---

## The Plan Mode Workflow

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  1. Describe │ → │  2. Lock      │ → │  3. Review    │ → │  4. Execute   │
│   your goal   │    │   the plan    │    │  Kanban board │    │   in parallel │
└──────────────┘    └──────────────┘    └──────────────┘    └──────┬───────┘
                                                                    │
                    ┌──────────────┐    ┌──────────────┐    ┌───────▼──────┐
                    │  6. Retry    │ ← │  5. Complete  │ ← │  Task runs    │
                    │  if needed    │    │  auto-PR      │    │  in isolated  │
                    └──────────────┘    └──────────────┘    │  git branch   │
                                                            └──────────────┘
```

1. **Describe** — Open a new plan and describe your goal. The planning agent reads your codebase and refines the approach with you.
2. **Lock** — When ready, lock the plan. The agent generates a structured task breakdown with effort (S/M/L/XL), complexity, and dependencies.
3. **Review** — View tasks in the Kanban board. Drag, reorder, or start tasks individually.
4. **Execute** — Start tasks. Each gets its own git branch and isolated agent session. Independent tasks run in **parallel**.
5. **Complete** — Finished tasks auto-commit and open pull requests.
6. **Retry** — If a task fails, retry it. The stale branch is removed and the task starts fresh.

Plans are archived as markdown in `.ogcode/archives/` once complete.

---

## Agentic Session Memory

![Agentic Memory Demo](assets/agentic_memory.gif)

Traditional assistants send the *entire conversation history* to the LLM every turn — expensive, and quick to hit token limits. Ogcode's **Agentic Session Memory** extracts, stores, and retrieves only the context relevant to each query.

| Benefit              | Impact                                              |
| -------------------- | --------------------------------------------------- |
| **~70% token savings** | Drastically reduced API costs on long sessions    |
| **Infinite context** | No practical limit on session length or codebase size |
| **Higher accuracy**  | Only relevant memories are retrieved per query      |

### Token savings example

| Session Length | Traditional | With Agentic Memory | Savings |
| -------------- | ----------- | ------------------- | ------- |
| 50 messages    | ~25K tokens | ~8K tokens          | **68%** |
| 200 messages   | ~100K tokens| ~28K tokens         | **72%** |
| 1000 messages  | ~500K tokens| ~120K tokens        | **76%** |

### How it works

Ogcode maintains a persistent **Topic → Concept → Fact** hierarchy with vector embeddings, plus a **function-level Call Graph** for codebase navigation. This knowledge graph survives across sessions, so your agent remembers your codebase structure, your conventions, and your past decisions.

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

Ogcode is a single Go binary that embeds a SolidJS web UI and runs its own HTTP server.

```
┌─────────────┐     REST + SSE      ┌──────────────┐
│  Web UI     │ ◄────────────────► │  Go Server   │
│  (SolidJS)  │                    │  (port 9595) │
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

| Component         | Responsibility                                                                 |
| ----------------- | ------------------------------------------------------------------------------ |
| **Agent Loop**    | Streaming LLM chat with tool execution (bash, read, write, edit, glob, grep, memory_recall, callgraph, deep_search) |
| **Session Store** | SQLite database for conversations, plans, tasks, and permissions               |
| **Git Worktrees** | An isolated branch per task, so multiple agents work in parallel               |
| **Knowledge Graph** | Persistent semantic memory with vector embeddings                            |
| **Call Graph**    | Function-level code relationship tracking                                       |
| **Search Bridge** | Playwright-based headless Chrome for web research                              |

---

## Roadmap

- [ ] **Advanced Task Planning & Parallel Execution** — Enhanced plan decomposition with manual/automatic agent assignment to tasks.
- [ ] **AI Daily Standups** — Voice-enabled meetings where agents report progress and discuss their work.
- [ ] **Ogland Integration** — Connect external services (Slack, Email, Jira) for planning-phase use.
- [ ] **Agentic Deployment** — End-to-end agentic deployment for major cloud providers, starting with AWS.

---

## Community

Join the Ogcode community on Discord to ask questions, share feedback and feature ideas, and stay up to date with releases.

[**Join us on Discord →**](https://discord.gg/JQP9t8y2Zv)

If Ogcode is useful to you, **starring the repo** helps more developers discover it.

---

## Contributing

Contributions are welcome — bug fixes, features, and documentation alike.

1. **Fork** the repository.
2. Create a feature branch: `git checkout -b feature/my-improvement`.
3. Make your changes and add tests where applicable.
4. Submit a pull request.

Please ensure your code follows the existing Go style and passes `go test ./...`.

---

## Security

- **Destructive operations require approval** — The agent cannot write files or run shell commands without your explicit permission. Approve once, or always per tool.
- **Git worktree isolation** — Each task runs in a separate git worktree, preventing accidental contamination of your working branch.
- **No data sent to third parties** — Your codebase is analyzed locally. Only conversation text is sent to your chosen LLM provider.
- **Local-first** — All data (sessions, plans, memory) is stored in local SQLite databases.

For security concerns, please open an issue or reach out on Discord.

---

## License

MIT License — see [LICENSE](LICENSE) for details.

<br/>

<div align="center">

**Made with care by the Ogcode team and contributors.**

[Star on GitHub](https://github.com/prasenjeet-symon/ogcode) · [Discord](https://discord.gg/JQP9t8y2Zv) · [Docs](docs/OUTLINE.md)

</div>
