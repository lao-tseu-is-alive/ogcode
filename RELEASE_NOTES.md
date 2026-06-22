# Release Notes — v0.9.2

## Manual Notes & AI-Powered Block Editor

This release turns Notes from an AI-only, read-only artifact into a fully editable workspace. You can now **create blank notes manually**, **edit any note inline** with a Notion-style block editor, and **transform selected text with AI** — all without leaving the notes page.

---

### ✨ New Features

- **Manual note creation** — A new **New Note** button on the notes page creates a blank note instantly (no AI loop required). Manual notes are tagged with a `source: 'manual'` badge so they're visually distinct from AI-generated notes. Empty manual notes drop you straight into the editor.
- **Inline note editing** — Any note (AI-generated or manual) can now be edited in place. Click the **Edit** button on a note's detail page to switch into edit mode with an editable title and content. Saving increments the version and records a version snapshot, so the full history is preserved alongside AI-generated versions.
- **Notion-style block editor** (`note-editor.tsx`) — A new rich block editor component powers editing:
  - **Slash commands** — Type `/` to insert headings, bullet/numbered lists, to-dos, code blocks, blockquotes, dividers, bold, italic, links, and images.
  - **Block reordering** — Drag blocks by their grip handle to reorder. Top/bottom drop indicators show the target position.
  - **Image support** — Paste images, drag-and-drop from the OS, or use the slash `image` command. Images are embedded as base64 (max 5 MB) with editable alt text, replace, and remove actions.
  - **Keyboard navigation** — Enter to split blocks (with list continuation), Backspace at the start of a line to merge, ArrowUp/ArrowDown to move between blocks.
- **AI text transformation** — Select any text in the editor to reveal a floating toolbar with **Improve**, **Shorter**, **Longer**, and **Fix grammar** actions. The selected text is sent to your chosen LLM model and the result can be previewed and applied back into the note (single-block or cross-block selection supported).
- **Model picker in edit mode** — A model selector (with `bottom` placement so it doesn't overflow the toolbar) lets you choose which LLM powers AI transformations for that note.

### 🔧 Backend

- **`source` column on notes** — New `028_notes_source.sql` migration adds a `source TEXT NOT NULL DEFAULT 'ai'` column to the `note` table. The `Note` model, store, and all scan paths now read/write the source field. Existing notes default to `'ai'`.
- **`PATCH /api/notes/{noteID}`** — New endpoint to update a note's title and content. `Store.SaveContent` increments the version, updates the row, and records a `NoteVersion` snapshot.
- **`POST /api/notes/transform`** — New endpoint that runs an AI text transformation (improve / shorter / longer / grammar) using the configured provider, streaming the result and returning the trimmed text.
- **Manual note creation** — `POST /api/notes` now accepts an optional `source` field. When `source === 'manual'`, a blank note is created immediately with status `done` and no agent loop is started.
- **`note.created` / `note.manual_updated` events** — Manual note creation and manual edits publish events through the existing event bus.

### 🎨 Web UI

- **Notes page** — New Note button added next to the search box; empty-state copy updated to mention both manual writing and "Save to Notes". Note cards show a "Manual note" badge with a pencil icon instead of the query text for manual notes.
- **Note detail page** — Edit mode toggles the title input, swaps the markdown preview for the block editor, and hides the status badge / history / export / delete controls while editing. Cancel returns to view mode without saving. Manual notes with empty content show an "empty note" prompt with a hint to click Edit.
- **Model selector placement** — `ModelSelector` now accepts a `placement` prop (`'top'` default, `'bottom'` for use in toolbars near the top of the viewport).

### 📁 Files Changed (13 files)

**New:** `internal/db/028_notes_source.sql`, `web/src/components/note-editor.tsx`

**Modified (backend):** `internal/note/note.go`, `internal/note/store.go`, `internal/server/note_routes.go`, `internal/server/routes.go`, `internal/cli/version.go`, `internal/version/version.go`

**Modified (web):** `web/src/api/client.ts`, `web/src/components/model-selector.tsx`, `web/src/context/note.tsx`, `web/src/pages/note-detail.tsx`, `web/src/pages/notes.tsx`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.9.1...v0.9.2*