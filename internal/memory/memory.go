package memory

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/ogcode/ogcode/internal/mcp"
)

// Memory provides the agentic memory lifecycle: read, recall, and write.
// It wraps an MCP client that connects to the ogden memory server.
type Memory struct {
	client  *mcp.Client
	enabled bool
}

// New creates a Memory wrapper. If client is nil or doesn't have memory tools,
// memory is disabled and all methods gracefully degrade.
func New(client *mcp.Client) *Memory {
	m := &Memory{client: client}
	if client != nil && client.HasTool("memory_graph") {
		m.enabled = true
	}
	if m.enabled {
		slog.Info("agentic memory enabled")
	} else {
		slog.Info("agentic memory disabled (no memory_graph tool available)")
	}
	return m
}

// Enabled returns whether agentic memory is active.
func (m *Memory) Enabled() bool {
	return m.enabled
}

// ReadMemory fetches the session knowledge graph as text via memory_graph.
// Returns empty string on failure (graceful degradation).
func (m *Memory) ReadMemory(ctx context.Context, sessionID string) string {
	if !m.enabled {
		return ""
	}
	result, _, err := m.client.CallTool(ctx, "memory_graph", map[string]any{
		"session": sessionID,
		"mode":    "lightweight",
	})
	if err != nil {
		slog.Warn("memory_graph call failed", "err", err)
		return ""
	}
	if strings.TrimSpace(result) == "" || strings.TrimSpace(result) == "Graph is empty." {
		slog.Info("memory_graph returned empty", "session", sessionID)
		return ""
	}
	slog.Info("memory_graph returned context", "session", sessionID, "len", len(result))
	// Truncate to 10000 chars to stay within context limits
	if len(result) > 10000 {
		result = result[:10000]
	}
	return result
}

// RecallMemory performs semantic recall for a specific question via memory_recall.
// Returns empty string on failure (graceful degradation).
func (m *Memory) RecallMemory(ctx context.Context, sessionID, question string) string {
	if !m.enabled {
		return ""
	}
	result, _, err := m.client.CallTool(ctx, "memory_recall", map[string]any{
		"session":  sessionID,
		"question": question,
	})
	if err != nil {
		slog.Warn("memory_recall call failed", "err", err)
		return ""
	}
	slog.Info("memory_recall returned context", "session", sessionID, "len", len(result))
	return result
}

// WriteMemory persists a conversation turn via memory_add.
// Runs in the background — errors are logged but not returned.
func (m *Memory) WriteMemory(ctx context.Context, sessionID, question, response string) {
	if !m.enabled {
		return
	}
	go func() {
		// Use a separate context with generous timeout since this runs in background
		// and memory_add may need LLM inference for topic placement
		bgCtx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()

		_, _, err := m.client.CallTool(bgCtx, "memory_add", map[string]any{
			"session":  sessionID,
			"question": question,
			"response": response,
		})
		if err != nil {
			slog.Warn("memory_add call failed", "err", err)
		} else {
			slog.Info("memory_add succeeded", "session", sessionID)
		}
	}()
}