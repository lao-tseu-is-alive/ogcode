# Release Guide

This document outlines the complete release process for ogcode.

## Release Checklist

### 1. Update Version
Update the version in `web/package.json`:
```bash
# Edit the "version" field in web/package.json
# Example: 0.2.6 → 0.2.7 (patch) or 0.3.0 (minor)
```

### 2. Update Release Notes
Update `RELEASE_NOTES.md` with:
- Version number
- Executive summary
- List of changes
- Key commits
- Updated installation commands

### 3. Commit Changes
```bash
git add web/package.json RELEASE_NOTES.md
git commit -m "release: bump version to vX.Y.Z"
```

### 4. Build the Project
```bash
make build
```
This runs:
- `make build-web` - Builds the SolidJS frontend
- `make build-server` - Builds the Go binary with version embedded

### 5. Install Globally
```bash
make install
```
This copies the `ogcode` binary to `~/.local/bin/ogcode`.

### 6. Verify Installation
```bash
ogcode --version
# Should output: ogcode version X.Y.Z
```

### 7. Push to GitHub
```bash
# Push the release commit
git push origin main

# Create and push the git tag
git tag vX.Y.Z
git push origin vX.Y.Z
```

### 8. Create GitHub Release
1. Go to https://github.com/prasenjeet-symon/ogcode/releases
2. Click "Draft a new release"
3. Select the `vX.Y.Z` tag
4. Paste the release notes
5. Click "Publish release"

---

## What NOT to Do

### ❌ Do NOT run `npm publish`
The web package is marked as `"private": true` in `web/package.json`. Publishing to npm is not supported and will fail.

### ❌ Do NOT commit built artifacts
The following are automatically generated and should NOT be committed:
- `ogcode` (binary)
- `web/dist/` (frontend bundle)
- `web/node_modules/` (dependencies)

These are already listed in `.gitignore`.

---

## Versioning Convention

| Change Type | Example | When to Use |
|-------------|---------|-------------|
| Patch | 0.2.6 → 0.2.7 | Bug fixes, small improvements |
| Minor | 0.2.6 → 0.3.0 | New features, backwards compatible |
| Major | 0.2.6 → 1.0.0 | Breaking changes |

---

## Quick Release Command Sequence

```bash
# 1. Update version in web/package.json
vim web/package.json

# 2. Update RELEASE_NOTES.md
vim RELEASE_NOTES.md

# 3. Commit
git add web/package.json RELEASE_NOTES.md
git commit -m "release: bump version to vX.Y.Z"

# 4. Build and install
make build && make install

# 5. Verify
ogcode --version

# 6. Push
git push origin main
git tag vX.Y.Z
git push origin vX.Y.Z

# 7. Create GitHub release (via web UI)
# https://github.com/prasenjeet-symon/ogcode/releases/new
```

---

## Troubleshooting

### Binary not found after install
Ensure `~/.local/bin` is in your PATH:
```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Build fails
- Ensure Go is installed (v1.21+)
- Ensure Node.js is installed (v18+)
- Clear caches: `rm -rf web/node_modules web/dist`

### Version shows "dev"
This means the version wasn't read from `web/package.json` correctly. Ensure:
- Node.js is installed
- `package.json` exists and has valid JSON