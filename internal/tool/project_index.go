package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/docindex"
)

// ProjectIndexTool returns a labeled JSON tree of all indexed text/code files
// in the session directory, so the agent can navigate the project by topic
// without knowing file paths upfront.
type ProjectIndexTool struct {
	Store *docindex.Store
}

func NewProjectIndexTool(store *docindex.Store) ProjectIndexTool {
	return ProjectIndexTool{Store: store}
}

func (ProjectIndexTool) ID() string { return "codebase_map" }

func (ProjectIndexTool) Description() string {
	return "Return a labeled JSON tree of indexed text and code files. Each file appears as a leaf with its topic labels. Use this to discover which files are relevant to a topic before reading them. Pass subdir to scope the results to a specific folder (e.g. \"internal/auth\") — recommended for large projects."
}

func (ProjectIndexTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"properties": {
			"subdir": {
				"type": "string",
				"description": "Optional subdirectory path relative to the project root (e.g. \"internal/auth\"). Scopes results to that folder and its children. Omit to return the full project index."
			}
		}
	}`)
}

func (t ProjectIndexTool) Execute(_ context.Context, args json.RawMessage, tctx Context) (Result, error) {
	var params struct {
		Subdir string `json:"subdir"`
	}
	if args != nil {
		_ = json.Unmarshal(args, &params)
	}

	prefix := tctx.SessionDir
	title := "Project Index"
	if params.Subdir != "" {
		prefix = filepath.Join(tctx.SessionDir, filepath.FromSlash(params.Subdir))
		title = fmt.Sprintf("Project Index / %s", params.Subdir)
	}

	entries, err := t.Store.ListTextFiles(prefix)
	if err != nil {
		return Result{}, fmt.Errorf("list text files: %w", err)
	}
	if len(entries) == 0 {
		msg := "no indexed files found — run Index Docs first to build the project index"
		if params.Subdir != "" {
			msg = fmt.Sprintf("no indexed files found under %q", params.Subdir)
		}
		return Result{Title: title, Output: msg}, nil
	}

	tree := buildProjectTree(tctx.SessionDir, entries)
	out, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return Result{}, fmt.Errorf("marshal project tree: %w", err)
	}

	return Result{
		Title:  fmt.Sprintf("%s (%d files)", title, len(entries)),
		Output: string(out),
	}, nil
}

// buildProjectTree converts a flat list of indexed entries into a nested
// folder/file tree. Each file leaf is the array of labels for that file.
func buildProjectTree(baseDir string, entries []*docindex.PageEntry) map[string]any {
	root := make(map[string]any)
	for _, e := range entries {
		rel, err := filepath.Rel(baseDir, e.DocPath)
		if err != nil {
			rel = e.DocPath
		}
		parts := strings.Split(filepath.ToSlash(rel), "/")

		node := root
		for _, part := range parts[:len(parts)-1] {
			child, ok := node[part]
			if !ok {
				child = make(map[string]any)
				node[part] = child
			}
			node = child.(map[string]any)
		}

		labels := e.Labels
		if labels == nil {
			labels = []string{}
		}
		node[parts[len(parts)-1]] = labels
	}
	return root
}
