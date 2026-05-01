package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"
)

// OpenAIProvider implements Provider for the OpenAI Chat Completions API.
// Also used for OpenRouter (same API format, different base URL).
type OpenAIProvider struct {
	id      string
	apiKey  string
	model   string
	baseURL string
}

func NewOpenAIProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENAI_API_KEY")
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-4o"
	}
	baseURL := os.Getenv("OPENAI_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{id: "openai", apiKey: apiKey, model: model, baseURL: baseURL}
}

// NewOpenRouterProvider creates an OpenAI-compatible provider for OpenRouter.
func NewOpenRouterProvider() *OpenAIProvider {
	apiKey := os.Getenv("OPENROUTER_API_KEY")
	model := os.Getenv("OPENROUTER_MODEL")
	if model == "" {
		model = "anthropic/claude-sonnet-4.6"
	}
	return &OpenAIProvider{
		id:      "openrouter",
		apiKey:  apiKey,
		model:   model,
		baseURL: "https://openrouter.ai/api/v1",
	}
}

// NewOllamaProvider creates an OpenAI-compatible provider for Ollama.
func NewOllamaProvider() *OpenAIProvider {
	baseURL := os.Getenv("OLLAMA_BASE_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434/v1"
	}
	apiKey := os.Getenv("OLLAMA_API_KEY")
	model := os.Getenv("OLLAMA_MODEL")
	if model == "" {
		model = "qwen3"
	}
	return &OpenAIProvider{
		id:      "ollama",
		apiKey:  apiKey,
		model:   model,
		baseURL: baseURL,
	}
}

func (p *OpenAIProvider) ID() string { return p.id }

func (p *OpenAIProvider) Models() []ModelInfo {
	var list []ModelInfo
	if p.id == "openrouter" {
		list = []ModelInfo{
			{ID: "anthropic/claude-sonnet-4.6", Name: "Claude Sonnet 4.6", ProviderID: "openrouter"},
			{ID: "anthropic/claude-opus-4.5", Name: "Claude Opus 4.5", ProviderID: "openrouter"},
			{ID: "anthropic/claude-haiku-4.5", Name: "Claude Haiku 4.5", ProviderID: "openrouter"},
			{ID: "openai/gpt-5.2-chat", Name: "GPT-5.2 Chat", ProviderID: "openrouter"},
			{ID: "google/gemini-2.5-pro", Name: "Gemini 2.5 Pro", ProviderID: "openrouter"},
			{ID: "deepseek/deepseek-r1", Name: "DeepSeek R1", ProviderID: "openrouter"},
			{ID: "minimax/minimax-m2.5", Name: "MiniMax M2.5", ProviderID: "openrouter"},
		}
	} else if p.id == "ollama" {
		list = []ModelInfo{
			{ID: "glm-5.1", Name: "GLM-5.1", ProviderID: "ollama"},
			{ID: "kimi-k2.6", Name: "Kimi K2.6", ProviderID: "ollama"},
			{ID: "deepseek-v4-flash", Name: "DeepSeek V4 Flash", ProviderID: "ollama"},
				{ID: "deepseek-v4-pro", Name: "DeepSeek V4 Pro", ProviderID: "ollama"},
			{ID: "qwen3-coder-next", Name: "Qwen3 Coder Next", ProviderID: "ollama"},
			{ID: "qwen3.6", Name: "Qwen3.6", ProviderID: "ollama"},
			{ID: "qwen3", Name: "Qwen3", ProviderID: "ollama"},
			{ID: "llama3.1", Name: "Llama 3.1", ProviderID: "ollama"},
			{ID: "codellama", Name: "Code Llama", ProviderID: "ollama"},
			{ID: "deepseek-coder-v2", Name: "DeepSeek Coder V2", ProviderID: "ollama"},
			{ID: "mistral", Name: "Mistral", ProviderID: "ollama"},
		}
	} else {
		list = []ModelInfo{
			{ID: "gpt-5.2-chat", Name: "GPT-5.2 Chat", ProviderID: "openai"},
			{ID: "gpt-5.1", Name: "GPT-5.1", ProviderID: "openai"},
			{ID: "gpt-5-mini", Name: "GPT-5 Mini", ProviderID: "openai"},
			{ID: "o4-mini", Name: "o4 Mini", ProviderID: "openai"},
		}
	}
	for i := range list {
		if list[i].ID == p.model {
			list[i].Default = true
		}
	}
	return list
}

func (p *OpenAIProvider) StreamChat(ctx context.Context, req StreamRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]oaiMessage, 0, len(req.Messages)+len(req.System))
	if len(req.System) > 0 {
		messages = append(messages, oaiMessage{
			Role:    "system",
			Content: strings.Join(req.System, "\n\n"),
		})
	}
	for _, m := range req.Messages {
		msg := oaiMessage{Role: m.Role}
		if m.ToolCallID != "" {
			// Tool result message: role=tool, content=output, tool_call_id, name
			msg.ToolCallID = m.ToolCallID
			msg.Name = m.Name
			var content any
			if err := json.Unmarshal(m.Content, &content); err != nil {
				content = string(m.Content)
			}
			msg.Content = content
		} else if len(m.ToolCalls) > 0 {
			// Assistant message with tool calls
			msg.ToolCalls = m.ToolCalls
			if len(m.Content) > 0 {
				var content any
				if err := json.Unmarshal(m.Content, &content); err != nil {
					msg.Content = string(m.Content)
				} else {
					msg.Content = content
				}
			} else {
				// OpenAI requires content to be null (not omitted) when only tool calls
				msg.Content = nil
			}
		} else {
			var content any
			if err := json.Unmarshal(m.Content, &content); err != nil {
				content = string(m.Content)
			}
			msg.Content = content
		}
		messages = append(messages, msg)
	}

	tools := make([]oaiTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, oaiTool{
			Type: "function",
			Function: oaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}

	body := oaiRequest{
		Model:       model,
		Messages:    messages,
		Tools:       tools,
		Stream:      true,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}
	// stream_options.include_usage is supported by OpenAI, OpenRouter, and Ollama (v0.5+).
	// The final chunk will contain a usage object alongside an empty choices array.
	body.StreamOptions = &oaiStreamOptions{IncludeUsage: true}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := strings.TrimRight(p.baseURL, "/") + "/chat/completions"
	slog.Info("streaming chat request", "provider", p.id, "model", model, "url", url)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.apiKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	}
	if p.id == "openrouter" {
		httpReq.Header.Set("HTTP-Referer", "https://ogcode.dev")
		httpReq.Header.Set("X-Title", "ogcode")
	}

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		slog.Error("API error response", "provider", p.id, "status", resp.StatusCode, "body", string(body))
		return nil, fmt.Errorf("openai API error %d: %s", resp.StatusCode, string(body))
	}
	slog.Info("stream connected", "provider", p.id, "model", model)

	ch := make(chan StreamEvent, 256)
	go p.streamEvents(resp.Body, ch)
	return ch, nil
}

func (p *OpenAIProvider) streamEvents(body io.ReadCloser, ch chan<- StreamEvent) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// Track active tool calls by index so we can match deltas
	activeToolCalls := make(map[int]string) // index -> callID

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var evt oaiStreamResponse
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		// Usage chunks (with stream_options.include_usage) typically arrive
		// in the final SSE chunk and may have zero choices. Surface them as
		// a separate event before the stream closes.
		if evt.Usage != nil {
			usage := &TokenUsage{
				InputTokens:  evt.Usage.PromptTokens,
				OutputTokens: evt.Usage.CompletionTokens,
			}
			if evt.Usage.PromptTokensDetails != nil {
				usage.CacheReadTokens = evt.Usage.PromptTokensDetails.CachedTokens
			}
			if evt.Usage.CompletionTokensDetails != nil {
				usage.ReasoningTokens = evt.Usage.CompletionTokensDetails.ReasoningTokens
			}
			ch <- StreamEvent{Type: EventUsage, Usage: usage}
		}

		if len(evt.Choices) == 0 {
			continue
		}

		choice := evt.Choices[0]
		delta := choice.Delta

		if delta == nil {
			// Still check finish_reason
			if choice.FinishReason != nil && *choice.FinishReason != "" {
				ch <- StreamEvent{Type: EventFinish, FinishReason: choice.FinishReason}
			}
			continue
		}

		if delta.Content != "" {
			ch <- StreamEvent{Type: EventTextDelta, Text: delta.Content}
		}

		if len(delta.ToolCalls) > 0 {
			for _, tc := range delta.ToolCalls {
				if tc.ID != "" {
					// New tool call starting
					activeToolCalls[tc.Index] = tc.ID
					ch <- StreamEvent{
						Type:       EventToolCallStart,
						ToolCallID: tc.ID,
						ToolName:   tc.Function.Name,
						ToolInput:  []byte(tc.Function.Arguments),
					}
				} else if tc.Function.Arguments != "" {
					// Argument delta — use the tracked callID
					callID := activeToolCalls[tc.Index]
					ch <- StreamEvent{
						Type:       EventToolCallDelta,
						ToolCallID: callID,
						ToolInput:  []byte(tc.Function.Arguments),
					}
				}
			}
		}

		if delta.ReasoningContent != "" {
			ch <- StreamEvent{Type: EventReasoning, Text: delta.ReasoningContent}
		}
		if delta.Reasoning != "" {
			ch <- StreamEvent{Type: EventReasoning, Text: delta.Reasoning}
		}

		// Check finish reason on the same chunk as the delta
		if choice.FinishReason != nil && *choice.FinishReason != "" {
			ch <- StreamEvent{Type: EventFinish, FinishReason: choice.FinishReason}
		}
	}
}

// OpenAI API types

type oaiRequest struct {
	Model         string             `json:"model"`
	Messages      []oaiMessage       `json:"messages"`
	Tools         []oaiTool          `json:"tools,omitempty"`
	Stream        bool               `json:"stream"`
	StreamOptions *oaiStreamOptions  `json:"stream_options,omitempty"`
	Temperature   float64            `json:"temperature,omitempty"`
	MaxTokens     int                `json:"max_tokens,omitempty"`
}

type oaiStreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type oaiMessage struct {
	Role       string          `json:"role"`
	Content    any             `json:"content,omitempty"`
	ToolCalls  json.RawMessage `json:"tool_calls,omitempty"`
	ToolCallID string          `json:"tool_call_id,omitempty"`
	Name       string          `json:"name,omitempty"`
}

type oaiTool struct {
	Type     string      `json:"type"`
	Function oaiFunction `json:"function"`
}

type oaiFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
}

type oaiStreamResponse struct {
	ID      string      `json:"id,omitempty"`
	Choices []oaiChoice `json:"choices"`
	Usage   *oaiUsage   `json:"usage,omitempty"`
}

type oaiUsage struct {
	PromptTokens            int                       `json:"prompt_tokens"`
	CompletionTokens        int                       `json:"completion_tokens"`
	TotalTokens             int                       `json:"total_tokens"`
	PromptTokensDetails     *oaiPromptTokensDetails   `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails *oaiCompletionTokenDetails `json:"completion_tokens_details,omitempty"`
}

type oaiPromptTokensDetails struct {
	CachedTokens int `json:"cached_tokens"`
}

type oaiCompletionTokenDetails struct {
	ReasoningTokens int `json:"reasoning_tokens"`
}

type oaiChoice struct {
	Index        int           `json:"index"`
	Delta        *oaiDelta     `json:"delta,omitempty"`
	FinishReason *string       `json:"finish_reason,omitempty"`
}

type oaiDelta struct {
	Role             string           `json:"role,omitempty"`
	Content          string           `json:"content,omitempty"`
	ToolCalls        []oaiToolCallDelta `json:"tool_calls,omitempty"`
	ReasoningContent string           `json:"reasoning_content,omitempty"`
	Reasoning        string           `json:"reasoning,omitempty"`
}

type oaiToolCallDelta struct {
	Index    int              `json:"index"`
	ID       string           `json:"id,omitempty"`
	Type     string           `json:"type,omitempty"`
	Function oaiFunctionDelta `json:"function"`
}

type oaiFunctionDelta struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}