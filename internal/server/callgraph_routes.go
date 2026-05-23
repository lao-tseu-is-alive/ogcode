package server

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/prasenjeet-symon/ogcode/internal/callgraph"
	"github.com/prasenjeet-symon/ogcode/internal/session"
)

// ─── Call Graph Model Preference ───

func (s *Server) handleGetCallGraphModel(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	var model string
	row := s.db.QueryRow(`SELECT model FROM callgraph_config WHERE directory = ?`, directory)
	_ = row.Scan(&model)
	writeJSON(w, http.StatusOK, map[string]string{"model": model})
}

func (s *Server) handleSetCallGraphModel(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Directory string `json:"directory"`
		Model     string `json:"model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if input.Directory == "" {
		input.Directory = s.dir
	}
	_, err := s.db.Exec(
		`INSERT INTO callgraph_config (directory, model) VALUES (?, ?)
		 ON CONFLICT(directory) DO UPDATE SET model = excluded.model`,
		input.Directory, input.Model,
	)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"model": input.Model})
}

// ─── Call Graph Build ───

func (s *Server) handleCallGraphBuildStatus(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	sessID := s.cgBuildSessID
	_, running := s.running[sessID]
	s.mu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"running":   running,
		"sessionId": string(sessID),
	})
}

func (s *Server) handleBuildCallGraph(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Directory string `json:"directory"`
		Rebuild   bool   `json:"rebuild"`
		Model     string `json:"model,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	dir := input.Directory
	if dir == "" {
		dir = s.dir
	}

	// Reject if a build is already running
	s.mu.Lock()
	if s.cgBuildSessID != "" {
		if _, running := s.running[s.cgBuildSessID]; running {
			sid := s.cgBuildSessID
			s.mu.Unlock()
			writeJSON(w, http.StatusConflict, map[string]any{
				"running":   true,
				"sessionId": string(sid),
			})
			return
		}
	}
	s.mu.Unlock()

	// Clear existing graph data if this is a rebuild
	if input.Rebuild {
		if err := s.callgraphStore.DeleteNodesByDirectory(dir); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create a dedicated session for the build
	sess := &session.Session{
		ID:          session.NewSessionID(),
		ProjectID:   dir,
		Directory:   dir,
		Title:       "Build call graph",
		Model:       input.Model,
		SessionType: "callgraph",
		CreatedAt:   session.Now(),
		UpdatedAt:   session.Now(),
	}
	if err := s.store.Create(sess); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track as the current build session
	s.mu.Lock()
	s.cgBuildSessID = sess.ID
	s.mu.Unlock()

	go func() {
		// Seed the conversation with a user message describing the task
		userMsg := &session.MessageInfo{
			ID:        session.NewMessageID(),
			SessionID: sess.ID,
			Role:      session.RoleUser,
			Agent:     "callgraph",
			CreatedAt: session.Now(),
		}
		if err := s.store.CreateMessage(userMsg); err != nil {
			slog.Error("callgraph build: create user message", "err", err)
			return
		}
		prompt := "Build the complete code knowledge graph for this project. Discover all source files, extract every code symbol (functions, methods, types, interfaces, enums, constants, modules), and record all relationships between them. Be thorough and systematic — process every file."
		textData, _ := json.Marshal(session.TextPartData{Text: prompt})
		userPart := &session.Part{
			ID:        session.NewPartID(),
			MessageID: userMsg.ID,
			SessionID: sess.ID,
			Type:      session.PartText,
			Data:      textData,
			CreatedAt: session.Now(),
			UpdatedAt: session.Now(),
		}
		if err := s.store.CreatePart(userPart); err != nil {
			slog.Error("callgraph build: create user part", "err", err)
			return
		}
		s.bus.Publish("message.updated", userMsg)

		ctx, cancel := context.WithCancel(context.Background())
		s.mu.Lock()
		s.nextToken++
		token := s.nextToken
		s.running[sess.ID] = cancel
		s.runningToken[sess.ID] = token
		s.mu.Unlock()

		defer func() {
			s.mu.Lock()
			if s.runningToken[sess.ID] == token {
				delete(s.running, sess.ID)
				delete(s.runningToken, sess.ID)
			}
			s.mu.Unlock()
		}()

		if err := s.loopRunner.RunLoop(ctx, sess.ID, "callgraph"); err != nil {
			slog.Error("callgraph build loop error", "session", sess.ID, "err", err)
		}
	}()

	writeJSON(w, http.StatusCreated, map[string]any{
		"sessionId": string(sess.ID),
	})
}

// ─── Call Graph API ───

func (s *Server) handleCallGraphStats(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	nodes, edges, err := s.callgraphStore.Stats(directory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"nodes": nodes,
		"edges": edges,
	})
}

func (s *Server) handleCallGraphNodes(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	pkg := r.URL.Query().Get("package")
	kind := r.URL.Query().Get("kind")

	nodes, err := s.callgraphStore.ListNodesByDirectory(directory, pkg, callgraph.NodeKind(kind))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nodes == nil {
		writeJSON(w, http.StatusOK, []CallNodeResponse{})
		return
	}
	result := make([]CallNodeResponse, len(nodes))
	for i, n := range nodes {
		result[i] = toNodeResponse(n)
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCallGraphEdges(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	rawEdges, err := s.callgraphStore.ListEdgesByDirectory(directory)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result := make([]CallEdgeResponse, len(rawEdges))
	for i, e := range rawEdges {
		result[i] = CallEdgeResponse{
			ID:       e.ID,
			CallerID: e.CallerID,
			CalleeID: e.CalleeID,
			CallType: string(e.CallType),
		}
	}
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) handleCallGraphNodeDetail(w http.ResponseWriter, r *http.Request) {
	nodeIDStr := chi.URLParam(r, "nodeID")
	nodeID, err := strconv.ParseInt(nodeIDStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid node id", http.StatusBadRequest)
		return
	}

	node, err := s.callgraphStore.GetNode(nodeID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if node == nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	callees, _ := s.callgraphStore.CalleesOf(nodeID)
	callers, _ := s.callgraphStore.CallersOf(nodeID)

	var calleeList []CallNodeSummary
	for _, c := range callees {
		calleeList = append(calleeList, CallNodeSummary{
			ID:       c.Callee.ID,
			Package:  c.Callee.Package,
			Symbol:   c.Callee.Symbol,
			FilePath: c.Callee.FilePath,
			Line:     c.Callee.Line,
			Kind:     string(c.Callee.Kind),
			Doc:      c.Callee.Doc,
			CallType: string(c.Edge.CallType),
		})
	}
	if calleeList == nil {
		calleeList = []CallNodeSummary{}
	}

	var callerList []CallNodeSummary
	for _, c := range callers {
		callerList = append(callerList, CallNodeSummary{
			ID:       c.Caller.ID,
			Package:  c.Caller.Package,
			Symbol:   c.Caller.Symbol,
			FilePath: c.Caller.FilePath,
			Line:     c.Caller.Line,
			Kind:     string(c.Caller.Kind),
			Doc:      c.Caller.Doc,
			CallType: string(c.Edge.CallType),
		})
	}
	if callerList == nil {
		callerList = []CallNodeSummary{}
	}

	writeJSON(w, http.StatusOK, CallNodeDetailResponse{
		Node:    toNodeResponse(*node),
		Callees: calleeList,
		Callers: callerList,
	})
}

func (s *Server) handleCallGraphSearch(w http.ResponseWriter, r *http.Request) {
	directory := r.URL.Query().Get("directory")
	if directory == "" {
		directory = s.dir
	}
	query := r.URL.Query().Get("q")
	if query == "" {
		http.Error(w, "q parameter is required", http.StatusBadRequest)
		return
	}

	nodes, err := s.callgraphStore.SearchNodes(directory, query, 50)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if nodes == nil {
		writeJSON(w, http.StatusOK, []CallNodeResponse{})
		return
	}
	result := make([]CallNodeResponse, len(nodes))
	for i, n := range nodes {
		result[i] = toNodeResponse(n)
	}
	writeJSON(w, http.StatusOK, result)
}

// ─── Response types ───

type CallNodeResponse struct {
	ID        int64  `json:"id"`
	Directory string `json:"directory"`
	Package   string `json:"package"`
	Symbol    string `json:"symbol"`
	FilePath  string `json:"filePath"`
	Line      int    `json:"line"`
	Kind      string `json:"kind"`
	Signature string `json:"signature,omitempty"`
	Doc       string `json:"doc,omitempty"`
}

type CallEdgeResponse struct {
	ID       int64  `json:"id"`
	CallerID int64  `json:"callerId"`
	CalleeID int64  `json:"calleeId"`
	CallType string `json:"callType"`
}

type CallNodeSummary struct {
	ID       int64  `json:"id"`
	Package  string `json:"package"`
	Symbol   string `json:"symbol"`
	FilePath string `json:"filePath"`
	Line     int    `json:"line"`
	Kind     string `json:"kind"`
	Doc      string `json:"doc,omitempty"`
	CallType string `json:"callType,omitempty"`
}

type CallNodeDetailResponse struct {
	Node    CallNodeResponse  `json:"node"`
	Callees []CallNodeSummary `json:"callees"`
	Callers []CallNodeSummary `json:"callers"`
}

func toNodeResponse(n callgraph.CallNode) CallNodeResponse {
	return CallNodeResponse{
		ID:        n.ID,
		Directory: n.Directory,
		Package:   n.Package,
		Symbol:    n.Symbol,
		FilePath:  n.FilePath,
		Line:      n.Line,
		Kind:      string(n.Kind),
		Signature: n.Signature,
		Doc:       n.Doc,
	}
}