# 🚀 Ogcode v0.1.11 Release

## 📋 Executive Summary
This patch release focuses on **critical reliability improvements** for the task execution engine, fixing race conditions and crash recovery issues that could leave tasks in inconsistent states.

---

## 🐛 Critical Bug Fixes

### Task Retry & Crash Recovery System

**Bug #2 - Stale Error Messages on Retry**  
`ResetFailed()` now properly clears the `pr_error` field when retrying tasks. Previously, old error messages would persist across retries, causing confusion.

**Bug #3 - Race Condition in DeleteBranch**  
Fixed concurrent access issue by wrapping `git.DeleteBranch()` calls with `s.gitMu.Lock()/Unlock()`. This prevents corruption when multiple task retries happen simultaneously.

**Bug #4 - Orphaned Worktree Cleanup**  
Implemented automatic cleanup of orphaned worktree directories after server crashes. The `FailStuckTasks()` function now returns failed tasks, allowing the server to remove abandoned `.ogcode/worktrees/` directories on startup.

**Impact:** Tasks are now significantly more reliable when retrying failed operations and recovering from unexpected shutdowns.

---

## 🔧 Improvements & Compatibility

### GitHub CLI Support
- Removed `--json` flag from `gh pr create` for backward compatibility with older GitHub CLI versions

### Build Pipeline
- Fixed GoReleaser configuration to prevent automatic commits of winget manifests to the main branch

---

## 📝 Documentation Updates

- Updated Discord community invite link in README

---

## 📊 Changes Summary

```
3 files changed, 60 insertions(+), 10 deletions(-)

internal/task/store.go         | 46 +++++++++++++++++++++++++++--------
internal/server/server.go      | 18 +++++++++++---
internal/server/task_routes.go |  6 ++++-
README.md                      |  4 ++--
.goreleaser.yaml              |  2 +-
```

---

## 🎯 Upgrade Priority: HIGH

**Recommended for all users**, especially those:
- Using task retry functionality frequently
- Running Ogcode in production environments
- Experiencing "stuck" tasks after server restarts

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/releases/tag/v0.1.11*
