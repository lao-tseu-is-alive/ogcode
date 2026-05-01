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

// AnthropicProvider implements Provider for the Anthropic Messages API.
type AnthropicProvider struct {
	apiKey string
	model  string
}

func NewAnthropicProvider() *AnthropicProvider {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	model := os.Getenv("ANTHROPIC_MODEL")
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}
	return &AnthropicProvider{apiKey: apiKey, model: model}
}

func (p *AnthropicProvider) ID() string { return "anthropic" }

func (p *AnthropicProvider) Models() []ModelInfo {
	all := []ModelInfo{
		{ID: "claude-sonnet-4-20250514", Name: "Claude Sonnet 4", ProviderID: "anthropic"},
		{ID: "claude-opus-4-20250514", Name: "Claude Opus 4", ProviderID: "anthropic"},
		{ID: "claude-haiku-4-5-20251001", Name: "Claude Haiku 4.5", ProviderID: "anthropic"},
	}
	for i := range all {
		if all[i].ID == p.model {
			all[i].Default = true
		}
	}
	return all
}

func (p *AnthropicProvider) StreamChat(ctx context.Context, req StreamRequest) (<-chan StreamEvent, error) {
	model := req.Model
	if model == "" {
		model = p.model
	}

	messages := make([]anthropicMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		if m.Role == "system" {
			continue
		}
		var content any
		if err := json.Unmarshal(m.Content, &content); err != nil {
			content = string(m.Content)
		}
		messages = append(messages, anthropicMessage{
			Role:    m.Role,
			Content: content,
		})
	}

	systemPrompt := strings.Join(req.System, "\n\n")

	tools := make([]anthropicTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		})
	}

	body := anthropicRequest{
		Model:      model,
		MaxTokens:  max(req.MaxTokens, 4096),
		System:     systemPrompt,
		Messages:   messages,
		Tools:      tools,
		Stream:     true,
		Temperature: req.Temperature,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 600 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	slog.Info("anthropic stream connected", "model", model, "status", resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic API error %d: %s", resp.StatusCode, string(body))
	}

	ch := make(chan StreamEvent, 256)
	go p.streamEvents(resp.Body, ch)
	return ch, nil
}

func (p *AnthropicProvider) streamEvents(body io.ReadCloser, ch chan<- StreamEvent) {
	defer body.Close()
	defer close(ch)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var currentToolID string
	var currentToolName string
	usage := TokenUsage{}
	usageDirty := false

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var evt anthropicEvent
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "message_start":
			if evt.Message != nil && evt.Message.Usage != nil {
				usage.InputTokens = evt.Message.Usage.InputTokens
				usage.OutputTokens = evt.Message.Usage.OutputTokens
				usage.CacheReadTokens = evt.Message.Usage.CacheReadInputTokens
				usage.CacheWriteTokens = evt.Message.Usage.CacheCreationInputTokens
				usageDirty = true
			}
		case "content_block_start":
			if evt.ContentBlock != nil && evt.ContentBlock.Type == "tool_use" {
				currentToolID = evt.ContentBlock.ID
				currentToolName = evt.ContentBlock.Name
				var input json.RawMessage = evt.ContentBlock.Input
				if input == nil {
					input = json.RawMessage("{}")
				}
				ch <- StreamEvent{
					Type:       EventToolCallStart,
					ToolCallID: currentToolID,
					ToolName:   currentToolName,
					ToolInput:  input,
				}
			}
		case "content_block_delta":
			if evt.Delta != nil {
				switch evt.Delta.Type {
				case "text_delta":
					ch <- StreamEvent{Type: EventTextDelta, Text: evt.Delta.Text}
				case "input_json_delta":
					if currentToolID != "" {
						ch <- StreamEvent{
							Type:       EventToolCallDelta,
							ToolCallID: currentToolID,
							ToolName:   currentToolName,
							ToolInput:  []byte(evt.Delta.PartialJson),
						}
					}
				case "thinking_delta":
					ch <- StreamEvent{Type: EventReasoning, Text: evt.Delta.Thinking}
				}
			}
		case "content_block_stop":
			if currentToolID != "" {
				ch <- StreamEvent{Type: EventToolCallEnd, ToolCallID: currentToolID, ToolName: currentToolName}
				currentToolID = ""
				currentToolName = ""
			}
		case "message_stop":
			if usageDirty {
				u := usage
				ch <- StreamEvent{Type: EventUsage, Usage: &u}
				usageDirty = false
			}
			reason := "stop"
			ch <- StreamEvent{Type: EventFinish, FinishReason: &reason}
		case "message_delta":
			if evt.Usage != nil {
				// message_delta carries the final OutputTokens count.
				if evt.Usage.OutputTokens > 0 {
					usage.OutputTokens = evt.Usage.OutputTokens
					usageDirty = true
				}
			}
			if evt.Delta != nil && evt.Delta.StopReason != "" {
				reason := evt.Delta.StopReason
				ch <- StreamEvent{Type: EventFinish, FinishReason: &reason}
			}
		case "error":
			ch <- StreamEvent{Type: EventError, Error: evt.Error}
		}
	}
}

// Anthropic API types

type anthropicRequest struct {
	Model      string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	System     string            `json:"system,omitempty"`
	Messages   []anthropicMessage `json:"messages"`
	Tools      []anthropicTool   `json:"tools,omitempty"`
	Stream     bool              `json:"stream"`
	Temperature float64         `json:"temperature,omitempty"`
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicEvent struct {
	Type         string                 `json:"type"`
	Index        int                    `json:"index,omitempty"`
	Message      *anthropicMessageInfo  `json:"message,omitempty"`
	ContentBlock *anthropicContentBlock `json:"content_block,omitempty"`
	Delta        *anthropicDelta        `json:"delta,omitempty"`
	Usage        *anthropicUsage        `json:"usage,omitempty"`
	Error        string                 `json:"error,omitempty"`
}

type anthropicMessageInfo struct {
	Usage *anthropicUsage `json:"usage,omitempty"`
}

type anthropicUsage struct {
	InputTokens              int `json:"input_tokens,omitempty"`
	OutputTokens             int `json:"output_tokens,omitempty"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

type anthropicContentBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
	Text  string          `json:"text,omitempty"`
}

type anthropicDelta struct {
	Type         string `json:"type"`
	Text         string `json:"text,omitempty"`
	PartialJson  string `json:"partial_json,omitempty"`
	Thinking     string `json:"thinking,omitempty"`
	StopReason   string `json:"stop_reason,omitempty"`
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}