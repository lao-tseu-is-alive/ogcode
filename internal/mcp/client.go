package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

// ToolDef describes a tool discovered from an MCP server.
type ToolDef struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

// Client wraps an MCP stdio client for communicating with MCP servers like ogden.
type Client struct {
	client *mcpclient.Client
	tools  map[string]ToolDef
	cfg    ServerConfig
	mu     sync.Mutex // serializes CallTool to prevent interleaved JSON-RPC on stdio
}

// NewClient spawns an MCP server as a child process and initializes the connection.
func NewClient(ctx context.Context, cfg ServerConfig) (*Client, error) {
	c := &Client{cfg: cfg}

	// Build env: inherit current env + extra vars
	env := os.Environ()
	env = append(env, cfg.Env...)

	client, err := mcpclient.NewStdioMCPClient(cfg.Command, env, cfg.Args...)
	if err != nil {
		return nil, fmt.Errorf("spawn mcp server %q: %w", cfg.Command, err)
	}
	c.client = client

	// Initialize handshake
	initReq := mcp.InitializeRequest{
		Params: mcp.InitializeParams{
			ProtocolVersion: "2024-11-05",
			ClientInfo: mcp.Implementation{
				Name:    "ogcode",
				Version: "0.1.0",
			},
		},
	}
	initCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := client.Initialize(initCtx, initReq); err != nil {
		client.Close()
		return nil, fmt.Errorf("mcp initialize: %w", err)
	}

	// Discover tools
	toolsCtx, toolsCancel := context.WithTimeout(ctx, 5*time.Second)
	defer toolsCancel()

	result, err := client.ListTools(toolsCtx, mcp.ListToolsRequest{})
	if err != nil {
		client.Close()
		return nil, fmt.Errorf("mcp list tools: %w", err)
	}

	c.tools = make(map[string]ToolDef, len(result.Tools))
	for _, t := range result.Tools {
		schema, _ := json.Marshal(t.InputSchema)
		c.tools[t.Name] = ToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: schema,
		}
	}

	slog.Info("mcp client connected", "server", cfg.Command, "tools", len(c.tools))
	for name := range c.tools {
		slog.Info("mcp tool discovered", "name", name)
	}

	return c, nil
}

// Close shuts down the MCP client and its child process.
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// HasTool returns true if the server has a tool whose name contains the given substring.
func (c *Client) HasTool(name string) bool {
	for k := range c.tools {
		if strings.Contains(k, name) {
			return true
		}
	}
	return false
}

// Tools returns all discovered tool definitions.
func (c *Client) Tools() map[string]ToolDef {
	return c.tools
}

// CallTool invokes a tool on the MCP server.
// Uses the provided context for timeout control — callers should set appropriate deadlines.
// Returns the text content of the response, duration of the call, or empty string on error.
func (c *Client) CallTool(ctx context.Context, name string, arguments map[string]any) (string, time.Duration, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	start := time.Now()
	result, err := c.client.CallTool(callCtx, mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      name,
			Arguments: arguments,
		},
	})
	duration := time.Since(start)

	if err != nil {
		slog.Info("mcp tool call", "tool", name, "duration_ms", duration.Milliseconds(), "err", err)
		return "", duration, fmt.Errorf("call tool %q: %w", name, err)
	}

	slog.Info("mcp tool call", "tool", name, "duration_ms", duration.Milliseconds())
	return extractText(result), duration, nil
}

// extractText pulls text content from a CallToolResult, following the same
// extraction logic as opencode: try content[0].text, then text, then output.
func extractText(result *mcp.CallToolResult) string {
	if result == nil {
		return ""
	}

	for _, c := range result.Content {
		if tc, ok := c.(mcp.TextContent); ok && tc.Text != "" {
			return tc.Text
		}
	}

	b, _ := json.Marshal(result)
	return string(b)
}