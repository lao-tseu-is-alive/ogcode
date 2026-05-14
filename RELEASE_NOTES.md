# 🚀 Ogcode v0.2.6 Release

## 📋 Executive Summary
This patch release enhances agent instructions with better documentation references and stricter rules against unnecessary exploration of dependency directories, improving agent performance and reliability.

---

## 📝 Agent Instructions Improvements

### Documentation Reference
- **Added devdocs.io**: All agent system prompts (BuildAgent, PlanAgent, BreakdownAgent) now reference `https://devdocs.io` as the canonical source for API documentation and library references
- Agents will consult devdocs.io when encountering unfamiliar libraries or APIs

### Dependency Directory Rule
- **New hard rule**: Agents are now explicitly prohibited from exploring or reading package manager and dependency directories
- **Protected directories** include: `node_modules`, `vendor`, `.venv`, `__pycache__`, `dist`
- These directories contain third-party code and should only be accessed when a specific issue explicitly requires it
- This prevents agents from wasting token budget and time on irrelevant third-party code

### Scope Clarification
- Refined the research instruction in BuildAgent to clarify that documentation lookup is for APIs, not dependency source code

---

## 📝 Changes Summary

```
1 file changed, 9 insertions(+), 6 deletions(-)

internal/agent/agent.go | 15 +++++++++++++++
```

### Key Commit
- `cf47011` - docs: add devdocs.io reference and dependency directory rule to agent instructions

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/releases/tag/v0.2.6*
