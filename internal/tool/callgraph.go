package tool

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/prasenjeet-symon/ogcode/internal/callgraph"
)

// CallGraphTool lets the agent read and query the call graph store.
// The agent *is* the parser — it reads code, understands call relationships,
// and writes them into the database. This tool provides structured
// read/write access to the persisted graph.
type CallGraphTool struct {
	Store *callgraph.Store
}

func NewCallGraphTool(s *callgraph.Store) CallGraphTool {
	return CallGraphTool{Store: s}
}

func (CallGraphTool) ID() string { return "callgraph" }

const validNodeKindList = "function, method, constructor, init, type, interface, enum, const, variable, module, macro"
const validCallTypeList = "direct, dynamic, interface, callback, async, implements, extends, overrides, instantiates, contains, aliases, imports, uses, reads, writes, decorates, throws"

func (CallGraphTool) Description() string {
	return `Read and query the persisted code knowledge graph for the project.

The graph stores all code symbols and the relationships between them. It is
language-agnostic — the same node kinds and edge types work across Go, Python,
TypeScript, Java, Rust, C/C++, and any other language.

Node kinds:
  Callable:     function, method, constructor, init
  Structural:   type, interface, enum
  Values:       const, variable
  Organization: module, macro

Edge types:
  Calls:        direct, dynamic, interface, callback, async
  Structural:   implements, extends, overrides, instantiates, contains, aliases
  Dependency:   imports, uses, reads, writes, decorates, throws

Use this to recall what you've previously learned about code structure without
re-reading source files. The "search" action is especially powerful: it matches
symbol names, doc text, and signatures — replacing many grep + read cycles.

Actions:
- "stats"              — Node and edge counts for a directory
- "nodes"              — List all nodes (optionally filter by package or kind)
- "edges"              — List all edges for a directory
- "callees"            — Nodes this node has outgoing edges to
- "callers"            — Nodes with edges pointing to this node
- "reachable"          — All nodes transitively reachable via outgoing edges
- "search"             — Case-insensitive substring search across symbol, doc, and signature. Use INSTEAD of grep for code structure exploration
- "upsert_node"        — Add or update a node (MUST include a meaningful "doc" field)
- "add_edge"           — Add a relationship edge between two nodes
- "add_nodes_batch"    — Add multiple nodes at once
- "add_edges_batch"    — Add multiple edges at once
- "delete_nodes_by_file" — Remove all nodes and edges for a file (call before re-syncing after a mutation)
- "clear"              — Remove all graph data for a directory

IMPORTANT: The "doc" field on nodes
Every node MUST have a meaningful "doc" field. Tailor it to the node's kind:
- function/method/constructor: what it does, who calls it, what it calls, why it exists
- type/interface/enum: what concept it models, what fields/methods/variants it defines, where it is used in the system
- const/variable: what value it holds, why it exists, where it is referenced
- module: what this package/module is responsible for, its main exports and key dependencies
- macro: what code it generates, when to invoke it

Examples:
  method: "Executes the main agent loop — streaming, tool dispatch, memory writes. Called by Run; calls buildSystemPrompt and processToolCalls. Separates orchestration from per-request setup."
  type:   "Core server struct owning the HTTP router, DB, and config. Instantiated once at startup; all HTTP handlers are methods on it. Owns the lifetime of all subsystem stores."
  module: "Agent orchestration package — owns the run loop, tool dispatch, and prompt assembly. Depends on the tool and callgraph packages; called from the server on session start."

Do NOT leave the doc field empty or set it to something vague like "handles request".`
}

func (t CallGraphTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
				"type": "object",
				"required": ["action", "directory"],
				"properties": {
					"action": {
						"type": "string",
						"description": "One of: stats, nodes, edges, callees, callers, reachable, search, upsert_node, add_edge, add_nodes_batch, add_edges_batch, delete_nodes_by_file, clear"
					},
					"directory": {
						"type": "string",
						"description": "The project directory (usually the working directory)"
					},
					"file_path": {
						"type": "string",
						"description": "File path (relative to directory) for delete_nodes_by_file — removes all nodes and edges belonging to this file"
					},
					"node_id": {
						"type": "integer",
						"description": "Node ID for callees/callers/reachable queries"
					},
					"package": {
						"type": "string",
						"description": "Package filter for nodes query"
					},
					"kind": {
						"type": "string",
						"description": "Node kind filter for nodes query. Callable: function, method, constructor, init. Structural: type, interface, enum. Values: const, variable. Organization: module, macro."
					},
					"query": {
						"type": "string",
						"description": "Search query for the search action. Matches against symbol names, doc text, and function signatures (case-insensitive substring). Use this to discover functions by name or concept without knowing exact names."
					},
					"limit": {
						"type": "integer",
						"description": "Maximum number of results for the search action (default 50)"
					},
					"max_depth": {
						"type": "integer",
						"description": "Max depth for reachable query (default 20)"
					},
					"node": {
						"type": "object",
						"description": "Node data for upsert_node action",
						"properties": {
							"package": {"type": "string"},
							"symbol": {"type": "string"},
							"filePath": {"type": "string"},
							"line": {"type": "integer"},
							"kind": {"type": "string"},
							"signature": {"type": "string"},
							"doc": {"type": "string", "description": "REQUIRED: What this function does, how it relates to other nodes, and why it exists. Include: (1) what it does — concise behavior summary, (2) how it relates — callers/callees and data flow, (3) why it exists — architectural purpose. Do NOT leave empty."}
						}
					},
					"edge": {
						"type": "object",
						"description": "Edge data for add_edge action",
						"properties": {
							"callerId": {"type": "integer"},
							"calleeId": {"type": "integer"},
							"callType": {"type": "string", "description": "Relationship type. Calls: direct, dynamic, interface, callback, async. Structural: implements, extends, overrides, instantiates, contains, aliases. Dependency: imports, uses, reads, writes, decorates, throws."}
						}
					},
					"nodes": {
						"type": "array",
						"description": "Array of nodes for add_nodes_batch",
						"items": {
							"type": "object",
							"properties": {
								"package": {"type": "string"},
								"symbol": {"type": "string"},
								"filePath": {"type": "string"},
								"line": {"type": "integer"},
								"kind": {"type": "string"},
								"signature": {"type": "string"},
								"doc": {"type": "string", "description": "REQUIRED: What this function does, how it relates to other nodes, and why it exists. Do NOT leave empty."}
							}
						}
					},
					"edges": {
						"type": "array",
						"description": "Array of edges for add_edges_batch",
						"items": {
							"type": "object",
							"properties": {
								"callerId": {"type": "integer"},
								"calleeId": {"type": "integer"},
								"callType": {"type": "string", "description": "Relationship type. Calls: direct, dynamic, interface, callback, async. Structural: implements, extends, overrides, instantiates, contains, aliases. Dependency: imports, uses, reads, writes, decorates, throws."}
							}
						}
					}
				}
			}`)
}

func (t CallGraphTool) Execute(ctx context.Context, args json.RawMessage, tctx Context) (Result, error) {
	if t.Store == nil {
		return Result{Title: "CallGraph", Output: "Call graph store is not available."}, nil
	}

	var params struct {
		Action    string `json:"action"`
		Directory string `json:"directory"`
		FilePath  string `json:"file_path"`
		NodeID    int64  `json:"node_id"`
		Package   string `json:"package"`
		Kind      string `json:"kind"`
		Query     string `json:"query"`
		Limit     int    `json:"limit"`
		MaxDepth  int    `json:"max_depth"`
		Node      *struct {
			Package   string `json:"package"`
			Symbol    string `json:"symbol"`
			FilePath  string `json:"filePath"`
			Line      int    `json:"line"`
			Kind      string `json:"kind"`
			Signature string `json:"signature"`
			Doc       string `json:"doc"`
		} `json:"node"`
		Edge *struct {
			CallerID int64  `json:"callerId"`
			CalleeID int64  `json:"calleeId"`
			CallType string `json:"callType"`
		} `json:"edge"`
		Nodes []struct {
			Package   string `json:"package"`
			Symbol    string `json:"symbol"`
			FilePath  string `json:"filePath"`
			Line      int    `json:"line"`
			Kind      string `json:"kind"`
			Signature string `json:"signature"`
			Doc       string `json:"doc"`
		} `json:"nodes"`
		Edges []struct {
			CallerID int64  `json:"callerId"`
			CalleeID int64  `json:"calleeId"`
			CallType string `json:"callType"`
		} `json:"edges"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return Result{}, fmt.Errorf("parse callgraph args: %w", err)
	}

	dir := params.Directory
	if dir == "" {
		dir = tctx.SessionDir
	}

	slog.Info("callgraph tool invoked", "action", params.Action, "directory", dir)

	switch params.Action {
	case "stats":
		return t.stats(dir)
	case "nodes":
		return t.nodes(dir, params.Package, params.Kind)
	case "edges":
		return t.edges(dir)
	case "callees":
		return t.callees(params.NodeID)
	case "callers":
		return t.callers(params.NodeID)
	case "reachable":
		return t.reachable(params.NodeID, params.MaxDepth)
	case "search":
		return t.search(dir, params.Query, params.Limit)
	case "upsert_node":
		return t.upsertNode(dir, params.Node)
	case "add_edge":
		return t.addEdge(dir, params.Edge)
	case "add_nodes_batch":
		return t.addNodesBatch(dir, params.Nodes)
	case "add_edges_batch":
		return t.addEdgesBatch(dir, params.Edges)
	case "delete_nodes_by_file":
		return t.deleteNodesByFile(dir, params.FilePath)
	case "clear":
		return t.clear(dir)
	default:
		return Result{Title: "CallGraph", Output: fmt.Sprintf("Unknown action: %s", params.Action)}, nil
	}
}

func formatDoc(doc string) string {
	if doc == "" {
		return ""
	}
	return "\n    " + strings.ReplaceAll(doc, "\n", "\n    ")
}

func (t CallGraphTool) stats(dir string) (Result, error) {
	nodes, edges, err := t.Store.Stats(dir)
	if err != nil {
		return Result{}, err
	}
	out := fmt.Sprintf("Call graph stats for %s:\n- Nodes: %d\n- Edges: %d", dir, nodes, edges)
	return Result{Title: "CallGraph Stats", Output: out}, nil
}

func (t CallGraphTool) nodes(dir, pkg, kind string) (Result, error) {
	nodes, err := t.Store.ListNodesByDirectory(dir, pkg, callgraph.NodeKind(kind))
	if err != nil {
		return Result{}, err
	}
	if len(nodes) == 0 {
		return Result{Title: "CallGraph Nodes", Output: "No nodes found."}, nil
	}
	var lines []string
	for _, n := range nodes {
		sig := n.Signature
		if sig != "" {
			sig = " " + sig
		}
		line := fmt.Sprintf("[%d] %s.%s (%s:%d) %s%s", n.ID, n.Package, n.Symbol, n.FilePath, n.Line, n.Kind, sig)
		if n.Doc != "" {
			line += formatDoc(n.Doc)
		}
		lines = append(lines, line)
	}
	return Result{Title: "CallGraph Nodes", Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) edges(dir string) (Result, error) {
	edges, err := t.Store.ListEdgesByDirectory(dir)
	if err != nil {
		return Result{}, err
	}
	if len(edges) == 0 {
		return Result{Title: "CallGraph Edges", Output: "No edges found."}, nil
	}

	nodes, err := t.Store.ListNodesByDirectory(dir, "", "")
	if err != nil {
		return Result{}, err
	}
	nodeMap := make(map[int64]callgraph.CallNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	var lines []string
	for _, e := range edges {
		caller, callerOK := nodeMap[e.CallerID]
		callee, calleeOK := nodeMap[e.CalleeID]
		if !callerOK || !calleeOK {
			lines = append(lines, fmt.Sprintf("[%d] WARN: dangling edge (caller=%d found=%v, callee=%d found=%v)", e.ID, e.CallerID, callerOK, e.CalleeID, calleeOK))
			continue
		}
		lines = append(lines, fmt.Sprintf("[%d] %s.%s → %s.%s (%s)", e.ID, caller.Package, caller.Symbol, callee.Package, callee.Symbol, e.CallType))
	}
	return Result{Title: "CallGraph Edges", Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) callees(nodeID int64) (Result, error) {
	if nodeID == 0 {
		return Result{Title: "CallGraph", Output: "node_id is required for callees query."}, nil
	}
	results, err := t.Store.CalleesOf(nodeID)
	if err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{Title: "CallGraph Callees", Output: "No callees found."}, nil
	}
	// Get the caller info for context
	caller, err := t.Store.GetNode(nodeID)
	if err != nil || caller == nil {
		return Result{Title: "CallGraph Callees", Output: "Node not found."}, nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("%s.%s calls:", caller.Package, caller.Symbol))
	for _, r := range results {
		line := fmt.Sprintf("  → [%d] %s.%s (%s:%d) [%s]", r.Callee.ID, r.Callee.Package, r.Callee.Symbol, r.Callee.FilePath, r.Callee.Line, r.Edge.CallType)
		if r.Callee.Doc != "" {
			line += formatDoc(r.Callee.Doc)
		}
		lines = append(lines, line)
	}
	return Result{Title: fmt.Sprintf("Callees of %s.%s", caller.Package, caller.Symbol), Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) callers(nodeID int64) (Result, error) {
	if nodeID == 0 {
		return Result{Title: "CallGraph", Output: "node_id is required for callers query."}, nil
	}
	results, err := t.Store.CallersOf(nodeID)
	if err != nil {
		return Result{}, err
	}
	if len(results) == 0 {
		return Result{Title: "CallGraph Callers", Output: "No callers found."}, nil
	}
	// Get the callee info for context
	callee, err := t.Store.GetNode(nodeID)
	if err != nil || callee == nil {
		return Result{Title: "CallGraph Callers", Output: "Node not found."}, nil
	}
	var lines []string
	lines = append(lines, fmt.Sprintf("%s.%s is called by:", callee.Package, callee.Symbol))
	for _, r := range results {
		line := fmt.Sprintf("  ← [%d] %s.%s (%s:%d) [%s]", r.Caller.ID, r.Caller.Package, r.Caller.Symbol, r.Caller.FilePath, r.Caller.Line, r.Edge.CallType)
		if r.Caller.Doc != "" {
			line += formatDoc(r.Caller.Doc)
		}
		lines = append(lines, line)
	}
	return Result{Title: fmt.Sprintf("Callers of %s.%s", callee.Package, callee.Symbol), Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) reachable(nodeID int64, maxDepth int) (Result, error) {
	if nodeID == 0 {
		return Result{Title: "CallGraph", Output: "node_id is required for reachable query."}, nil
	}
	if maxDepth <= 0 {
		maxDepth = 20
	}
	nodes, err := t.Store.ReachableFrom(nodeID, maxDepth)
	if err != nil {
		return Result{}, err
	}
	if len(nodes) == 0 {
		return Result{Title: "CallGraph Reachable", Output: "No reachable nodes found."}, nil
	}
	var lines []string
	for _, n := range nodes {
		line := fmt.Sprintf("  [%d] %s.%s (%s:%d)", n.ID, n.Package, n.Symbol, n.FilePath, n.Line)
		if n.Doc != "" {
			line += formatDoc(n.Doc)
		}
		lines = append(lines, line)
	}
	return Result{Title: "CallGraph Reachable", Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) search(dir, query string, limit int) (Result, error) {
	if query == "" {
		return Result{Title: "CallGraph Search", Output: "query is required for search action."}, nil
	}
	nodes, err := t.Store.SearchNodes(dir, query, limit)
	if err != nil {
		return Result{}, err
	}
	if len(nodes) == 0 {
		return Result{Title: "CallGraph Search", Output: fmt.Sprintf("No results found for %q.", query)}, nil
	}
	var lines []string
	for _, n := range nodes {
		sig := n.Signature
		if sig != "" {
			sig = " " + sig
		}
		line := fmt.Sprintf("[%d] %s.%s (%s:%d) %s%s", n.ID, n.Package, n.Symbol, n.FilePath, n.Line, n.Kind, sig)
		if n.Doc != "" {
			line += formatDoc(n.Doc)
		}
		lines = append(lines, line)
	}
	return Result{Title: fmt.Sprintf("CallGraph Search: %q (%d results)", query, len(nodes)), Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) upsertNode(dir string, node *struct {
	Package   string `json:"package"`
	Symbol    string `json:"symbol"`
	FilePath  string `json:"filePath"`
	Line      int    `json:"line"`
	Kind      string `json:"kind"`
	Signature string `json:"signature"`
	Doc       string `json:"doc"`
}) (Result, error) {
	if node == nil {
		return Result{Title: "CallGraph", Output: "node data is required for upsert_node action."}, nil
	}
	kind := callgraph.NodeKind(node.Kind)
	if !callgraph.ValidNodeKind(kind) {
		return Result{Title: "CallGraph", Output: fmt.Sprintf("invalid node kind %q; valid: "+validNodeKindList, node.Kind)}, nil
	}
	n := callgraph.CallNode{
		Directory: dir,
		Package:   node.Package,
		Symbol:    node.Symbol,
		FilePath:  node.FilePath,
		Line:      node.Line,
		Kind:      kind,
		Signature: node.Signature,
		Doc:       node.Doc,
	}
	result, err := t.Store.UpsertNode(n)
	if err != nil {
		return Result{}, fmt.Errorf("upsert call node: %w", err)
	}
	var output string
	if result.Doc != "" {
		output = fmt.Sprintf("Node [%d] %s.%s\n  doc: %s", result.ID, result.Package, result.Symbol, result.Doc)
	} else {
		output = fmt.Sprintf("Node [%d] %s.%s (no doc)", result.ID, result.Package, result.Symbol)
	}
	return Result{Title: "CallGraph Node Upserted", Output: output}, nil
}

func (t CallGraphTool) addEdge(dir string, edge *struct {
	CallerID int64  `json:"callerId"`
	CalleeID int64  `json:"calleeId"`
	CallType string `json:"callType"`
}) (Result, error) {
	if edge == nil {
		return Result{Title: "CallGraph", Output: "edge data is required for add_edge action."}, nil
	}
	ct := callgraph.CallType(edge.CallType)
	if !callgraph.ValidCallType(ct) {
		return Result{Title: "CallGraph", Output: fmt.Sprintf("invalid call type %q; valid: "+validCallTypeList, edge.CallType)}, nil
	}
	e := callgraph.CallEdge{
		Directory: dir,
		CallerID:  edge.CallerID,
		CalleeID:  edge.CalleeID,
		CallType:  ct,
	}
	result, err := t.Store.AddEdge(e)
	if err != nil {
		return Result{}, fmt.Errorf("add call edge: %w", err)
	}
	return Result{Title: "CallGraph Edge Added", Output: fmt.Sprintf("Edge [%d] %d → %d (%s)", result.ID, result.CallerID, result.CalleeID, result.CallType)}, nil
}

func (t CallGraphTool) addNodesBatch(dir string, nodes []struct {
	Package   string `json:"package"`
	Symbol    string `json:"symbol"`
	FilePath  string `json:"filePath"`
	Line      int    `json:"line"`
	Kind      string `json:"kind"`
	Signature string `json:"signature"`
	Doc       string `json:"doc"`
}) (Result, error) {
	if len(nodes) == 0 {
		return Result{Title: "CallGraph", Output: "No nodes provided."}, nil
	}
	var results []callgraph.CallNode
	for _, node := range nodes {
		kind := callgraph.NodeKind(node.Kind)
		if !callgraph.ValidNodeKind(kind) {
			return Result{Title: "CallGraph", Output: fmt.Sprintf("invalid node kind %q for %s.%s; valid: "+validNodeKindList, node.Kind, node.Package, node.Symbol)}, nil
		}
		n := callgraph.CallNode{
			Directory: dir,
			Package:   node.Package,
			Symbol:    node.Symbol,
			FilePath:  node.FilePath,
			Line:      node.Line,
			Kind:      kind,
			Signature: node.Signature,
			Doc:       node.Doc,
		}
		result, err := t.Store.UpsertNode(n)
		if err != nil {
			return Result{}, fmt.Errorf("upsert call node %s.%s: %w", n.Package, n.Symbol, err)
		}
		results = append(results, *result)
	}
	var lines []string
	for _, r := range results {
		if r.Doc != "" {
			lines = append(lines, fmt.Sprintf("[%d] %s.%s\n    doc: %s", r.ID, r.Package, r.Symbol, r.Doc))
		} else {
			lines = append(lines, fmt.Sprintf("[%d] %s.%s (no doc)", r.ID, r.Package, r.Symbol))
		}
	}
	return Result{Title: fmt.Sprintf("CallGraph: %d Nodes Upserted", len(results)), Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) addEdgesBatch(dir string, edges []struct {
	CallerID int64  `json:"callerId"`
	CalleeID int64  `json:"calleeId"`
	CallType string `json:"callType"`
}) (Result, error) {
	if len(edges) == 0 {
		return Result{Title: "CallGraph", Output: "No edges provided."}, nil
	}
	var cgEdges []callgraph.CallEdge
	for _, e := range edges {
		ct := callgraph.CallType(e.CallType)
		if !callgraph.ValidCallType(ct) {
			return Result{Title: "CallGraph", Output: fmt.Sprintf("invalid call type %q for edge %d→%d; valid: "+validCallTypeList, e.CallType, e.CallerID, e.CalleeID)}, nil
		}
		cgEdges = append(cgEdges, callgraph.CallEdge{
			Directory: dir,
			CallerID:  e.CallerID,
			CalleeID:  e.CalleeID,
			CallType:  ct,
		})
	}
	if err := t.Store.AddEdgesBatch(cgEdges); err != nil {
		return Result{}, fmt.Errorf("add edges batch: %w", err)
	}
	return Result{Title: fmt.Sprintf("CallGraph: %d Edges Added", len(cgEdges)), Output: fmt.Sprintf("Added %d edges.", len(cgEdges))}, nil
}

func (t CallGraphTool) clear(dir string) (Result, error) {
	if err := t.Store.DeleteNodesByDirectory(dir); err != nil {
		return Result{}, fmt.Errorf("clear call graph: %w", err)
	}
	return Result{Title: "CallGraph Cleared", Output: fmt.Sprintf("Cleared all call graph data for %s", dir)}, nil
}

func (t CallGraphTool) deleteNodesByFile(dir, filePath string) (Result, error) {
	if filePath == "" {
		return Result{Title: "CallGraph", Output: "file_path is required for delete_nodes_by_file action."}, nil
	}
	deleted, err := t.Store.DeleteNodesByFile(dir, filePath)
	if err != nil {
		return Result{}, fmt.Errorf("delete nodes by file: %w", err)
	}
	return Result{Title: "CallGraph Nodes Deleted by File", Output: fmt.Sprintf("Deleted %d nodes (and their edges) for %s in %s", deleted, filePath, dir)}, nil
}