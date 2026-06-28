# Release Notes — v0.11.0

## Zero-Config Local Memory Embedder

Agentic session memory now works **out of the box with no setup** — no embedding API key, no external embedding service, no separate model server. ogcode ships with a built-in, in-process sentence-embedding model (`all-MiniLM-L6-v2`, 384-dim) that runs entirely on your machine. The one-time ~86 MB model weights download automatically at first startup; everything after that is fully local and offline.

This removes the previous hard dependency on a third-party embedding provider (OpenAI/OpenRouter/Ollama) for memory, and lets memory synthesis follow the provider + model you've already selected for the session.

---

### ✨ New Features

- **Built-in local embedder** (`internal/provider/embedmodel`, `internal/provider/local.go`) — A pure-Go (`hugot` / GoMLX) sentence-embedding provider runs in-process with no CGO and no external service. The small tokenizer/config assets (~700 KB) are embedded in the binary; the ~86 MB ONNX weights are downloaded on first use from Hugging Face (SHA-256 verified, atomic rename) into `~/.ogcode/embed-model`. This keeps the distributable binary small (~55 MB instead of ~141 MB) while preserving the single-command, no-API-key experience. Inference is mutex-serialized because the Go backend is not goroutine-safe, and a sidecar marker skips re-download/re-hash on later runs.
- **Eager model prefetch at startup** (`LocalEmbedder.Prefetch`, `Memory.PrefetchEmbedder`) — The one-time weight download kicks off during boot in a background goroutine (guarded by `sync.Once`) instead of blocking the first memory-related agent turn. No-op for non-local embed providers.
- **Blocking preflight download** (`EnsureLocalEmbedderModel`) — The embedder weights are downloaded before the server accepts requests (`ogcode` and `ogcode plan`), regardless of whether memory is enabled at boot, so a runtime enable via the settings UI needs no restart. Uses a 5-minute timeout and is non-fatal on error (logs and continues; the next `Embed` call retries). Model preparation (download + verify) is split from full init so the preflight and later inference instances share one cache directory and never re-download.
- **Session-driven memory synthesis** — Topic/concept inference, enrichment, and recall refinement now use the **same provider + model selected for the current session** rather than a server-wide default baked in at boot. The synthesis `ChatClient` is injected per call into `WriteMemory` / `RecallMemory`, and the `memory_recall` tool resolves the provider from the session model via the registry.

### 🔧 Backend

- **Embedding is always local** — `ResolveEmbedProvider` and the external chat-provider embedding constructors are removed; third-party embedders are no longer used for agentic memory. `GraphOpts` and `MemoryConfig` collapse to their essential fields (`Enabled` / `EmbedProvider`). Legacy DB columns are retained for migration safety but are no longer read or written.
- **Removed model-selection surface** — The `/memory/models` route and its handler (`internal/server/memory_models.go`) are gone, since memory no longer needs embed/chat model selection.

### 🐛 Fixes

- **Role-aware project-index prompt** (`internal/agent/prompt_builder.go`) — `projectIndexPrompt(role)` is now tailored per agent role instead of always ending in "Then make changes": `build → make changes`, `plan → produce your plan`, `note → produce your note`. This fixes the incorrect write-oriented instruction leaking into read-only agents (PlanAgent, NoteAgent), mirroring the existing role-aware `callGraphPrompt`. PlanAgent's start-of-session step 2 now explicitly begins with `codebase_map`.

### 🎨 Web UI

- **Simplified memory settings** — Now that the local embedder works with zero setup, the "Agentic Session Memory" card, its `MemoryConfigForm`, and the embed/chat provider, model, key, and base-URL fields are removed from General settings, along with the now-unused `getMemoryConfig`/`setMemoryConfig` API client code.

### 🧪 Tests

- **Local embedder unit tests** (no network) — cover static identifiers, cache-dir env override and explicit-precedence, the `prepareModel` cached short-circuit (marker + model present ⇒ no download), and the `EnsureLocalEmbedderModel` preflight with a warm cache.
- **Gated integration test** (`OGCODE_EMBED_INTEGRATION=1`) — downloads the real ~86 MB model, runs inference, and asserts semantic sanity (related sentences score higher cosine similarity than unrelated: ~0.77 vs ~0.005), plus cache persistence and no re-download on a second instance. Gated so CI stays fast and offline.
- **Prompt-builder tests** — cover the role-specific workflow tails and the plan-agent step-2 `codebase_map` reinforcement.

### 📁 Files Changed

**New:** `internal/provider/embedmodel/` (embedder package + bundled tokenizer/config assets), `internal/provider/local.go`, `internal/provider/local_test.go`

**Removed:** `internal/server/memory_models.go`

**Modified (backend):** `internal/agent/agent.go`, `internal/agent/loop.go`, `internal/agent/prompt_builder.go`, `internal/memory/graph.go`, `internal/memory/memory.go`, `internal/provider/provider.go`, `internal/server/config_routes.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/session/memory_store.go`, `internal/tool/memory_recall.go`, `internal/cli/version.go`, `internal/version/version.go`

**Modified (web):** `web/src/api/client.ts`, `web/src/lib/providers.ts`, `web/src/pages/settings/general.tsx`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.10.0...v0.11.0*
