package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type ReadTool struct{}

func (ReadTool) ID() string          { return "read" }
func (ReadTool) Description() string { return "Read file contents or list directory contents" }
func (ReadTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"path": {"type": "string", "description": "File or directory path"},
			"offset": {"type": "number", "description": "Line offset to start reading from"},
			"limit": {"type": "number", "description": "Maximum number of lines to read"}
		},
		"required": ["path"]
	}`)
}

func (ReadTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	var input struct {
		Path   string `json:"path"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return Result{}, fmt.Errorf("parse args: %w", err)
	}

	path := input.Path
	if !filepath.IsAbs(path) {
		path = filepath.Join(tctx.SessionDir, path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return Result{}, fmt.Errorf("stat %s: %w", path, err)
	}

	if info.IsDir() {
		entries, err := os.ReadDir(path)
		if err != nil {
			return Result{}, fmt.Errorf("read dir: %w", err)
		}
		var lines []string
		for _, e := range entries {
			prefix := "  "
			if e.IsDir() {
				prefix = "D "
			}
			lines = append(lines, prefix+e.Name())
		}
		return Result{
			Title:  filepath.Base(path) + "/",
			Output: strings.Join(lines, "\n"),
		}, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Result{}, fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	start := input.Offset
	if start > len(lines) {
		start = len(lines)
	}
	end := len(lines)
	if input.Limit > 0 && start+input.Limit < end {
		end = start + input.Limit
	}

	// Add line numbers
	var numbered []string
	for i := start; i < end; i++ {
		numbered = append(numbered, fmt.Sprintf("%6d\t%s", i+1, lines[i]))
	}

	return Result{
		Title:  filepath.Base(path),
		Output: strings.Join(numbered, "\n"),
	}, nil
}