# Release Notes — v0.6.0

## Document Indexing & PDF Intelligence

This release adds **first-class PDF support**: semantic page indexing, vision-aware page reading, and a full document-index management UI.

### New: Document Index System

- **`ogcode index` CLI command** — Scans the workspace for PDF files, extracts keyword corpora per page, and runs a lightweight **IndexAgent** to produce 2-5 semantic labels per page. Labels are persisted in the database and used by agents to navigate documents intelligently.
- **IndexAgent** — A new dedicated agent (`index`) that receives keyword corpora and outputs concise, descriptive topic labels for each page. Designed for speed and low token cost.
- **`pdf_index` tool** — Returns the stored semantic index (page labels + keyword corpus) for a PDF so agents know which pages to read before calling `read_pdf_page`.
- **`read_pdf_page` tool** — Extracts plain text from a single PDF page. When the page yields little text (<20 words) and the active model supports vision, the page is rendered to a JPEG image and attached instead — ideal for graphs, diagrams, and scanned documents.
- **`submit_doc_index` tool** — Used by the IndexAgent to persist semantic labels back to the database.

### New: Model Image-Support Detection

- **Automatic capability probing** — On first use of a model, ogcode sends a tiny test image to determine whether it accepts vision input. The result is cached permanently in `model_capability` until manually cleared.
- **Static catalog annotations** — The Anthropic and OpenAI model catalogs now include a `SupportsImages` field. All listed Claude and GPT-4o/4.1/5 models are marked as multimodal; reasoning-only models (o3-mini, o1-mini) are marked text-only.
- **Vision model heuristic** — For dynamically-fetched models (OpenRouter/Ollama), a name-based heuristic (`visionModelHints`) provides a conservative fallback when the live probe is inconclusive.
- **API endpoint: `POST /models/capability/clear`** — Clears cached image-support results so they are re-probed on next use. Accepts an empty `modelId` to clear all entries.
- **`ModelInfo.SupportsImages`** propagated through the models API and displayed in the web UI.

### New: Doc Index Web UI

- **Doc Index page** (`/docindex`) — Lists all indexed documents with page counts, file types, and timestamps. Includes "Index Docs" and "Rebuild" actions with a model-picker modal.
- **Sidebar navigation** — A new book icon in the sidebar provides quick access to the Doc Index page.
- **Realtime build status** — The UI tracks the indexing process via SSE events and shows a live "Indexing…" indicator.
- **Context provider** — `DocIndexProvider` manages doc list state, model selection, build/refresh actions, and SSE reconnect logic.

### Image Delivery in Tool Results

- **Tool Image support** — Tools can now attach images to their `Result`. The image is persisted in the session's `ToolState`, replayed on history loads, and delivered to the model in a provider-appropriate way:
  - **Anthropic** — Images embedded directly inside `tool_result` content blocks.
  - **OpenAI-family** — Images buffered from consecutive tool results and emitted as a follow-up user message (since OpenAI rejects images inside tool results).
- **`ModelSupportsImages` flag** — Passed to every tool execution via `tool.Context`, so tools can conditionally return images (e.g. `read_pdf_page` renders PDF pages as JPEG for vision models only).

### Database Migrations

- **`022_doc_index.sql`** — Creates the `doc_page_index` table for per-page keyword corpora and semantic labels.
- **`023_model_capability.sql`** — Creates the `model_capability` table for cached image-support probe results.

### New Dependencies

- `github.com/gen2brain/go-fitz` — MuPDF bindings for PDF rendering (page → JPEG image).
- `github.com/ledongthuc/pdf` — Pure-Go PDF text extraction (page → plain text).
- `github.com/joho/godotenv` — Promoted from indirect to direct dependency.
- `golang.org/x/sync` — Promoted from indirect to direct dependency.

### Build Change

- **`CGO_ENABLED=1`** — The Makefile now builds with CGO enabled (required by the `go-fitz` MuPDF bindings).

### Full Changelog

**Modified (20 files):**
`Makefile`, `go.mod`, `go.sum`, `internal/agent/agent.go`, `internal/agent/loop.go`, `internal/cli/root.go`, `internal/provider/anthropic.go`, `internal/provider/models_catalog.go`, `internal/provider/openai.go`, `internal/provider/provider.go`, `internal/server/config_routes.go`, `internal/server/model_routes.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/session/message.go`, `internal/session/model_store.go`, `internal/session/schema.go`, `internal/session/store.go`, `internal/tool/tool.go`, `web/src/api/client.ts`, `web/src/app.tsx`, `web/src/components/session-sidebar.tsx`

**New (13 files):**
`internal/db/022_doc_index.sql`, `internal/db/023_model_capability.sql`, `internal/docindex/model.go`, `internal/docindex/store.go`, `internal/indexer/indexer.go`, `internal/indexer/pdf.go`, `internal/indexer/stopwords.go`, `internal/provider/probe.go`, `internal/server/docindex_routes.go`, `internal/tool/pdf_index.go`, `internal/tool/pdf_render.go`, `internal/tool/read_pdf_page.go`, `internal/tool/submit_doc_index.go`, `web/src/context/docindex.tsx`, `web/src/pages/docindex.tsx`