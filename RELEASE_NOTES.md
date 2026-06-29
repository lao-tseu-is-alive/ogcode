# Release Notes — v0.12.0

## Plan mode: PRs target your branch, and task chains build correctly

This release sharpens Plan mode end to end. Pull requests opened from a plan's
tasks now target the branch you're actually working on, dependency-chained tasks
genuinely build on one another, and every chain PR is surfaced in the app. Plus a
batch of plan-mode reliability fixes and a UI parity fix in the plan sidebar.

---

### ✨ Highlights

- **Task PRs target your active branch, not just `main`.** When you lock a plan,
  ogcode now captures the repository's **active branch** and uses it as the base
  for every PR raised by that plan's tasks (and for the combined chain PR). The
  work merges back into wherever you're building, not always the default branch.
  If the active branch isn't on the remote yet, ogcode pushes it first so the PR
  can target it. (Adds a `base_branch` column to plans — migration runs
  automatically on first start.)

- **Chain PRs are now visible in the app.** When a dependency chain finishes, its
  single pull request is recorded on **every task in the chain** — the PR link and
  number on success, or a clear reason on failure (e.g. no remote, push failed).
  Previously the chain PR was opened on GitHub but never surfaced in the UI.

- **Plan sidebar parity.** The Plan-mode left nav now includes **Call Graph** and
  **Doc Index** alongside Notes, matching the Build-mode sidebar. Previously only
  Notes was shown in Plan mode.

### 🐛 Fixes

- **Chained tasks build on their predecessor's work.** Fixed an ordering bug where
  a dependent task's worktree was created from the shared chain branch *before* the
  just-completed task was merged into it — so each step was implemented missing the
  previous step's changes. The merge now happens before the next task starts.

- **A linear chain can no longer be split across two branches.** `assignChainBranches`
  is now order-independent: every task in a chain resolves to the same chain-root
  branch regardless of the order the breakdown produced them in.

- **Failed chains no longer strand completed work silently.** If a task in a chain
  fails, the chain's already-completed tasks now show why their PR is pending
  ("chain blocked — retry the failed task") instead of disappearing with no PR and
  no explanation.

- **Consistent plan completion state.** "All tasks completed" is now computed the
  same way for the plan list and the single-plan view, and a locked plan whose
  breakdown produced no tasks is no longer stuck "active" forever.

- **Plan finalization race.** The final-summary agent loop that runs while a plan is
  being locked is now registered and cancelable, so a message sent mid-lock can't
  start a second concurrent loop on the same session.

### 📁 Files Changed

**Modified:** `internal/server/plan_routes.go`, `internal/server/task_routes.go`,
`internal/plan/plan.go`, `internal/plan/store.go`, `internal/git/git.go`,
`web/src/components/plan-sidebar.tsx`, `internal/cli/version.go`,
`internal/version/version.go`, `web/package.json`, `web/package-lock.json`
**Added:** `internal/db/029_plan_base_branch.sql`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.11.2...v0.12.0*
