package provider

import (
	"context"
	"encoding/json"
	"sync"
)

type StreamEventType string

const (
	EventTextDelta      StreamEventType = "text-delta"
	EventToolCallStart  StreamEventType = "tool-call-start"
	EventToolCallDelta  StreamEventType = "tool-call-delta"
	EventToolCallEnd    StreamEventType = "tool-call-end"
	EventReasoning      StreamEventType = "reasoning"
	EventFinish         StreamEventType = "finish"
	EventUsage          StreamEventType = "usage"
	EventError          StreamEventType = "error"
)

// TokenUsage carries per-message token accounting from a provider.
// Fields are non-zero where the provider reports them.
type TokenUsage struct {
	InputTokens      int `json:"inputTokens,omitempty"`
	OutputTokens     int `json:"outputTokens,omitempty"`
	ReasoningTokens  int `json:"reasoningTokens,omitempty"`
	CacheReadTokens  int `json:"cacheReadTokens,omitempty"`
	CacheWriteTokens int `json:"cacheWriteTokens,omitempty"`
}

type StreamEvent struct {
	Type         StreamEventType  `json:"type"`
	Text         string           `json:"text,omitempty"`
	ToolCallID   string           `json:"toolCallId,omitempty"`
	ToolName     string           `json:"toolName,omitempty"`
	ToolInput    json.RawMessage  `json:"toolInput,omitempty"`
	FinishReason *string          `json:"finishReason,omitempty"`
	Usage        *TokenUsage      `json:"usage,omitempty"`
	Error        string           `json:"error,omitempty"`
}

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

type ModelMessage struct {
	Role       string          `json:"role"`
	Content    json.RawMessage `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type ToolDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type StreamRequest struct {
	Model       string           `json:"model"`
	System      []string        `json:"system"`
	Messages    []ModelMessage  `json:"messages"`
	Tools       []ToolDefinition `json:"tools"`
	Temperature float64          `json:"temperature,omitempty"`
	MaxTokens   int              `json:"maxTokens,omitempty"`
	Abort       context.Context  `json:"-"`
}

type ModelInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProviderID string `json:"providerId"`
	Default    bool   `json:"default"`
}

type Provider interface {
	ID() string
	Models() []ModelInfo
	StreamChat(ctx context.Context, req StreamRequest) (<-chan StreamEvent, error)
}

// ModelRefresher is an optional interface that providers can implement
// to support dynamic model list refreshing.
type ModelRefresher interface {
	RefreshModels()
}

type Registry struct {
	providers    map[string]Provider
	customModels map[string]string // modelID -> providerID
	customMu     sync.RWMutex
}

func NewRegistry() *Registry {
	return &Registry{
		providers:    make(map[string]Provider),
		customModels: make(map[string]string),
	}
}

func (r *Registry) Register(p Provider) {
	r.providers[p.ID()] = p
}

func (r *Registry) Get(id string) Provider {
	return r.providers[id]
}

func (r *Registry) List() []string {
	var ids []string
	for id := range r.providers {
		ids = append(ids, id)
	}
	return ids
}

func (r *Registry) ListModels() []ModelInfo {
	var models []ModelInfo
	for _, p := range r.providers {
		models = append(models, p.Models()...)
	}
	return models
}

func (r *Registry) RegisterCustomModel(modelID, providerID string) {
	r.customMu.Lock()
	r.customModels[modelID] = providerID
	r.customMu.Unlock()
}

func (r *Registry) UnregisterCustomModel(modelID string) {
	r.customMu.Lock()
	delete(r.customModels, modelID)
	r.customMu.Unlock()
}

func (r *Registry) ResolveProvider(modelID string) Provider {
	// Check custom model routing first
	r.customMu.RLock()
	providerID, customOk := r.customModels[modelID]
	r.customMu.RUnlock()
	if customOk {
		if p := r.providers[providerID]; p != nil {
			return p
		}
	}
	// Then check built-in models
	for _, p := range r.providers {
		for _, m := range p.Models() {
			if m.ID == modelID {
				return p
			}
		}
	}
	// Fallback to first provider
	for _, p := range r.providers {
		return p
	}
	return nil
}

// RefreshModels clears cached model lists for all providers that support it,
// forcing re-fetch on next Models() call.
func (r *Registry) RefreshModels() {
	for _, p := range r.providers {
		if refresher, ok := p.(ModelRefresher); ok {
			refresher.RefreshModels()
		}
	}
}