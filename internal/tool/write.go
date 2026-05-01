package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type WriteTool struct{}

func (WriteTool) ID() string          { return "write" }
func (WriteTool) Description() string { return "Write content to a file, creating it if it doesn't exist" }
func (WriteTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File path to write to"},
			"content": {"type": "string", "description": "Content to write"}
		},
		"required": ["path", "content"]
	}`)
}

func (WriteTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	var input struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return Result{}, fmt.Errorf("parse args: %w", err)
	}

	path := input.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(tctx.SessionDir, path)
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Result{}, fmt.Errorf("create dirs: %w", err)
	}

	if err := os.WriteFile(path, []byte(input.Content), 0o644); err != nil {
		return Result{}, fmt.Errorf("write file: %w", err)
	}

	return Result{
		Title:  filepath.Base(path),
		Output: fmt.Sprintf("Wrote %d bytes to %s", len(input.Content), path),
	}, nil
}