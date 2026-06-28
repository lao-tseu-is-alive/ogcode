# Release Notes — v0.11.1

## Agentic Memory On by Default

A follow-up to v0.11.0: **agentic session memory is now enabled by default**. Since the embedder is fully built-in and needs zero setup (no API key, no external service), there's no reason for memory to be opt-in — so it now works out of the box on a fresh install.

---

### 🐛 Fixes

- **Memory enabled by default** (`internal/session/memory_store.go`) — When no memory-config row exists, `GetMemoryConfig` now returns an **enabled** config instead of a disabled one. v0.11.0 removed the memory enable/disable settings card (the local embedder made it redundant) but left the backend defaulting to *off* with no in-app way to turn it on, so memory never actually engaged. It now engages automatically: the server wires up the local embedder and the `memory_recall` tool at startup, and Build/Plan agent sessions read, write, and embed turn context as intended.

  An explicit opt-out is still honored — a user who disables memory via `POST /api/memory/config` persists an `enabled = 0` row, which is respected. No schema migration is required (the `enabled` column is always written explicitly by `SetMemoryConfig`).

### 📁 Files Changed

**Modified:** `internal/session/memory_store.go`, `internal/cli/version.go`, `internal/version/version.go`, `web/package.json`, `web/package-lock.json`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.11.0...v0.11.1*
