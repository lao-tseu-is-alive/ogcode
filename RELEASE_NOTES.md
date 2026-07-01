# Release Notes — v0.13.4

## Anthropic: fix 400 error on no-argument tool calls

When replaying a conversation that contains a tool call with no arguments (or
arguments that round-trip through storage as `null`), the Anthropic Messages API
rejected the request with a `400` error:

```
tool_use.input: Input should be an object
```

Anthropic requires the `input` field of every `tool_use` block to be a JSON
object. A no-arg tool call serialises its arguments as `null`, which unmarshals
to a `nil` Go interface (not a `map[string]any`). The provider now coerces any
non-object `input` — `nil`, scalars, or arrays — to an empty object `{}` before
sending it, preventing the 400.

---

### 🐛 Bug Fixes

- **Anthropic provider:** Coerce `tool_use.input` to `{}` when it is not a JSON
  object, fixing `400 "Input should be an object"` errors on no-argument tool
  calls.

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.13.3...v0.13.4*