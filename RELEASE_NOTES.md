# Release Notes — v0.13.3

## GitHub-style file diffs in the UI

File edits made by the agent now render as a clean, GitHub-style unified diff
instead of raw JSON input/output.

---

### ✨ Inline diffs for `write` and `edit` tools

- **New `FileDiff` component.** When the agent uses the `write` or `edit` tool,
  the tool card now shows a unified line-by-line diff with green/red gutter
  colors — the same visual language you'd see in a pull request.
- **Diff stat in the collapsed header.** The tool card header now displays a
  compact `+N −M` add/remove line count, so you can gauge the size of a change
  at a glance without expanding it.
- **"Created" vs "Wrote".** The `write` tool now distinguishes between creating a
  new file and overwriting an existing one — the output verb and diff header
  reflect which happened.
- **Large-file guard.** Files larger than 256 KB are not diffed (the UI shows a
  "File too large to show a diff" message and the tool metadata flags
  `diffOmitted`), keeping message sizes and rendering snappy.
- **Truncation cap.** Diffs are capped at 600 rendered rows with a trailing
  "… N more lines" indicator, so very long changesets don't overwhelm the view.

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.13.2...v0.13.3*