# 🚀 Ogcode v0.4.0 Release

## 📋 Executive Summary
This release introduces MEMORY.md — persistent project memory that survives across sessions — alongside parallel tool execution for faster agent runs, per-model pricing visibility in the UI, expanded model catalog support, and key reliability fixes.

---

## 📝 Changes Summary

### ✨ New Features
- **MEMORY.md — persistent project memory** — agents can now read/write a `MEMORY.md` file in your project to persist decisions, architecture notes, and hard-won knowledge across sessions. The system prompt always explains MEMORY.md's purpose and how to maintain it, even when the file doesn't yet exist.
- **Parallel tool execution** — multiple independent tool calls are now executed concurrently via goroutines instead of sequentially, significantly reducing round-trip latency on agent turns with multiple file reads, searches, or bash commands.
- **LLM instructed to prefer parallel tool calls** — each agent system prompt now includes an explicit section telling the LLM to batch independent tool calls in a single response block, taking full advantage of parallel execution.
- **Per-model pricing in catalog and UI** — InputPricePerM and OutputPricePerM are now stored in the model catalog and displayed as price badges in the model selector dropdown and Settings > Models page.
- **New models added to catalog** — Anthropic: claude-opus-4-6, claude-opus-4-5-20251101, claude-opus-4-1-20250805, claude-sonnet-4-5-20250929. OpenAI: gpt-5, gpt-5-mini, gpt-5-nano, gpt-4.1, gpt-4.1-mini, gpt-4.1-nano. GPT-4o/GPT-4o-mini retired from active-by-default.

### 🔧 Bug Fixes
- **Nil-safety checks for GetPart** — two places in RunLoop could dereference a nil part when GetPart returns (nil, nil), causing potential panics. Now explicitly checked.
- **MEMORY.md instructions always injected** — previously only injected when a file already existed, meaning agents in projects without MEMORY.md never learned about the feature. Now always present.

### 🛠 Internal
- **Version bump from 0.3.0 → 0.3.1 → 0.4.0** — version references updated across CLI and web package.

---

## 📥 Installation

**macOS/Linux:**
```bash
curl -fsSL http://ogcode.xyz/install.sh | sh
```

**Windows:**
```powershell
irm http://ogcode.xyz/install.ps1 | iex
```

**Homebrew:**
```bash
brew install prasenjeet-symon/tap/ogcode
```

**Winget:**
```powershell
winget install prasenjeet-symon.ogcode
```

**Go Install:**
```bash
go install github.com/prasenjeet-symon/ogcode@latest
```

---

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.3.1...v0.4.0*