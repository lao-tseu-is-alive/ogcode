# Release Process

This document describes the automated release workflow for ogcode.

## Overview

Releases are fully automated via GitHub Actions. When a version tag is pushed, GoReleaser handles:
- Cross-platform binary builds (macOS, Linux, Windows)
- GitHub Release creation with changelog
- Homebrew tap update
- Winget package submission

---

## Release Checklist

### 1. Update Version
```bash
# Edit "version" in web/package.json
# Semantic versioning:
#   Patch: 0.2.6 → 0.2.7 (bug fixes)
#   Minor: 0.2.6 → 0.3.0 (new features)
#   Major: 0.2.6 → 1.0.0 (breaking changes)
```

### 2. Update Release Notes
Edit `RELEASE_NOTES.md`:
- Update version heading
- Add executive summary
- List all changes with key commits
- Update installation commands section

### 3. Commit Changes
```bash
git add web/package.json RELEASE_NOTES.md
git commit -m "release: bump version to vX.Y.Z"
git push origin main
```

### 4. Create & Push Tag
```bash
git tag vX.Y.Z
git push origin vX.Y.Z
```

### 5. Automated Release (GitHub Actions)
Once the tag is pushed, the [`release.yml`](.github/workflows/release.yml) workflow runs:

| Step | Description |
|------|-------------|
| Checkout | Fetches full repository history |
| Setup Go | Configures Go 1.26.1 |
| Setup Node.js | Configures Node 20 |
| GoReleaser | Builds all platforms, creates GitHub release |
| Winget PR | Submits PR to microsoft/winget-pkgs |

---

## What Gets Released

### Binaries
| Platform | Architectures |
|----------|---------------|
| macOS | amd64, arm64 |
| Linux | amd64, arm64, armv7 |
| Windows | amd64, 386, arm64 |

### Package Managers
| Manager | Repository |
|---------|------------|
| Homebrew | [homebrew-tap](https://github.com/prasenjeet-symon/homebrew-tap) |
| Winget | [microsoft/winget-pkgs](https://github.com/microsoft/winget-pkgs) |

---

## Installation

After release, users can install via:

```bash
# macOS/Linux
curl -fsSL http://ogcode.xyz/install.sh | sh

# Windows
irm http://ogcode.xyz/install.ps1 | iex

# Go
go install github.com/prasenjeet-symon/ogcode@latest

# Homebrew
brew install prasenjeet-symon/tap/ogcode

# Winget
winget install prasenjeet-symon.ogcode
```

---

## Local Development

To build locally without releasing:

```bash
make build        # Build web + server
make install      # Build and install to ~/.local/bin
ogcode --version  # Verify version
make clean        # Clean build artifacts
```

---

## Troubleshooting

### Tag not triggering release
- Ensure tag format is `v*.*.*` (e.g., `v0.2.7`)
- Tags must be pushed to GitHub: `git push origin v0.2.7`

### Winget PR not created
- Requires `WINGET_PKGS_TOKEN` secret in repository settings
- Check GoReleaser step logs for errors

### Build fails locally but passes in CI
- Ensure Go 1.26+ and Node.js 20+ are installed
- Clear caches: `rm -rf web/node_modules web/dist`

---

## Manual Override

If GitHub Actions is unavailable, you can run GoReleaser locally:

```bash
# Requires GITHUB_TOKEN and WINGET_PKGS_TOKEN env vars
goreleaser release --clean
```

**Note:** This requires the secrets to be available locally and will create releases directly.