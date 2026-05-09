# Release Notes - v0.1.11

## Overview
This release includes critical bug fixes for task retry and crash recovery, along with minor improvements to GitHub CLI compatibility and winget publishing.

---

## 🐛 Bug Fixes

### Task Retry & Crash Recovery Issues
Fixed three critical bugs affecting task reliability and system stability:

- **Bug #2 - pr_error persistence on retry**: Fixed `ResetFailed()` function to properly clear the `pr_error` field when retrying failed tasks, preventing stale error messages from persisting
- **Bug #3 - Race condition in DeleteBranch**: Fixed race condition by wrapping `git.DeleteBranch()` in `handleRetryTask()` with proper mutex locking (`s.gitMu.Lock()/Unlock()`)
- **Bug #4 - Orphaned worktree cleanup**: Added worktree cleanup after server crash recovery - `FailStuckTasks()` now returns failed tasks so orphaned worktree directories can be removed on startup

**Files Changed:**
- `internal/server/server.go` (+18 lines)
- `internal/server/task_routes.go` (+6 lines)
- `internal/task/store.go` (+46 lines)

---

## 🔧 Improvements

### GitHub CLI Compatibility
- **PR #XX**: Removed `--json` flag from `gh pr create` command for compatibility with older GitHub CLI versions
- **GoReleaser**: Fixed configuration to prevent committing winget manifests back to main branch

---

## 📝 Documentation

### Community
- Updated Discord invite link in README to active community server

---

## 📁 Files Changed

```
internal/server/server.go      | 18 ++++++++++++++---
internal/server/task_routes.go |  6 +++++-
internal/task/store.go         | 46 ++++++++++++++++++++++++++++++++++++------
README.md                      |  4 ++--
.goreleaser.yaml              |  2 +-
```

---

## 🎯 Focus Areas
- **Reliability**: Tasks now properly reset state on retry and clean up resources after crashes
- **Compatibility**: Better support for various GitHub CLI versions
- **Stability**: Eliminated race conditions in concurrent git operations

---

*This release builds on v0.1.10 and is recommended for all users experiencing task retry or crash recovery issues.*
