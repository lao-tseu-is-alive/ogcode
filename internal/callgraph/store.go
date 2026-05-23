package callgraph

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/prasenjeet-symon/ogcode/internal/db"
)

// Store provides persistence operations for call graph nodes and edges.
type Store struct {
	db *db.DB
}

// NewStore creates a new Store backed by the given database.
func NewStore(database *db.DB) *Store {
	return &Store{db: database}
}

// ──── Node CRUD ────

// UpsertNode inserts a call node or updates it if one with the same
// (directory, package, symbol) already exists. Returns the node with its ID set.
func (s *Store) UpsertNode(n CallNode) (*CallNode, error) {
	now := time.Now().UnixMilli()
	res, err := s.db.Exec(`
		INSERT INTO call_node (directory, package, symbol, file_path, line, kind, signature, doc, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(directory, package, symbol) DO UPDATE SET
			file_path = excluded.file_path,
			line = excluded.line,
			kind = excluded.kind,
			signature = excluded.signature,
			doc = excluded.doc,
			updated_at = excluded.updated_at
	`, n.Directory, n.Package, n.Symbol, n.FilePath, n.Line, string(n.Kind), n.Signature, n.Doc, now, now)
	if err != nil {
		return nil, fmt.Errorf("upsert call node: %w", err)
	}
	id, _ := res.LastInsertId()
	n.ID = id
	n.UpdatedAt = now
	if n.CreatedAt == 0 {
		n.CreatedAt = now
	}
	return &n, nil
}

// GetNode retrieves a call node by its row ID.
func (s *Store) GetNode(id int64) (*CallNode, error) {
	row := s.db.QueryRow(`
		SELECT id, directory, package, symbol, file_path, line, kind, signature, doc, created_at, updated_at
		FROM call_node WHERE id = ?
	`, id)
	return scanNode(row)
}

// GetNodeBySymbol retrieves a call node by directory + package + symbol.
func (s *Store) GetNodeBySymbol(directory, pkg, symbol string) (*CallNode, error) {
	row := s.db.QueryRow(`
		SELECT id, directory, package, symbol, file_path, line, kind, signature, doc, created_at, updated_at
		FROM call_node WHERE directory = ? AND package = ? AND symbol = ?
	`, directory, pkg, symbol)
	return scanNode(row)
}

// ListNodesByDirectory returns all call nodes for a directory, optionally filtered by package or kind.
func (s *Store) ListNodesByDirectory(directory string, pkgFilter string, kindFilter NodeKind) ([]CallNode, error) {
	query := `SELECT id, directory, package, symbol, file_path, line, kind, signature, doc, created_at, updated_at
			  FROM call_node WHERE directory = ?`
	args := []any{directory}
	if pkgFilter != "" {
		query += ` AND package = ?`
		args = append(args, pkgFilter)
	}
	if kindFilter != "" {
		query += ` AND kind = ?`
		args = append(args, string(kindFilter))
	}
	query += ` ORDER BY package, symbol`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list call nodes: %w", err)
	}
	defer rows.Close()

	var nodes []CallNode
	for rows.Next() {
		n, err := scanNodeRows(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *n)
	}
	return nodes, rows.Err()
}

// DeleteNode removes a call node by ID and its associated edges.
func (s *Store) DeleteNode(id int64) error {
	// Edges are cascade-deleted via foreign key, but we delete explicitly
	// to avoid depending on FK triggers being enabled.
	_, err := s.db.Exec(`DELETE FROM call_edge WHERE caller_id = ? OR callee_id = ?`, id, id)
	if err != nil {
		return fmt.Errorf("delete edges for node: %w", err)
	}
	_, err = s.db.Exec(`DELETE FROM call_node WHERE id = ?`, id)
	return err
}

// DeleteNodesByFile removes all call nodes (and their edges) for a given file path
// within a directory. This is used to purge stale call graph data after a file is
// mutated, so the agent can re-populate it with accurate information.
func (s *Store) DeleteNodesByFile(directory, filePath string) (int64, error) {
	// First, collect the IDs of nodes in this file so we can delete their edges.
	rows, err := s.db.Query(`
		SELECT id FROM call_node
		WHERE directory = ? AND file_path = ?
	`, directory, filePath)
	if err != nil {
		return 0, fmt.Errorf("query nodes by file: %w", err)
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan node id: %w", err)
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, err
	}

	if len(ids) == 0 {
		return 0, nil
	}

	// Delete edges referencing these nodes.
	for _, id := range ids {
		_, err := s.db.Exec(`DELETE FROM call_edge WHERE caller_id = ? OR callee_id = ?`, id, id)
		if err != nil {
			return 0, fmt.Errorf("delete edges for node %d: %w", id, err)
		}
	}

	// Delete the nodes themselves.
	res, err := s.db.Exec(`DELETE FROM call_node WHERE directory = ? AND file_path = ?`, directory, filePath)
	if err != nil {
		return 0, fmt.Errorf("delete nodes by file: %w", err)
	}
	deleted, _ := res.RowsAffected()
	return deleted, nil
}

// DeleteNodesByDirectory removes all call nodes and edges for a directory.
func (s *Store) DeleteNodesByDirectory(directory string) error {
	_, err := s.db.Exec(`DELETE FROM call_edge WHERE directory = ?`, directory)
	if err != nil {
		return fmt.Errorf("delete edges for directory: %w", err)
	}
	_, err = s.db.Exec(`DELETE FROM call_node WHERE directory = ?`, directory)
	return err
}

// ──── Edge CRUD ────

// AddEdge creates a call edge between a caller and callee.
func (s *Store) AddEdge(e CallEdge) (*CallEdge, error) {
	now := time.Now().UnixMilli()
	res, err := s.db.Exec(`
		INSERT INTO call_edge (directory, caller_id, callee_id, call_type, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(directory, caller_id, callee_id, call_type) DO UPDATE SET
			created_at = excluded.created_at
	`, e.Directory, e.CallerID, e.CalleeID, string(e.CallType), now)
	if err != nil {
		return nil, fmt.Errorf("add call edge: %w", err)
	}
	id, _ := res.LastInsertId()
	e.ID = id
	e.CreatedAt = now
	return &e, nil
}

// AddEdgesBatch inserts multiple edges in a single transaction.
func (s *Store) AddEdgesBatch(edges []CallEdge) error {
	tx, err := s.db.BeginTx(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO call_edge (directory, caller_id, callee_id, call_type, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(directory, caller_id, callee_id, call_type) DO UPDATE SET
			created_at = excluded.created_at
	`)
	if err != nil {
		return fmt.Errorf("prepare edge insert: %w", err)
	}
	defer stmt.Close()

	now := time.Now().UnixMilli()
	for _, e := range edges {
		_, err := stmt.Exec(e.Directory, e.CallerID, e.CalleeID, string(e.CallType), now)
		if err != nil {
			return fmt.Errorf("insert edge: %w", err)
		}
	}
	return tx.Commit()
}

// CalleesOf returns all nodes that the given node calls, along with the edge info.
func (s *Store) CalleesOf(nodeID int64) ([]CalleeOfResult, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.directory, e.caller_id, e.callee_id, e.call_type, e.created_at,
		       n.id, n.directory, n.package, n.symbol, n.file_path, n.line, n.kind, n.signature, n.doc, n.created_at, n.updated_at
		FROM call_edge e
		JOIN call_node n ON n.id = e.callee_id
		WHERE e.caller_id = ?
		ORDER BY n.package, n.symbol
	`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("query callees: %w", err)
	}
	defer rows.Close()

	var results []CalleeOfResult
	for rows.Next() {
		var e CallEdge
		var n CallNode
		var callType string
		var kind string
		if err := rows.Scan(&e.ID, &e.Directory, &e.CallerID, &e.CalleeID, &callType, &e.CreatedAt,
			&n.ID, &n.Directory, &n.Package, &n.Symbol, &n.FilePath, &n.Line, &kind, &n.Signature, &n.Doc, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		e.CallType = CallType(callType)
		n.Kind = NodeKind(kind)
		results = append(results, CalleeOfResult{Edge: e, Callee: n})
	}
	return results, rows.Err()
}

// CallersOf returns all nodes that call the given node, along with the edge info.
func (s *Store) CallersOf(nodeID int64) ([]CallerOfResult, error) {
	rows, err := s.db.Query(`
		SELECT e.id, e.directory, e.caller_id, e.callee_id, e.call_type, e.created_at,
		       n.id, n.directory, n.package, n.symbol, n.file_path, n.line, n.kind, n.signature, n.doc, n.created_at, n.updated_at
		FROM call_edge e
		JOIN call_node n ON n.id = e.caller_id
		WHERE e.callee_id = ?
		ORDER BY n.package, n.symbol
	`, nodeID)
	if err != nil {
		return nil, fmt.Errorf("query callers: %w", err)
	}
	defer rows.Close()

	var results []CallerOfResult
	for rows.Next() {
		var e CallEdge
		var n CallNode
		var callType string
		var kind string
		if err := rows.Scan(&e.ID, &e.Directory, &e.CallerID, &e.CalleeID, &callType, &e.CreatedAt,
			&n.ID, &n.Directory, &n.Package, &n.Symbol, &n.FilePath, &n.Line, &kind, &n.Signature, &n.Doc, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		e.CallType = CallType(callType)
		n.Kind = NodeKind(kind)
		results = append(results, CallerOfResult{Edge: e, Caller: n})
	}
	return results, rows.Err()
}

// ReachableFrom returns all nodes reachable from the given node via call edges
// (transitive closure up to maxDepth hops). maxDepth=0 means unlimited.
func (s *Store) ReachableFrom(nodeID int64, maxDepth int) ([]CallNode, error) {
	if maxDepth <= 0 {
		maxDepth = 20 // sensible default cap
	}

	// Iterative BFS using a visited set
	visited := map[int64]bool{nodeID: true}
	queue := []int64{nodeID}
	var result []CallNode

	for depth := 0; depth < maxDepth && len(queue) > 0; depth++ {
		var nextQueue []int64
		for _, id := range queue {
			callees, err := s.CalleesOf(id)
			if err != nil {
				return nil, err
			}
			for _, c := range callees {
				if !visited[c.Edge.CalleeID] {
					visited[c.Edge.CalleeID] = true
					nextQueue = append(nextQueue, c.Edge.CalleeID)
					result = append(result, c.Callee)
				}
			}
		}
		queue = nextQueue
	}
	return result, nil
}

// SearchNodes searches call nodes by substring matching on symbol names, doc
// text, and signatures. It returns nodes whose symbol, doc, or signature
// contains the query string (case-insensitive). This enables discovery-oriented
// queries like "find all functions related to encryption" or "where is Store
// defined" without knowing exact package or symbol names — replacing many
// grep + read cycles with a single call graph query.
func (s *Store) SearchNodes(directory, query string, limit int) ([]CallNode, error) {
	if limit <= 0 {
		limit = 50
	}
	pattern := "%" + strings.ToLower(query) + "%"
	rows, err := s.db.Query(`
		SELECT id, directory, package, symbol, file_path, line, kind, signature, doc, created_at, updated_at
		FROM call_node
		WHERE directory = ?
		  AND (LOWER(symbol) LIKE ? OR LOWER(doc) LIKE ? OR LOWER(signature) LIKE ?)
		ORDER BY
		  CASE
			WHEN LOWER(symbol) = LOWER(?) THEN 0
			WHEN LOWER(symbol) LIKE LOWER(?) THEN 1
			ELSE 2
		  END,
		  package, symbol
		LIMIT ?
	`, directory, pattern, pattern, pattern, query, "%"+strings.ToLower(query)+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("search call nodes: %w", err)
	}
	defer rows.Close()

	var nodes []CallNode
	for rows.Next() {
		n, err := scanNodeRows(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, *n)
	}
	return nodes, rows.Err()
}

// ListEdgesByDirectory returns all call edges for a directory.
func (s *Store) ListEdgesByDirectory(directory string) ([]CallEdge, error) {
	rows, err := s.db.Query(`
		SELECT id, directory, caller_id, callee_id, call_type, created_at
		FROM call_edge WHERE directory = ?
		ORDER BY caller_id, callee_id
	`, directory)
	if err != nil {
		return nil, fmt.Errorf("list call edges: %w", err)
	}
	defer rows.Close()

	var edges []CallEdge
	for rows.Next() {
		e, err := scanEdgeRow(rows)
		if err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

// Stats returns the count of nodes and edges for a directory.
func (s *Store) Stats(directory string) (nodes int, edges int, err error) {
	row := s.db.QueryRow(`SELECT COUNT(*) FROM call_node WHERE directory = ?`, directory)
	if err := row.Scan(&nodes); err != nil {
		return 0, 0, err
	}
	row = s.db.QueryRow(`SELECT COUNT(*) FROM call_edge WHERE directory = ?`, directory)
	if err := row.Scan(&edges); err != nil {
		return 0, 0, err
	}
	return nodes, edges, nil
}

// ──── Internal helpers ────

func scanEdgeRow(rows *sql.Rows) (CallEdge, error) {
	var e CallEdge
	var callType string
	if err := rows.Scan(&e.ID, &e.Directory, &e.CallerID, &e.CalleeID, &callType, &e.CreatedAt); err != nil {
		return CallEdge{}, err
	}
	e.CallType = CallType(callType)
	return e, nil
}

func scanNode(row *sql.Row) (*CallNode, error) {
	var n CallNode
	var kind string
	err := row.Scan(&n.ID, &n.Directory, &n.Package, &n.Symbol, &n.FilePath, &n.Line, &kind, &n.Signature, &n.Doc, &n.CreatedAt, &n.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("scan call node: %w", err)
	}
	n.Kind = NodeKind(kind)
	return &n, nil
}

func scanNodeRows(rows *sql.Rows) (*CallNode, error) {
	var n CallNode
	var kind string
	err := rows.Scan(&n.ID, &n.Directory, &n.Package, &n.Symbol, &n.FilePath, &n.Line, &kind, &n.Signature, &n.Doc, &n.CreatedAt, &n.UpdatedAt)
	if err != nil {
		return nil, err
	}
	n.Kind = NodeKind(kind)
	return &n, nil
}