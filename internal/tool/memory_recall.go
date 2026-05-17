package tool

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/prasenjeet-symon/ogcode/internal/memory"
)

// MemoryRecallTool lets the LLM query the agentic knowledge graph on demand.
type MemoryRecallTool struct {
	Memory *memory.Memory
}

func NewMemoryRecallTool(mem *memory.Memory) MemoryRecallTool {
	return MemoryRecallTool{Memory: mem}
}

func (t MemoryRecallTool) ID() string        { return "memory_recall" }
func (t MemoryRecallTool) Description() string { return "Search the agentic memory graph for past facts, context, and prior reasoning relevant to a specific question. Use this when you need precise historical details (e.g., exact config values, file paths, decisions made earlier) that may be summarized too coarsely in <prior_context>." }

func (t MemoryRecallTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["question"],
		"properties": {
			"question": {
				"type": "string",
				"description": "A clear, specific question to look up in the memory graph."
			}
		}
	}`)
}

func (t MemoryRecallTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	if t.Memory == nil || !t.Memory.Enabled() {
		return Result{Title: "Memory Recall", Output: "Agentic memory is not enabled."}, nil
	}

	var params struct {
		Question string `json:"question"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return Result{}, err
	}
	if params.Question == "" {
		return Result{Title: "Memory Recall", Output: "No question provided."}, nil
	}

	slog.Info("memory_recall tool invoked", "question", params.Question, "session", tctx.SessionID)

	recall := t.Memory.RecallMemory(ctx, string(tctx.SessionID), params.Question)
	if recall == "" {
		return Result{Title: "Memory Recall", Output: "No relevant past context found in memory."}, nil
	}

	return Result{Title: "Memory Recall", Output: recall}, nil
}
