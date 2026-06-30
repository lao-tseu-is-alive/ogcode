# Release Notes — v0.13.0

## A Linear-style Plan mode, and a branch-sync safety check

This release gives Plan mode a focused redesign — a Linear-inspired Kanban board,
status iconography, a command palette, and full keyboard control — and adds a
launch-time check that warns when your branch is out of sync with its remote
before tasks branch from it.

---

### ✨ Plan mode, redesigned

- **Linear-style status icons.** The old colored dots are replaced by segmented
  status circles (todo ring → in-progress pie → done check → failed ✕), backed by
  a Linear-aligned status palette (gray / amber / indigo / red) exposed as design
  tokens.

- **A Linear-style Kanban board.** Columns are now flat and open (no boxed
  drop-zones), with minimal headers — status icon, name, and a quiet muted count.
  Cards are compact and flat: subtle neutral borders, a restrained hover (no lift
  or heavy shadow), and actions that reveal on hover.

- **Full-width, responsive board.** The board now fills the entire screen width and
  distributes columns evenly, while still scrolling horizontally on narrow windows
  instead of squishing.

- **Keyboard-first navigation.** Move focus across the board with the arrow keys or
  `h/j/k/l`, open a task with `Enter`, `S` to start, `R` to retry, `Esc` to close.

- **Command palette (⌘K).** A reusable, fuzzy-searchable palette surfaces the
  contextual actions for the focused task — Start, Open session, Mark complete /
  failed, Retry, Open PR — plus "Start all eligible".

- **Wider detail drawer.** The task detail side panel now opens at ~40% of the
  screen width (with a sensible floor) so descriptions are comfortable to read.

### ✅ Branch-sync check at launch

- When you start Plan mode, ogcode checks whether the **current active branch** is
  in sync with its upstream. It does a best-effort, time-bounded `git fetch` first
  so the result reflects the real remote, then shows a dismissible banner only when
  the branch is **behind**, **diverged**, or has **no upstream** — so your tasks
  don't branch from a stale base. New `GET /api/git/sync` endpoint backs it.

### 📁 Files Changed

**Modified:** `web/src/pages/plan-tasks.tsx`, `web/src/styles/index.css`,
`web/src/App.tsx`, `web/src/api/client.ts`, `internal/git/git.go`,
`internal/server/config_routes.go`, `internal/server/routes.go`,
`internal/cli/version.go`, `internal/version/version.go`, `web/package.json`,
`web/package-lock.json`
**Added:** `web/src/components/status-icon.tsx`, `web/src/components/command-menu.tsx`,
`web/src/components/git-sync-banner.tsx`, `internal/git/sync_test.go`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.12.0...v0.13.0*
