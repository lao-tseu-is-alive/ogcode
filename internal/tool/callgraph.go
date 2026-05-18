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

func (CallGraphTool) Description() string {
	return `Read and query the persisted call graph for the project.

The call graph stores function/method definitions and their call relationships.
Use this to recall what you've previously learned about code structure — which
functions exist, which call which, and how code is connected.

Actions:
- "stats" — Get node and edge counts for a directory
- "nodes" — List all call nodes (optionally filter by package or kind)
- "edges" — List all call edges for a directory
- "callees" — Given a node ID, find what functions it calls
- "callers" — Given a node ID, find what functions call it
- "reachable" — Find all nodes reachable from a node (transitive callers)
- "upsert_node" — Add or update a call node
- "add_edge" — Add a call edge (caller → callee)
- "add_nodes_batch" — Add multiple nodes at once
- "add_edges_batch" — Add multiple edges at once
- "delete_nodes_by_file" — Remove all nodes and edges for a specific file (used after mutations to clear stale data before re-syncing)
- "clear" — Remove all call graph data for a directory`
}

func (t CallGraphTool) Parameters() json.RawMessage {
	return json.RawMessage(`{
		"type": "object",
		"required": ["action", "directory"],
		"properties": {
			"action": {
				"type": "string",
				"description": "One of: stats, nodes, edges, callees, callers, reachable, upsert_node, add_edge, add_nodes_batch, add_edges_batch, delete_nodes_by_file, clear"
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
				"description": "Node kind filter (function, method) for nodes query"
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
					"doc": {"type": "string"}
				}
			},
			"edge": {
				"type": "object",
				"description": "Edge data for add_edge action",
				"properties": {
					"callerId": {"type": "integer"},
					"calleeId": {"type": "integer"},
					"callType": {"type": "string"}
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
						"doc": {"type": "string"}
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
						"callType": {"type": "string"}
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
		lines = append(lines, fmt.Sprintf("[%d] %s.%s (%s:%d) %s%s", n.ID, n.Package, n.Symbol, n.FilePath, n.Line, n.Kind, sig))
	}
	return Result{Title: "CallGraph Nodes", Output: strings.Join(lines, "\n")}, nil
}

func (t CallGraphTool) edges(dir string) (Result, error) {
	nodes, err := t.Store.ListNodesByDirectory(dir, "", "")
	if err != nil {
		return Result{}, err
	}
	if len(nodes) == 0 {
		return Result{Title: "CallGraph Edges", Output: "No edges found (no nodes)."}, nil
	}

	nodeMap := make(map[int64]callgraph.CallNode, len(nodes))
	for _, n := range nodes {
		nodeMap[n.ID] = n
	}

	// Get all edges by iterating nodes and querying callees
	var lines []string
	for _, n := range nodes {
		callees, err := t.Store.CalleesOf(n.ID)
		if err != nil {
			continue
		}
		for _, c := range callees {
			lines = append(lines, fmt.Sprintf("[%d] %s.%s → %s.%s (%s)", c.Edge.ID, n.Package, n.Symbol, c.Callee.Package, c.Callee.Symbol, c.Edge.CallType))
		}
	}
	if len(lines) == 0 {
		return Result{Title: "CallGraph Edges", Output: "No edges found."}, nil
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
		lines = append(lines, fmt.Sprintf("  → [%d] %s.%s (%s:%d) [%s]", r.Callee.ID, r.Callee.Package, r.Callee.Symbol, r.Callee.FilePath, r.Callee.Line, r.Edge.CallType))
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
		lines = append(lines, fmt.Sprintf("  ← [%d] %s.%s (%s:%d) [%s]", r.Caller.ID, r.Caller.Package, r.Caller.Symbol, r.Caller.FilePath, r.Caller.Line, r.Edge.CallType))
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
		lines = append(lines, fmt.Sprintf("  [%d] %s.%s (%s:%d)", n.ID, n.Package, n.Symbol, n.FilePath, n.Line))
	}
	return Result{Title: "CallGraph Reachable", Output: strings.Join(lines, "\n")}, nil
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
	n := callgraph.CallNode{
		Directory: dir,
		Package:   node.Package,
		Symbol:    node.Symbol,
		FilePath:  node.FilePath,
		Line:      node.Line,
		Kind:      callgraph.NodeKind(node.Kind),
		Signature: node.Signature,
		Doc:       node.Doc,
	}
	result, err := t.Store.UpsertNode(n)
	if err != nil {
		return Result{}, fmt.Errorf("upsert call node: %w", err)
	}
	return Result{Title: "CallGraph Node Upserted", Output: fmt.Sprintf("Node [%d] %s.%s", result.ID, result.Package, result.Symbol)}, nil
}

func (t CallGraphTool) addEdge(dir string, edge *struct {
	CallerID int64  `json:"callerId"`
	CalleeID int64  `json:"calleeId"`
	CallType string `json:"callType"`
}) (Result, error) {
	if edge == nil {
		return Result{Title: "CallGraph", Output: "edge data is required for add_edge action."}, nil
	}
	e := callgraph.CallEdge{
		Directory: dir,
		CallerID:  edge.CallerID,
		CalleeID:  edge.CalleeID,
		CallType:  callgraph.CallType(edge.CallType),
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
		n := callgraph.CallNode{
			Directory: dir,
			Package:   node.Package,
			Symbol:    node.Symbol,
			FilePath:  node.FilePath,
			Line:      node.Line,
			Kind:      callgraph.NodeKind(node.Kind),
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
		lines = append(lines, fmt.Sprintf("[%d] %s.%s", r.ID, r.Package, r.Symbol))
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
		cgEdges = append(cgEdges, callgraph.CallEdge{
			Directory: dir,
			CallerID:  e.CallerID,
			CalleeID:  e.CalleeID,
			CallType:  callgraph.CallType(e.CallType),
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