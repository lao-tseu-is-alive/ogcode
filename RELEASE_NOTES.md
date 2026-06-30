# Release Notes — v0.13.2

## Leaner agents, a more Linear-feeling UI

This release removes the Call Graph feature, polishes Plan mode and Settings to a
consistent Linear-inspired look and feel, and sharpens the task-breakdown agent.

---

### 🗑️ Call Graph removed

The Call Graph feature has been fully removed — from the agents (no agent is aware
of it anymore), the tools, the API, and the UI. This trims agent context and
simplifies the surface area. Memory (the knowledge graph + `memory_recall`) is
unaffected and continues to work as before.

### ✨ Plan mode & Settings — Linear polish

- **Motion.** Messages now animate in subtly as they arrive, and Settings pages
  fade in on navigation — all gated behind `prefers-reduced-motion` for anyone who
  opts out.
- **Consistent status icons.** The conversation-view task panel now uses the same
  Linear-style status circles as the board, so status reads identically everywhere.
- **Refined surfaces.** Tighter, more consistent corner radii and lighter,
  border-driven shadows across inputs, cards, and the Settings module — plus
  design-token cleanup so theming stays coherent.
- **Sidebar fix.** The minimized Plan sidebar now shows the **Doc Index** icon
  alongside Notes (previously it was missing when collapsed).

### 🔧 Task-breakdown agent improvements

- The breakdown agent now has **`codebase_map`** and **`deep_search`** — it leads
  exploration with a labeled codebase overview and can verify library/API details,
  producing more accurate, implementation-ready task descriptions. (Previously the
  prompt referenced `deep_search` without the tool actually being available.)
- Each generated task now reliably ends with a **verification step** (run tests, or
  build the project) in both the system prompt and the breakdown instructions.
- More stable task ordering and a language-neutral example so descriptions aren't
  biased toward any one stack.

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

**Docker:**
```bash
docker run -p 9595:9595 -v $(pwd):/workspace -w /workspace ghcr.io/prasenjeet-symon/ogcode:latest
```

---

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.13.1...v0.13.2*
