package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/prasenjeet-symon/ogcode/internal/agent"
	"github.com/prasenjeet-symon/ogcode/internal/bus"
	"github.com/prasenjeet-symon/ogcode/internal/db"
	"github.com/prasenjeet-symon/ogcode/internal/mcp"
	"github.com/prasenjeet-symon/ogcode/internal/memory"
	"github.com/prasenjeet-symon/ogcode/internal/plan"
	"github.com/prasenjeet-symon/ogcode/internal/provider"
	"github.com/prasenjeet-symon/ogcode/internal/session"
	"github.com/prasenjeet-symon/ogcode/internal/task"
	"github.com/prasenjeet-symon/ogcode/internal/tool"
)

// ServerMode determines the operational mode of the server.
type ServerMode string

const (
	ModeBuild ServerMode = "build"
	ModePlan  ServerMode = "plan"
)

type Server struct {
	port       int
	dir        string
	mode       ServerMode
	db         *db.DB
	bus        *bus.Bus
	store      *session.Store
	planStore  *plan.Store
	taskStore  *task.Store
	registry   *provider.Registry
	defaultProvider provider.Provider
	loopRunner *agent.LoopRunner
	mcpClient  *mcp.Client
	mcpCfg     mcp.ServerConfig
	mem        *memory.Memory

	// Track running agent loops so they can be cancelled on abort
	mu           sync.Mutex
	running      map[session.SessionID]context.CancelFunc
	runningToken map[session.SessionID]uint64 // prevents goroutine from deleting a newer cancel
	nextToken    uint64
}

func New(port int, dir string, mode ServerMode) *Server {
	return &Server{port: port, dir: dir, mode: mode, running: make(map[session.SessionID]context.CancelFunc), runningToken: make(map[session.SessionID]uint64)}
}

func (s *Server) Start() error {
	dbPath := filepath.Join(s.dir, ".ogcode", "ogcode.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}

	database, err := db.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	s.db = database
	s.bus = bus.New(256)
	s.store = session.NewStore(database)
	s.planStore = plan.NewStore(database)
	s.taskStore = task.NewStore(database)

	// Recover tasks that were in_progress when the server last stopped.
	if n, err := s.taskStore.FailStuckTasks(); err != nil {
		slog.Warn("recover stuck tasks", "err", err)
	} else if n > 0 {
		slog.Info("marked stuck tasks as failed", "count", n)
	}

	// Initialize provider
	registry := provider.NewRegistry()
	if apiKey := os.Getenv("ANTHROPIC_API_KEY"); apiKey != "" {
		registry.Register(provider.NewAnthropicProvider())
		slog.Info("registered anthropic provider")
	}
	if apiKey := os.Getenv("OPENAI_API_KEY"); apiKey != "" {
		registry.Register(provider.NewOpenAIProvider())
		slog.Info("registered openai provider")
	}
	if apiKey := os.Getenv("OPENROUTER_API_KEY"); apiKey != "" {
		registry.Register(provider.NewOpenRouterProvider())
		slog.Info("registered openrouter provider")
	}
	if os.Getenv("OLLAMA_BASE_URL") != "" || fileExists("/usr/local/bin/ollama") || fileExists("/opt/homebrew/bin/ollama") {
		registry.Register(provider.NewOllamaProvider())
		slog.Info("registered ollama provider")
	}

	// Initialize tools
	toolRegistry := tool.NewRegistry()
	toolRegistry.Register(tool.BashTool{})
	toolRegistry.Register(tool.ReadTool{})
	toolRegistry.Register(tool.WriteTool{})
	toolRegistry.Register(tool.EditTool{})
	toolRegistry.Register(tool.GlobTool{})
	toolRegistry.Register(tool.GrepTool{})
	toolRegistry.Register(tool.BreakdownTool{})

	// Determine default provider
	var defaultProvider provider.Provider
	for _, id := range registry.List() {
		defaultProvider = registry.Get(id)
		break
	}
	if defaultProvider == nil {
		slog.Warn("no LLM provider configured; set ANTHROPIC_API_KEY, OPENAI_API_KEY, OPENROUTER_API_KEY, or install Ollama")
		defaultProvider = provider.NewAnthropicProvider()
	}

	s.registry = registry
	s.defaultProvider = defaultProvider

	// Load custom model preferences from DB
	prefs, err := session.GetModelPreferences(s.db)
	if err != nil {
		slog.Warn("failed to load model preferences", "err", err)
	} else {
		for _, p := range prefs {
			if p.IsCustom {
				s.registry.RegisterCustomModel(p.ID, p.ProviderID)
				slog.Info("registered custom model", "id", p.ID, "provider", p.ProviderID)
			}
		}
	}

	// Initialize agentic memory via MCP
	var mem *memory.Memory
	if strings.EqualFold(os.Getenv("OGCODE_AGENTIC_MEMORY_MODE"), "true") {
		cfg := mcp.ConfigFromEnv()
		s.mcpCfg = cfg
		mcpClient, err := mcp.NewClient(context.Background(), cfg)
		if err != nil {
			slog.Warn("failed to connect to MCP memory server, memory disabled", "err", err)
		} else {
			s.mcpClient = mcpClient
			mem = memory.New(mcpClient)
			s.mem = mem
		}
	}

	s.loopRunner = &agent.LoopRunner{
		Store:           s.store,
		Bus:             s.bus,
		Registry:        registry,
		DefaultProvider: defaultProvider,
		Tools:           toolRegistry,
		Dir:             s.dir,
		Memory:          mem,
		MCP:             s.mcpClient,
	}

	r := s.routes()

	// Try ports starting from the configured port, up to 10 attempts.
	var listener net.Listener
	tryPort := s.port
	for i := 0; i < 10; i++ {
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", tryPort))
		if err == nil {
			listener = l
			s.port = tryPort
			break
		}
		if strings.Contains(err.Error(), "address already in use") {
			slog.Info("port in use, trying next", "port", tryPort)
			tryPort++
			continue
		}
		return fmt.Errorf("bind port: %w", err)
	}
	if listener == nil {
		return fmt.Errorf("no available port found (tried %d–%d)", s.port, tryPort-1)
	}

	addr := fmt.Sprintf(":%d", s.port)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  0,
		WriteTimeout: 0,
		IdleTimeout:  120 * time.Second,
	}

	url := fmt.Sprintf("http://localhost:%d", s.port)
	slog.Info("starting ogcode server", "addr", addr, "dir", s.dir)
	go openBrowser(url)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			quit <- syscall.SIGTERM
		}
	}()

	<-quit
	slog.Info("shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown HTTP server (closes all connections and releases port)
	if err := srv.Shutdown(ctx); err != nil {
		slog.Warn("server shutdown error", "err", err)
	}

	// Close listener explicitly
	if err := listener.Close(); err != nil {
		slog.Warn("close listener", "err", err)
	}

	// Close MCP client
	if s.mcpClient != nil {
		if err := s.mcpClient.Close(); err != nil {
			slog.Warn("close mcp client", "err", err)
		}
	}

	// Close database
	if s.db != nil {
		if err := s.db.Close(); err != nil {
			slog.Warn("close database", "err", err)
		}
	}

	slog.Info("server stopped, port released")
	return nil
}

func openBrowser(url string) {
	time.Sleep(500 * time.Millisecond)
	var cmd string
	var args []string
	switch {
	case fileExists("/usr/bin/open"):
		cmd, args = "open", []string{url}
	case fileExists("/usr/bin/xdg-open"):
		cmd, args = "xdg-open", []string{url}
	default:
		return
	}
	_ = exec.Command(cmd, args...).Start()
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}