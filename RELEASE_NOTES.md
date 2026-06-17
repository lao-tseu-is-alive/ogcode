# Release Notes — v0.9.0

## Smarter Search, Faster Indexing, Context Safety

This release significantly improves the **deep search agent** with Bing fallback, stealth anti-detection, and guaranteed source citations. The **document indexer** is now much faster with multi-file batching and parallel workers. A new **proactive context compaction** system prevents context-length errors before they happen, and **note queries** are now rewritten using conversation context for better retrieval.

---

### ✨ New Features

- **Bing fallback & stealth anti-detection** — The search bridge now falls back to Bing when Google is blocked, uses `headless=new` mode, and applies stealth anti-detection scripts for more reliable scraping.
- **Guaranteed Sources section** — Deep search results always append a deduplicated Sources section with URLs collected from `web_search` and `fetch_page` calls, so agents never lose citation links.
- **Markdown rendering for search results** — Deep search output is now rendered as proper markdown with an expanded Sources section, instead of raw text.
- **codebase_map for Note & Build agents** — Both the Note Agent and Build/Plan agents now receive the `codebase_map` tool and are instructed to use it as their first exploration step, ensuring smarter project navigation before reading files.
- **Note query rewriting** — Note queries are now rewritten using conversation context for more relevant retrieval, instead of sending raw user input.
- **Viewport-aware note generation** — The note agent RunLoop now receives viewport dimensions for responsive content rendering.

### ⚡ Performance Improvements

- **Multi-file batching & parallel workers** — The document indexer now batches multiple files together and processes them in parallel with configurable workers, dramatically speeding up large project indexing.
- **Real-time indexing progress bar** — The doc index UI now shows a live progress bar during indexing, so you know exactly how far along the build is.
- **Deep search speed optimizations** — Tighter timeouts, prompt optimization, and improved concurrency make deep search noticeably faster.
- **Search concurrency increase** — Default `MAX_CONCURRENCY` raised from 4 to 15, and bridge HTTP timeout bumped to 150s for reliability on slow networks.

### 🔧 Bug Fixes

- **Proactive context compaction** — The agent loop now estimates request body size before sending to the LLM and proactively compacts messages that exceed a 500KB threshold. This prevents context-length errors (especially with smaller local models like Ollama).
- **Ollama 400 error detection** — Context overflow errors from Ollama (HTTP 400) are now detected and trigger automatic message compaction and retry, instead of failing the session.
- **Deep search timeout** — Increased from 90s to 180s to prevent "context deadline exceeded" errors on complex research queries.
- **Note query model consistency** — Note query rewriting now uses the same model configured for the Note Agent, instead of overriding to haiku/mini/flash.
- **Search bridge HTTP timeout** — Bumped to 150s to handle slower search results without premature timeouts.

### 📁 Files Changed (23 files)

**Modified:** `AGENT.md`, `internal/agent/agent.go`, `internal/agent/loop.go`, `internal/agent/loop_test.go`, `internal/agent/prompt_builder.go`, `internal/agent/prompt_builder_test.go`, `internal/indexer/indexer.go`, `internal/indexer/indexer_test.go`, `internal/search/bridge.go`, `internal/server/docindex_routes.go`, `internal/server/note_routes.go`, `internal/server/server.go`, `internal/tool/deep_search.go`, `tools/search-bridge/server.js`, `web/src/api/client.ts`, `web/src/components/message-item.tsx`, `web/src/context/docindex.tsx`, `web/src/context/note.tsx`, `web/src/pages/docindex.tsx`

**New:** `internal/db/027_memory_config_base_url.sql` (from v0.8.x branch)

---

### 📥 Installation

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.8.2...v0.9.0*