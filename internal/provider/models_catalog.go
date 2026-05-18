package provider

// =============================================================================
// MODEL CATALOG
// =============================================================================
//
// This file is the single place to add, remove, or update models for Anthropic
// and OpenAI. No other file needs to change.
//
// HOW TO ADD A MODEL
//   1. Find the right provider section below.
//   2. Append a CatalogModel entry.
//   3. Set ActiveByDefault: true only for broadly-useful, stable models.
//      Keep the active-by-default list small — users can enable others in settings.
//
// HOW TO RETIRE A MODEL
//   Remove or comment out the entry. Existing user preferences referencing the
//   old ID are ignored gracefully (the model simply won't appear in the list).
//
// FIELDS
//   ID              — exact model ID sent to the API
//   Name            — human-readable label shown in the UI
//   ActiveByDefault — whether the model is enabled without any user action
//
// =============================================================================

// CatalogModel is a statically-known model for a provider that does not expose
// a live /v1/models discovery endpoint.
type CatalogModel struct {
	ID              string
	Name            string
	ActiveByDefault bool
}

// AnthropicModels is the authoritative list of Anthropic models.
// Maintained by contributors — see file header for instructions.
var AnthropicModels = []CatalogModel{
	// ── Claude 4 family ──────────────────────────────────────────────────────
	{ID: "claude-opus-4-7", Name: "Claude Opus 4.7", ActiveByDefault: true},
	{ID: "claude-opus-4-6", Name: "Claude Opus 4.6", ActiveByDefault: true},
	{ID: "claude-sonnet-4-6", Name: "Claude Sonnet 4.6", ActiveByDefault: true},
	{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", ActiveByDefault: true},

	// ── Claude 4 intermediate releases ───────────────────────────────────────
	{ID: "claude-opus-4-5-20251101", Name: "Claude Opus 4.5", ActiveByDefault: false},
	{ID: "claude-opus-4-1-20250805", Name: "Claude Opus 4.1", ActiveByDefault: false},
	{ID: "claude-sonnet-4-5-20250929", Name: "Claude Sonnet 4.5", ActiveByDefault: false},

	// ── Claude 4 (older releases) ────────────────────────────────────────────
	{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", ActiveByDefault: false},
	{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", ActiveByDefault: false},
}

// OpenAIModels is the authoritative list of OpenAI models.
// Maintained by contributors — see file header for instructions.
var OpenAIModels = []CatalogModel{
	// ── GPT-5 family ────────────────────────────────────────────────────────
	{ID: "gpt-5", Name: "GPT-5", ActiveByDefault: true},
	{ID: "gpt-5-mini", Name: "GPT-5 Mini", ActiveByDefault: true},
	{ID: "gpt-5-nano", Name: "GPT-5 Nano", ActiveByDefault: false},

	// ── GPT-4.1 family ─────────────────────────────────────────────────────
	{ID: "gpt-4.1", Name: "GPT-4.1", ActiveByDefault: true},
	{ID: "gpt-4.1-mini", Name: "GPT-4.1 Mini", ActiveByDefault: true},
	{ID: "gpt-4.1-nano", Name: "GPT-4.1 Nano", ActiveByDefault: false},

	// ── GPT-4o family ───────────────────────────────────────────────────────
	{ID: "gpt-4o", Name: "GPT-4o", ActiveByDefault: false},
	{ID: "gpt-4o-mini", Name: "GPT-4o Mini", ActiveByDefault: false},

	// ── Reasoning ────────────────────────────────────────────────────────────
	{ID: "o4-mini", Name: "o4 Mini", ActiveByDefault: true},
	{ID: "o3", Name: "o3", ActiveByDefault: true},
	{ID: "o3-mini", Name: "o3 Mini", ActiveByDefault: false},
	{ID: "o1", Name: "o1", ActiveByDefault: false},
	{ID: "o1-mini", Name: "o1 Mini", ActiveByDefault: false},
}
