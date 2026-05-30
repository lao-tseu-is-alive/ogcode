# Release Notes — v0.8.0

## Web Search Agent

This release adds a **deep research agent** that can search the web, fetch pages, and synthesise findings — giving every agent in ogcode the ability to look up current information instead of guessing.

### New: Search Agent (deep_search)

- **SearchAgent** — A new built-in agent that decomposes a query into sub-queries, runs parallel Google searches via a headless Chrome bridge, fetches the most relevant pages, and produces a single synthesised answer with cited sources.
- **`deep_search` tool** — Available to Build, Plan, Note, and Breakdown agents. Pass a question and optionally context, and receive a complete research report as the tool result.
- **Ephemeral sessions** — Search runs in a temporary session (capped at 20 steps) that is automatically deleted when complete. Search sessions are hidden from the session list.
- **Reasoning fallback** — `extractLastAssistantText` now falls back to reasoning/thinking content when the text part is empty, ensuring thinking-model output is captured correctly.

### New: web_search & fetch_page Tools

- **`web_search`** — Searches Google and returns titles, URLs, and snippets. Supports `limit` (max 15) and parallel calls.
- **`fetch_page`** — Retrieves the readable text content of a URL (handles JavaScript-rendered pages). Returns title, URL, and extracted text (truncated at 14,000 characters).

### New: Search Bridge (Node.js subprocess)

- **Playwright-based bridge** — A Node.js/Express server (`tools/search-bridge/server.js`) manages a headless Chromium instance for search and fetch operations. Capable of handling JavaScript-heavy pages.
- **Automatic startup** — When search is enabled, the Go server starts the bridge as a subprocess and waits up to 30 seconds for it to become healthy.
- **Profile modes** — Isolated profile (safe default, no shared cookies) or real Chrome profile (uses your cookies/logins for authenticated sites; Chrome must be fully closed).
- **Concurrency control** — Maximum 4 concurrent tabs by default (`OGCODE_SEARCH_MAX_CONCURRENCY`).

### New: Search Configuration UI

- **Settings → General → Web Search Agent** — Toggle search on/off and enable "Use real Chrome profile" from the web UI.
- **Live bridge status** — Green indicator when the bridge is running, red with recovery instructions when it's not.
- **Restart reminder** — Amber banner reminds users to restart the server after enabling/disabling search.
- **Config API** — `GET /search/config` and `POST /search/config` endpoints for programmatic access. Config persisted in `search_config` database table.

### Agent Prompt Updates

- **Build, Plan, Breakdown, Note agents** — All now include `deep_search` in their tool suite and are instructed to use it for external knowledge (library docs, API references, version compatibility, security advisories, best practices).
- **Build agent rule** — "After calling deep_search, always write the research findings as your own text response — do not just return the tool result silently."

### Bug Fixes

- **Data race on MaxSteps** — The `LoopRunner.MaxSteps` field was being mutated during execution, causing a data race when `deep_search` runs a nested loop concurrently. Now read into a local variable before the loop.
- **Default embedding model** — `GetMemoryConfig` now defaults `EmbedModel` to `text-embedding-3-small` when empty, preventing nil-value panics.

### Installation & Packaging

- **Homebrew** — `brew install ogcode` now automatically runs `npm install` and `npx playwright install chromium` in the bridge directory when Node.js is available.
- **curl install script** — `install.sh` extracts search-bridge files from the release archive and installs npm dependencies when Node.js is present.
- **Makefile** — `make install` copies bridge files and runs `npm install` + `playwright install` into `~/.local/share/ogcode/search-bridge/`.
- **GoReleaser** — Release archives now bundle `search-bridge/server.js` and `search-bridge/package.json`.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `OGCODE_SEARCH_ENABLED` | `false` | Enable search bridge on server start |
| `OGCODE_SEARCH_BRIDGE_PORT` | `7331` | Port for the Node.js bridge |
| `OGCODE_SEARCH_BRIDGE_DIR` | auto-detect | Override bridge directory location |
| `OGCODE_SEARCH_USE_REAL_PROFILE` | `false` | Use real Chrome profile instead of isolated |
| `OGCODE_SEARCH_MAX_CONCURRENCY` | `4` | Max concurrent browser tabs |

### Database Migrations

- **`025_search_config.sql`** — Creates the `search_config` table (singleton row for enabled/disabled toggle).
- **`026_search_config_profile.sql`** — Adds `use_real_profile` column to `search_config`.

### Full Changelog

**Modified (14 files):**
`.goreleaser.yaml`, `Makefile`, `install.sh`, `internal/agent/agent.go`, `internal/agent/loop.go`, `internal/server/config_routes.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/session/memory_store.go`, `internal/session/store.go`, `internal/tool/tool.go`, `web/src/api/client.ts`, `web/src/context/server.tsx`, `web/src/pages/settings/general.tsx`

**New (9 files):**
`internal/db/025_search_config.sql`, `internal/db/026_search_config_profile.sql`, `internal/search/bridge.go`, `internal/search/process.go`, `internal/session/search_store.go`, `internal/tool/deep_search.go`, `internal/tool/fetch_page.go`, `internal/tool/web_search.go`, `tools/search-bridge/server.js`, `tools/search-bridge/package.json`

---

*See also: [v0.7.0 Release Notes](#) for the Codebase Indexing, Exclude Patterns, and codebase_map features.*