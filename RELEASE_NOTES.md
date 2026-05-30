# Release Notes — v0.7.0

## Codebase Indexing & Source Navigation

This release adds **full-text codebase indexing** alongside the existing PDF support, plus an **exclude patterns** system to control what gets indexed, and a new **`codebase_map` tool** that lets agents discover relevant files by topic before reading them.

### New: Text & Code File Indexing

- **30+ source file extensions** — The indexer now walks PDFs **and** text/code files (`.go`, `.ts`, `.py`, `.rs`, `.java`, `.cpp`, `.html`, `.css`, `.yaml`, `.sql`, and more). Each file is treated as a single "page" and processed through the same keyword extraction + IndexAgent labelling pipeline.
- **Improved keyword extraction** — `splitCamelCase` now decomposes identifiers like `getUserName` into `get`, `user`, `name` for richer indexing of source code symbols.
- **Skip already-indexed docs** — The indexer checks `IsDocIndexed` before processing, so re-runs only pick up new or changed files.

### New: Exclude Patterns

- **`index_excludes` database table** — Stores per-directory glob patterns that the indexer should skip (e.g. `node_modules`, `vendor`, `*.min.js`).
- **Default excludes seeded automatically** — On first index, sensible defaults are populated: `node_modules`, `vendor`, `dist`, `build`, `.git`, `__pycache__`, `.ogcode`, lock files, minified assets, and more.
- **Full CRUD API** — `GET /docindex/excludes`, `POST /docindex/excludes`, and `DELETE /docindex/excludes/{id}` let you add, list, and remove patterns.
- **Excludes modal in the UI** — A new dialog on the Doc Index page for managing skip patterns without touching the database directly.

### New: `codebase_map` Tool

- **`codebase_map`** — A new agent tool that returns a labeled JSON tree of all indexed text/code files. Agents call it to discover which files are relevant to a topic before reading them, dramatically improving navigation of large codebases.
- **`subdir` parameter** — Scope the map to a specific directory (e.g. `"internal/auth"`) instead of loading the entire project tree.
- **Integrated into Build & Plan agents** — Both BuildAgent and PlanAgent now have `codebase_map` in their tool suite and are instructed to prefer it as the first exploration step.

### Doc Index UI Redesign

- **Collapsible folder tree** — The Doc Index page now shows indexed files in an expandable folder tree instead of a flat list.
- **Search & filter** — Quickly find files by name or label.
- **File type badges** — Visual badges distinguish PDFs from source files.

### Other Improvements

- **Index sessions hidden** — Sessions created by the IndexAgent (`session_type = 'index'`) are now excluded from the session list, keeping the UI clean.
- **Web version bumped** — Frontend package version updated to `0.7.0`.

### Database Migrations

- **`024_index_excludes.sql`** — Creates the `index_excludes` table for storing skip patterns per directory.

### Full Changelog

**Modified (11 files):**
`internal/agent/agent.go`, `internal/docindex/store.go`, `internal/indexer/indexer.go`, `internal/indexer/pdf.go`, `internal/server/docindex_routes.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/session/store.go`, `internal/cli/version.go`, `internal/version/version.go`, `web/package.json`

**New (4 files):**
`internal/db/024_index_excludes.sql`, `internal/docindex/excludes.go`, `internal/indexer/text.go`, `internal/tool/project_index.go`

---

*See also: [v0.6.0 Release Notes](#) for the Document Indexing & PDF Intelligence features that this release builds upon.*