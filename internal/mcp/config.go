package mcp

import (
	"os"
	"strings"
)

// ServerConfig defines how to connect to an MCP server.
type ServerConfig struct {
	Command string   // e.g., "ogden"
	Args    []string // e.g., ["mcp"]
	Env     []string // additional env vars
}

// ConfigFromEnv builds ServerConfig from OGCODE_MCP_* environment variables.
func ConfigFromEnv() ServerConfig {
	cfg := ServerConfig{
		Command: "ogden",
		Args:    []string{"mcp"},
	}

	if v := os.Getenv("OGCODE_MCP_COMMAND"); v != "" {
		cfg.Command = v
	}
	if v := os.Getenv("OGCODE_MCP_ARGS"); v != "" {
		cfg.Args = strings.Fields(v)
	}
	if v := os.Getenv("OGCODE_MCP_ENV"); v != "" {
		cfg.Env = strings.Fields(v)
	}

	return cfg
}