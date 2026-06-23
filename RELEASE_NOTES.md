# Release Notes — v0.10.0

## Onboarding Wizard & Live Provider Hot-Reload

This release adds a **first-run onboarding wizard** that walks new users through connecting their first LLM provider, a **credential validation** flow that tests an API key before saving it, and a **runtime provider hot-reload** so credential changes take effect immediately — no server restart required.

---

### ✨ New Features

- **Onboarding wizard** (`web/src/pages/onboarding.tsx`) — First-run users with no provider configured are redirected to a guided wizard that lets them pick a provider (Anthropic, OpenAI, OpenRouter, or Ollama), enter credentials, optionally test the key, and save. The wizard handles both keyless local Ollama and remote/cloud Ollama that needs an API key.
- **Onboarding context & redirect gate** (`web/src/context/onboarding.tsx`) — A new SolidJS context tracks whether any provider is configured. `OnboardingGate` redirects first-run users to the wizard and respects a "Skip for now" dismissal so it never traps users in a loop. Fails open if the config check errors.
- **Credential validation** (`internal/provider/validate.go`) — `ValidateCredentials` makes a minimal chat request with the supplied key to confirm the provider accepts it, returning a clear error otherwise. Used by the settings/onboarding "test key" flow.
- **Runtime provider hot-reload** (`internal/server/server.go`) — `loadProviderMap()` and `reloadProviders()` rebuild the provider registry from the current DB + env credentials and swap it into the running server in place. The shared `*provider.Registry` pointer is preserved, and custom-model routing survives the swap. Saving a key in settings or completing the wizard now makes the provider's models available immediately.
- **Default provider priority** (`internal/provider/provider.go`) — `Registry.Default()` returns the highest-priority registered provider (anthropic → openai → openrouter → ollama), so the loop runner picks up runtime credential changes instead of always falling back to the immutable startup default.

### 🔧 Backend

- **Thread-safe registry** — `Registry` now guards its `providers` map with a read/write lock. All read paths (`Get`, `List`, `ListModels`, `ResolveProvider`, `ModelSupportsImages`, `RefreshModels`) take a read lock or operate on a snapshot, and `ReplaceProviders` takes a write lock. This makes concurrent model lookups safe against hot-reload swaps.
- **`POST /api/providers/config/{id}/validate`** (`internal/server/provider_routes.go`) — New endpoint that tests credentials without persisting them. The `"__SET__"` sentinel resolves to the stored key so a saved provider can be re-tested without re-entering the key. Always responds `200` with `{ok, error?}` for inline UI rendering.
- **In-place reload on save** — `handleSetProviderConfig` now calls `reloadProviders()` after persisting, so newly saved credentials are live immediately.
- **Loop runner respects registry default** (`internal/agent/loop.go`) — `RunLoop` and `RunSearchSession` now prefer `Registry.Default()` over the startup `DefaultProvider`, so credential changes applied at runtime take effect for new sessions.

### 🎨 Web UI

- **Onboarding route & gate** (`web/src/app.tsx`) — New `/onboarding` route and `OnboardingGate` component redirect first-run users to the wizard. `OnboardingProvider` wraps the app tree.
- **Validate API client** (`web/src/api/client.ts`) — New `validateProviderConfig()` helper and `ValidateResult` type for the test-key flow.

### 🧪 Tests

- **`internal/provider/registry_test.go`** — Covers `ReplaceProviders` hot-reload (add/drop providers), preservation of custom-model routing across a swap, `Default()` priority ordering, and a concurrent reader/writer race test (run with `-race`).
- **`internal/server/provider_routes_test.go`** — End-to-end HTTP test verifying that with no provider configured the model list is empty, that POSTing an Anthropic key hot-reloads the provider and its models appear immediately, and that the validate endpoint always returns a well-formed `{ok, error}` body.

### 📁 Files Changed

**New:** `internal/provider/validate.go`, `internal/provider/registry_test.go`, `internal/server/provider_routes_test.go`, `web/src/context/onboarding.tsx`, `web/src/pages/onboarding.tsx`

**Modified (backend):** `internal/agent/loop.go`, `internal/provider/provider.go`, `internal/server/provider_routes.go`, `internal/server/routes.go`, `internal/server/server.go`, `internal/cli/version.go`, `internal/version/version.go`

**Modified (web):** `web/src/api/client.ts`, `web/src/app.tsx`

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

*Full changelog: https://github.com/prasenjeet-symon/ogcode/compare/v0.9.2...v0.10.0*