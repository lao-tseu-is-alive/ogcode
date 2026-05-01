package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type GlobTool struct{}

func (GlobTool) ID() string          { return "glob" }
func (GlobTool) Description() string { return "Find files matching a glob pattern" }
func (GlobTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"pattern": {"type": "string", "description": "Glob pattern to match (e.g. **/*.go)"}
		},
		"required": ["pattern"]
	}`)
}

func (GlobTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	var input struct {
		Pattern string `json:"pattern"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return Result{}, fmt.Errorf("parse args: %w", err)
	}

	pattern := input.Pattern
	var matches []string

	if strings.Contains(pattern, "**") {
		// Walk the directory tree for ** patterns
		// Split pattern: "**/*.go" -> walk all, match suffix ".go"
		suffix := strings.TrimPrefix(pattern, "**/")
		filepath.Walk(tctx.SessionDir, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(tctx.SessionDir, path)
			// Skip hidden dirs
			if strings.Contains(rel, "/.") {
				return nil
			}
			matched, _ := filepath.Match(suffix, filepath.Base(path))
			if matched {
				matches = append(matches, rel)
			}
			return nil
		})
	} else {
		// Simple pattern: use filepath.Glob
		absMatches, err := filepath.Glob(filepath.Join(tctx.SessionDir, pattern))
		if err != nil {
			return Result{}, fmt.Errorf("glob: %w", err)
		}
		for _, m := range absMatches {
			rel, _ := filepath.Rel(tctx.SessionDir, m)
			matches = append(matches, rel)
		}
	}

	if len(matches) == 0 {
		return Result{
			Title:  input.Pattern,
			Output: "No files found",
		}, nil
	}

	return Result{
		Title:  fmt.Sprintf("%s (%d files)", input.Pattern, len(matches)),
		Output: strings.Join(matches, "\n"),
	}, nil
}

