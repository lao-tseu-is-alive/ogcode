# Release Notes — v0.11.2

## Installs and runs again on a clean machine

A reliability release. Prebuilt binaries for **every** install path — the
`install.sh` one-liner, Homebrew, winget, direct download, and the Docker
image — were shipping broken and **panicked on startup for new users**. This
release fixes the build/release pipeline so a fresh install just works.

---

### 🐛 Fixes

- **Release binaries no longer panic on startup.** ogcode links
  `github.com/gen2brain/go-fitz` (PDF/MuPDF). The release was built with
  `CGO_ENABLED=0`, so go-fitz fell back to its pure-Go path and tried to
  `dlopen` `libmupdf.{dylib,so,dll}` at package-init — which isn't present on a
  clean machine — making **every** command (even `ogcode version` / `--help`)
  panic before `main` ran. Releases now build with `CGO_ENABLED=1`, statically
  linking the bundled libmupdf so no runtime library is needed.

- **Native, per-platform release build matrix.** GoReleaser OSS can't reliably
  CGO-cross-compile that C codebase from one runner (and has no `prebuilt`
  builder), so each target is now built on its own native runner — linux
  (amd64 + arm64), darwin (amd64 + arm64), windows (amd64 + arm64) — and
  packaged via a small `builds.tool` shim (`tools/gobuild-shim.sh`) so
  archives, checksums, the Homebrew formula and the winget manifest are
  produced exactly as before.

- **Docker image fixed too** (`Dockerfile`). It was also `CGO_ENABLED=0` on
  Alpine and panicked the same way; it now builds `CGO_ENABLED=1 -tags musl`
  (linking `libmupdf_linux_<arch>_musl.a`), and CI runs a smoke test that
  actually starts the image so a startup panic fails the build before publish.

- **Web-search bridge packaging.** The release archive collapsed
  `search-bridge/{server.js,package.json}` into a single file (dropping
  `package.json` and breaking the Homebrew `install` block). It's now a proper
  `search-bridge/` directory, and `install.sh` extracts it to the right place.

- **`install.sh` on ogcode.xyz refreshed.** The served `docs/install.sh` was
  stale (missing the web-search setup) and is now in sync with the repo.

### 📁 Files Changed

**Modified:** `.github/workflows/release.yml`, `.github/workflows/docker.yml`,
`.goreleaser.yaml`, `Dockerfile`, `install.sh`, `docs/install.sh`,
`internal/cli/version.go`, `internal/version/version.go`, `web/package.json`,
`web/package-lock.json`
**Added:** `tools/gobuild-shim.sh`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.11.1...v0.11.2*
