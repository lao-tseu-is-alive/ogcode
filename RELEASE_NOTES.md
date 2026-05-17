# 🚀 Ogcode v0.3.0 Release

## 📋 Executive Summary
This minor release introduces rich markdown rendering with LaTeX math and Plotly chart support, a project notes feature with export, improved session compaction and agentic memory architecture, and several provider reliability fixes including retry logic for transient errors.

---

## 📝 Changes Summary

### ✨ New Features
- **LaTeX math rendering** in chat markdown — inline ($...$) and display ($$...$$) equations rendered via KaTeX
- **Plotly interactive charts** in chat markdown — bar, line, scatter, pie, heatmap and more via triple-backtick `plotly` code blocks
- **Project notes** — persistent markdown notes per project, saved in `.ogcode/notes/`, with a dedicated NoteAgent
- **Note export** — download notes as markdown files via a new endpoint and UI button
- **Agent markdown capabilities** expanded — all agents (Build, Plan, Note) now know about Mermaid, LaTeX, and Plotly rendering and will use them when appropriate
- **Improved session compaction** — operates on user-turn boundaries instead of raw loop steps for more natural context trimming

### 🔧 Bug Fixes
- **Provider retry for transient body errors** — `400 Bad Request` responses containing "failed to read request body" are now automatically retried with exponential backoff instead of failing immediately
- **429 rate-limit retry logging** improved — retry messages now include status code and clearer context
- **Agentic memory and compaction decoupled** — these previously conflicting features now run on mutually exclusive paths, preventing token-savings miscalculation
- **Guard against empty slices** — `trimToRecent` protected from panicking on empty slices; token-savings math corrected
- **Title generation timeout** increased from 15s to 60s to prevent premature timeouts on slower providers
- **Dead `trimToRecent` function** removed from agent code

### 🛠 Reliability
- **Explicit `Content-Length` header** set on streaming chat requests to help cloud providers that require it
- **Request body byte count** logged at debug level for easier troubleshooting of truncation issues

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.2.9...v0.3.0*