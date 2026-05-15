# 🚀 Ogcode v0.2.9 Release

## 📋 Executive Summary
This patch release brings Mermaid diagram support throughout the chat interface, improves the WinGet distribution pipeline, and fixes a UI timeout issue that prematurely hid the cancel CTA.

---

## 📝 Changes Summary

### ✨ New Features
- **Mermaid diagram rendering** in chat markdown (sequence, flow, state, class, ER, Gantt diagrams)
- **BuildAgent auto-generates Mermaid diagrams** when describing plans or architectures
- **PlanAgent generates diagrams** when a visual plan is more helpful than text
- **Horizontally scrollable markdown tables** for better readability with wide content
- **Agentic Session Memory** — settings page label renamed from "Agentic Memory" for clarity

### 🔧 Bug Fixes
- **WinGet manifest path fixed** — manifests are now emitted to `manifests/p/prasenjeet-symon/ogcode/{{.Version}}` to satisfy Microsoft validation requirements
- **Removed 2-minute polling timeout** — prevents premature loss of the cancel CTA while agents are working

### 📦 Distribution
- WinGoReleaser WinGet block updated with proper manifest directory structure, `tags`, `publisher_support_url`, and `release_notes_url` metadata

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/releases/tag/v0.2.9*
