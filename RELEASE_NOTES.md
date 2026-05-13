# 🚀 Ogcode v0.2.5 Release

## 📋 Executive Summary
This patch release brings significant improvements to the agent system with smarter plan breakdown extraction, enhanced security through tool allowlists, and a more robust plan locking experience with better error feedback.

---

## 🔧 Backend Improvements

### Agent System Enhancements
- **Refined System Prompts**: Improved BuildAgent and PlanAgent prompts with clearer process steps and hard rules for scope, file ownership, and dependency linearity
- **Better Summary Extraction**: Added `extractFinalSummary()` to pass only the locked plan to the breakdown agent instead of the full conversation — reducing noise and improving accuracy
- **Tool Allowlist Security**: Added explicit tool allowlist checks in `executeTool` — rejects any tool not in the agent's explicit list, guarding against prompt injection or model errors
- **Session Mismatch Detection**: Added session/agent type mismatch warnings in RunLoop to catch call-site bugs early
- **Plan Locking Reliability**: Made `handleLockPlan` fail fatally with HTTP 500 when final plan summary generation fails, enabling UI retry prompts

### Plan Archiving Fix
- Fixed race condition in `tryArchivePlan` by using DB gate (`ArchivedAt > 0`) instead of `os.Stat`
- Properly call `planStore.Archive()` after successful file write to persist archived state

---

## 🎨 Frontend Improvements

### Token & Model Display
- **TokenPill Component**: Enhanced to accept a `messages` prop for displaying plan session tokens independently from build session
- **Model Selector Fallback**: Fixed fallback chain to include `allModels()` when selected model is disabled — prevents empty labels

### Error Handling
- **Lock Error Feedback**: Added `lockError` signal to PlanContext with dismissible error banner in `PlanPromptInput` when locking fails

### Build Output
- Renamed build output directory from `'build'` to `'dist'` in web embed configuration

---

## 📝 Changes Summary

```
18 files changed, 191 insertions(+), 109 deletions(-)

internal/agent/agent.go         | 119 ++++++++++++++++++++-----------
internal/agent/breakdown.go    |  85 +++++++++++-----------
internal/agent/loop.go         |  17 +++++
internal/server/plan_routes.go |  16 ++++++-
web/embed.go                   |   2 +-
web/src/components/*.tsx       |  21 +++++--
web/src/context/*.tsx          |   9 ++-
web/src/pages/*.tsx            |   3 +
```

### Key Commits
- `614481f` - feat: improve agent instructions, breakdown extraction, and token display
- `f00bfe7` - fix: use DB gate and call planStore.Archive in tryArchivePlan
- `cbf701e` - User Greeting (#3)

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

**Go Install:**
```bash
go install github.com/prasenjeet-symon/ogcode@latest
```

---

*Full changelog: https://github.com/prasenjeet-symon/ogcode/releases/tag/v0.2.5*