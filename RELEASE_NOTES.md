# Release Notes — v0.13.1

## Per-task model selection

You can now choose the **model per task**, and the breakdown agent now bakes a
build/test verification step into every task it generates.

---

### ✨ What's new

- **Per-task model override.** Each task can run on its own model instead of always
  inheriting the plan's. Open a task in the board, and (while it's pending or
  failed) pick a model from the drawer — or reset it back to the plan default.
  Resolution at run time is: the task's model → the plan's model → the server
  default. Each task is independent; there's no global change.

- **Model shown on every Kanban card.** Cards now display the effective model
  (override or plan default), with a small **custom** tag when a task uses its own
  model, so you can see at a glance what each task will run on.

- **Breakdown tasks now include a verification step.** Every task the breakdown
  agent produces now ends with an explicit instruction to run the project's tests
  (or build/compile the project) so the build agent confirms there are no
  compile-time or syntax errors before the task is considered done.

### 🐛 Fixes

- **Model dropdown stays on screen.** The model picker is now a viewport-anchored
  popover that clamps to the screen and flips up/down based on available space, so
  it no longer gets clipped when opened inside the task drawer or near a screen edge.

### 📁 Files Changed

**Modified:** `internal/agent/agent.go`, `internal/server/task_routes.go`,
`internal/task/task.go`, `internal/task/store.go`, `web/src/pages/plan-tasks.tsx`,
`web/src/components/model-selector.tsx`, `web/src/context/plan.tsx`,
`web/src/api/client.ts`, `internal/cli/version.go`, `internal/version/version.go`,
`web/package.json`, `web/package-lock.json`
**Added:** `internal/db/030_task_model.sql`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.13.0...v0.13.1*
