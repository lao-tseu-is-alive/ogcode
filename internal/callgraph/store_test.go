package callgraph

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/prasenjeet-symon/ogcode/internal/db"
)

func openTestDB(t *testing.T) *db.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	database, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() {
		database.Close()
		os.RemoveAll(dir)
	})
	return database
}

func TestUpsertAndGetNode(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	n := CallNode{
		Directory: "/project",
		Package:   "internal/server",
		Symbol:    "Server.Start",
		FilePath:  "internal/server/server.go",
		Line:      77,
		Kind:      KindMethod,
		Signature: "func (s *Server) Start() error",
		Doc:       "Start the server",
	}

	result, err := s.UpsertNode(n)
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	if result.ID == 0 {
		t.Error("expected non-zero ID after upsert")
	}
	if result.Package != n.Package {
		t.Errorf("package: got %q, want %q", result.Package, n.Package)
	}
	if result.Symbol != n.Symbol {
		t.Errorf("symbol: got %q, want %q", result.Symbol, n.Symbol)
	}
	if result.Kind != KindMethod {
		t.Errorf("kind: got %q, want %q", result.Kind, KindMethod)
	}

	// Fetch by ID
	got, err := s.GetNode(result.ID)
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got == nil {
		t.Fatal("GetNode returned nil")
	}
	if got.Symbol != n.Symbol {
		t.Errorf("GetNode symbol: got %q, want %q", got.Symbol, n.Symbol)
	}

	// Fetch by symbol
	got2, err := s.GetNodeBySymbol("/project", "internal/server", "Server.Start")
	if err != nil {
		t.Fatalf("GetNodeBySymbol: %v", err)
	}
	if got2 == nil {
		t.Fatal("GetNodeBySymbol returned nil")
	}
	if got2.ID != result.ID {
		t.Errorf("GetNodeBySymbol ID: got %d, want %d", got2.ID, result.ID)
	}

	// Upsert same node again (should update)
	n2 := CallNode{
		Directory: "/project",
		Package:   "internal/server",
		Symbol:    "Server.Start",
		FilePath:  "internal/server/server.go",
		Line:      79, // updated line
		Kind:      KindMethod,
	}
	result2, err := s.UpsertNode(n2)
	if err != nil {
		t.Fatalf("UpsertNode (update): %v", err)
	}
	if result2.ID != result.ID {
		t.Errorf("upsert should preserve ID: got %d, want %d", result2.ID, result.ID)
	}

	// Verify line was updated
	got3, err := s.GetNode(result.ID)
	if err != nil {
		t.Fatalf("GetNode after upsert: %v", err)
	}
	if got3.Line != 79 {
		t.Errorf("line after upsert: got %d, want 79", got3.Line)
	}
}

func TestListNodesByDirectory(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	nodes := []CallNode{
		{Directory: "/project", Package: "pkg/a", Symbol: "Foo", Kind: KindFunction},
		{Directory: "/project", Package: "pkg/a", Symbol: "Bar", Kind: KindFunction},
		{Directory: "/project", Package: "pkg/b", Symbol: "Baz", Kind: KindMethod},
		{Directory: "/other", Package: "pkg/c", Symbol: "Qux", Kind: KindFunction},
	}
	for _, n := range nodes {
		_, err := s.UpsertNode(n)
		if err != nil {
			t.Fatalf("UpsertNode: %v", err)
		}
	}

	// List all for /project
	all, err := s.ListNodesByDirectory("/project", "", "")
	if err != nil {
		t.Fatalf("ListNodesByDirectory: %v", err)
	}
	if len(all) != 3 {
		t.Errorf("expected 3 nodes, got %d", len(all))
	}

	// Filter by package
	apkg, err := s.ListNodesByDirectory("/project", "pkg/a", "")
	if err != nil {
		t.Fatalf("ListNodesByDirectory (pkg): %v", err)
	}
	if len(apkg) != 2 {
		t.Errorf("expected 2 nodes in pkg/a, got %d", len(apkg))
	}

	// Filter by kind
	methods, err := s.ListNodesByDirectory("/project", "", KindMethod)
	if err != nil {
		t.Fatalf("ListNodesByDirectory (kind): %v", err)
	}
	if len(methods) != 1 {
		t.Errorf("expected 1 method, got %d", len(methods))
	}
}

func TestAddEdgeAndQueries(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	// Create two nodes
	caller, err := s.UpsertNode(CallNode{
		Directory: "/project", Package: "main", Symbol: "main",
		FilePath: "main.go", Line: 10, Kind: KindFunction,
	})
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}
	callee, err := s.UpsertNode(CallNode{
		Directory: "/project", Package: "fmt", Symbol: "Println",
		FilePath: "print.go", Line: 100, Kind: KindFunction,
	})
	if err != nil {
		t.Fatalf("UpsertNode: %v", err)
	}

	// Add edge
	edge, err := s.AddEdge(CallEdge{
		Directory: "/project",
		CallerID:  caller.ID,
		CalleeID:  callee.ID,
		CallType:  CallDirect,
	})
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if edge.ID == 0 {
		t.Error("expected non-zero edge ID")
	}

	// CalleesOf
	callees, err := s.CalleesOf(caller.ID)
	if err != nil {
		t.Fatalf("CalleesOf: %v", err)
	}
	if len(callees) != 1 {
		t.Fatalf("expected 1 callee, got %d", len(callees))
	}
	if callees[0].Callee.Symbol != "Println" {
		t.Errorf("callee symbol: got %q, want 'Println'", callees[0].Callee.Symbol)
	}
	if callees[0].Edge.CallType != CallDirect {
		t.Errorf("call type: got %q, want 'direct'", callees[0].Edge.CallType)
	}

	// CallersOf
	callers, err := s.CallersOf(callee.ID)
	if err != nil {
		t.Fatalf("CallersOf: %v", err)
	}
	if len(callers) != 1 {
		t.Fatalf("expected 1 caller, got %d", len(callers))
	}
	if callers[0].Caller.Symbol != "main" {
		t.Errorf("caller symbol: got %q, want 'main'", callers[0].Caller.Symbol)
	}

	// Stats
	nodes, edges, err := s.Stats("/project")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if nodes != 2 {
		t.Errorf("expected 2 nodes, got %d", nodes)
	}
	if edges != 1 {
		t.Errorf("expected 1 edge, got %d", edges)
	}
}

func TestAddEdgesBatch(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	// Create 3 nodes
	n1, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "A", Kind: KindFunction})
	n2, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "B", Kind: KindFunction})
	n3, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "C", Kind: KindFunction})

	edges := []CallEdge{
		{Directory: "/project", CallerID: n1.ID, CalleeID: n2.ID, CallType: CallDirect},
		{Directory: "/project", CallerID: n1.ID, CalleeID: n3.ID, CallType: CallDirect},
		{Directory: "/project", CallerID: n2.ID, CalleeID: n3.ID, CallType: CallInterface},
	}

	err := s.AddEdgesBatch(edges)
	if err != nil {
		t.Fatalf("AddEdgesBatch: %v", err)
	}

	_, edgeCount, err := s.Stats("/project")
	if err != nil {
		t.Fatalf("Stats: %v", err)
	}
	if edgeCount != 3 {
		t.Errorf("expected 3 edges, got %d", edgeCount)
	}
}

func TestReachableFrom(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	// Create chain: A → B → C → D
	a, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "A", Kind: KindFunction})
	b, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "B", Kind: KindFunction})
	c, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "C", Kind: KindFunction})
	d, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "D", Kind: KindFunction})

	s.AddEdge(CallEdge{Directory: "/project", CallerID: a.ID, CalleeID: b.ID, CallType: CallDirect})
	s.AddEdge(CallEdge{Directory: "/project", CallerID: b.ID, CalleeID: c.ID, CallType: CallDirect})
	s.AddEdge(CallEdge{Directory: "/project", CallerID: c.ID, CalleeID: d.ID, CallType: CallDirect})

	reachable, err := s.ReachableFrom(a.ID, 0)
	if err != nil {
		t.Fatalf("ReachableFrom: %v", err)
	}
	if len(reachable) != 3 {
		t.Errorf("expected 3 reachable nodes from A, got %d", len(reachable))
	}

	// Test depth limit
	reachable2, err := s.ReachableFrom(a.ID, 1)
	if err != nil {
		t.Fatalf("ReachableFrom (depth 1): %v", err)
	}
	if len(reachable2) != 1 {
		t.Errorf("expected 1 reachable node at depth 1, got %d", len(reachable2))
	}
}

func TestDeleteNode(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	a, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "A", Kind: KindFunction})
	b, _ := s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "B", Kind: KindFunction})
	s.AddEdge(CallEdge{Directory: "/project", CallerID: a.ID, CalleeID: b.ID, CallType: CallDirect})

	err := s.DeleteNode(a.ID)
	if err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}

	got, err := s.GetNode(a.ID)
	if err != nil {
		t.Fatalf("GetNode after delete: %v", err)
	}
	if got != nil {
		t.Error("expected nil after deleting node")
	}

	// Verify edge was cascade-deleted
	_, edges, _ := s.Stats("/project")
	if edges != 0 {
		t.Errorf("expected 0 edges after deleting caller node, got %d", edges)
	}
}

func TestDeleteNodesByDirectory(t *testing.T) {
	database := openTestDB(t)
	s := NewStore(database)

	s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "A", Kind: KindFunction})
	s.UpsertNode(CallNode{Directory: "/project", Package: "pkg", Symbol: "B", Kind: KindFunction})
	s.UpsertNode(CallNode{Directory: "/other", Package: "pkg", Symbol: "C", Kind: KindFunction})

	err := s.DeleteNodesByDirectory("/project")
	if err != nil {
		t.Fatalf("DeleteNodesByDirectory: %v", err)
	}

	nodes, _, _ := s.Stats("/project")
	if nodes != 0 {
		t.Errorf("expected 0 nodes after clearing /project, got %d", nodes)
	}

	elseNodes, _, _ := s.Stats("/other")
	if elseNodes != 1 {
		t.Errorf("expected 1 node in /other, got %d", elseNodes)
	}
}