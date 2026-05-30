package tool

import (
	"context"
	"encoding/json"
	"fmt"
)

// DeepSearchFunc is a function that runs a child search-agent session and returns
// the synthesised answer. Implemented by agent.LoopRunner.RunSearchSession and
// wired in from server.go to avoid the tool→agent import cycle.
type DeepSearchFunc func(ctx context.Context, query, dir, model string) (string, error)

// DeepSearchTool lets any agent delegate a research query to the SearchAgent.
// It creates an ephemeral child session, runs the full search loop, and returns
// the synthesised text as the tool result.
type DeepSearchTool struct {
	Run DeepSearchFunc
}

func (DeepSearchTool) ID() string { return "deep_search" }
func (DeepSearchTool) Description() string {
	return "Delegate a research query to the deep search agent. The agent runs parallel web searches, reads top pages, and returns a synthesised answer. Use for any question that requires current web information."
}
func (DeepSearchTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {
				"type": "string",
				"description": "The research question or topic to investigate"
			},
			"context": {
				"type": "string",
				"description": "Optional: extra context to help the search agent focus (e.g. language, framework, constraints)"
			}
		},
		"required": ["query"]
	}`)
}

func (t DeepSearchTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	var input struct {
		Query   string `json:"query"`
		Context string `json:"context"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return Result{}, fmt.Errorf("parse args: %w", err)
	}
	if input.Query == "" {
		return Result{Output: "query is required"}, nil
	}

	fullQuery := input.Query
	if input.Context != "" {
		fullQuery = input.Query + "\n\nAdditional context: " + input.Context
	}

	answer, err := t.Run(ctx, fullQuery, tctx.SessionDir, tctx.Model)
	if err != nil {
		return Result{Output: fmt.Sprintf("Search agent error: %s", err)}, nil
	}

	return Result{
		Title:  input.Query,
		Output: answer,
	}, nil
}
