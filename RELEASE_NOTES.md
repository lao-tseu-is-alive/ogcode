# Release Notes — v0.9.1

## LaTeX Document Support & Intelligent Environment Detection

This release adds **full LaTeX document support** with inline PDF compilation, viewport rendering, and intelligent environment detection — so the agent always writes LaTeX compatible with your local system.

---

### ✨ New Features

- **LaTeX-to-PDF agent tool** — Agents can now compile LaTeX source code to PDF using the new `latex_to_pdf` tool. The agent receives the compiled PDF path and, for vision-capable models, the first page rendered as a JPEG image for verification.
- **Inline LaTeX document rendering** — `language-latex` code blocks in chat are now automatically detected and rendered inline. The web UI extracts LaTeX blocks, compiles them server-side via pdflatex, and displays styled page previews with document class/title extraction, source code toggle, and a "Download PDF" button.
- **LaTeX environment detection** — On startup, the server detects your local pdflatex version, TeX distribution (TeX Live, MiKTeX, etc.), available document classes, and key packages via `kpsewhich`. This information is injected into the agent system prompt so agents write compatible LaTeX without guessing what's available.
- **LaTeX API routes** — Three new endpoints power the rendering pipeline:
  - `POST /api/latex` — Compile LaTeX source to PDF
  - `POST /api/latex/pages` — Compile + render pages as base64-encoded JPEG images
  - `GET /api/latex/status` — Check pdflatex availability and version info
- **go-fitz PDF rendering** — The `latex_fitz.go` module uses the go-fitz library (MuPDF bindings) to render PDF pages as high-quality JPEG images for inline display and vision model consumption.

### 🎨 Web UI

- **LaTeX document preview cards** — Markdown content renderer now detects `language-latex` code blocks and renders them as styled preview cards with:
  - Document class and title extraction
  - Inline page image rendering
  - Source code preview toggle
  - Download PDF button
- **LaTeX-specific CSS** — New styles for LaTeX preview cards, page images, error states, and compilation status indicators.
- **Sandboxed rendering pipeline** — LaTeX source is encoded as base64 data attributes, extracted before DOMPurify sanitization, then sent to the server for compilation. Results are rendered inline without page reloads.

### 🔧 Agent System Prompt

- **LaTeX environment section** — When pdflatex is available, the agent system prompt now includes a "LaTeX environment" section listing the detected version, distribution, available document classes (article, report, book, etc.), and installed packages. This ensures agents write compilable LaTeX the first time.

### 📁 Files Changed (13 files)

**New:** `internal/server/latex_fitz.go`, `internal/server/latex_routes.go`, `internal/tool/latex_pdf.go`

**Modified:** `internal/agent/agent.go`, `internal/agent/loop.go`, `internal/agent/prompt_builder.go`, `internal/agent/prompt_builder_test.go`, `internal/server/routes.go`, `internal/server/server.go`, `web/src/components/markdown-content.tsx`, `web/src/styles/index.css`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.9.0...v0.9.1*