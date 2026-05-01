# Ogcode

An agentic coding assistant with a web UI, written in Go.

## Overview

Ogcode is an AI-powered coding assistant that provides a web-based interface for interacting with AI agents. It supports the Model Context Protocol (MCP) for extending functionality and uses SQLite for persistent storage.

## Features

- **Web UI** — Modern interface for interacting with the AI agent
- **MCP Integration** — Extend functionality via the Model Context Protocol
- **CLI Tool** — Command-line interface powered by Cobra
- **SQLite Database** — Persistent storage for sessions and memory
- **Agentic AI** — AI-powered coding assistance
- **Event Bus** — Internal event-driven architecture
- **Session Management** — Track and manage user sessions
- **Tool System** — Extensible tool definitions for AI agents
- **Permission Handling** — Built-in permission management

## Architecture

```
ogcode/
├── main.go              # Entry point
├── internal/
│   ├── agent/           # AI agent logic
│   ├── bus/             # Event bus
│   ├── cli/             # CLI commands (Cobra)
│   ├── db/              # Database layer
│   ├── id/              # ID generation (ULID)
│   ├── mcp/             # MCP server implementation
│   ├── memory/          # Memory/knowledge management
│   ├── permission/      # Permission handling
│   ├── provider/        # AI provider integrations
│   ├── server/          # HTTP server (chi router)
│   ├── session/         # Session management
│   └── tool/            # Tool definitions for the agent
└── web/                 # Web UI assets
```

### Component Details

| Package | Description |
|---------|-------------|
| `agent` | Core AI agent logic, handles tool execution and AI interactions |
| `bus` | Event bus for internal component communication |
| `cli` | Cobra-based command-line interface |
| `db` | SQLite database layer with migrations |
| `id` | ULID-based unique ID generation |
| `mcp` | Model Context Protocol server implementation |
| `memory` | Knowledge and context management |
| `permission` | User and tool permission handling |
| `provider` | Abstraction for AI provider integrations (OpenAI, Anthropic, etc.) |
| `server` | HTTP REST API server using chi router |
| `session` | Session lifecycle management |
| `tool` | Tool definitions available to the AI agent |

## Requirements

- Go 1.26.1 or later
- SQLite (embedded via modernc.org/sqlite)

## Installation

```bash
# Clone the repository
git clone https://github.com/ogcode/ogcode.git
cd ogcode

# Build the project
go build

# Or install globally
go install
```

## Configuration

Ogcode can be configured via environment variables or command-line flags.

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OGCODE_PORT` | Server port | `8080` |
| `OGCODE_HOST` | Server host | `0.0.0.0` |
| `OGCODE_DB_PATH` | Path to SQLite database | `./ogcode.db` |
| `OGCODE_LOG_LEVEL` | Log level (debug, info, warn, error) | `info` |
| `OGCODE_AI_PROVIDER` | AI provider to use | `openai` |
| `OGCODE_AI_API_KEY` | API key for AI provider | - |
| `OGCODE_MCP_ENABLED` | Enable MCP server | `true` |
| `OGCODE_MCP_PORT` | MCP server port | `8081` |

### CLI Flags

```bash
# Start server with custom port
ogcode serve --port 3000

# Start with custom database path
ogcode serve --db-path /path/to/database.db

# Enable debug logging
ogcode serve --debug

# Start MCP server
ogcode serve --mcp
```

## Usage

### Start the Web Server

```bash
# Default port (8080)
go run . serve

# Custom port
go run . serve --port 3000

# With debug logging
go run . serve --debug
```

### CLI Commands

```bash
# Start server (default command)
ogcode serve

# Show help
ogcode --help

# Show version
ogcode version
```

## API Endpoints

The REST API provides the following endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/health` | Health check endpoint |
| `GET` | `/api/sessions` | List all sessions |
| `POST` | `/api/sessions` | Create a new session |
| `GET` | `/api/sessions/:id` | Get session by ID |
| `DELETE` | `/api/sessions/:id` | Delete a session |
| `POST` | `/api/chat` | Send a chat message |
| `GET` | `/api/tools` | List available tools |
| `GET` | `/api/memory` | Get agent memory/knowledge |
| `POST` | `/api/memory` | Add to agent memory |

### WebSocket Endpoints

| Endpoint | Description |
|----------|-------------|
| `/ws/chat` | Real-time chat streaming |

## MCP (Model Context Protocol)

Ogcode implements the MCP protocol, allowing external tools and services to integrate with the agent.

### MCP Tools

The MCP server exposes the following tools:

- `execute_code` — Execute code in a sandboxed environment
- `read_file` — Read files from the filesystem
- `write_file` — Write content to files
- `list_directory` — List directory contents
- `search_code` — Search through code files
- `run_command` — Execute shell commands

### Connecting MCP Clients

```bash
# Start MCP server
ogcode serve --mcp

# MCP server runs on port 8081 by default
# Connect using any MCP-compatible client
```

## Database Schema

Ogcode uses SQLite for persistent storage. The database includes the following tables:

### Sessions Table

```sql
CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL,
    metadata TEXT
);
```

### Messages Table

```sql
CREATE TABLE messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    FOREIGN KEY (session_id) REFERENCES sessions(id)
);
```

### Memory Table

```sql
CREATE TABLE memory (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at TIMESTAMP NOT NULL,
    updated_at TIMESTAMP NOT NULL
);
```

### Tools Table

```sql
CREATE TABLE tools (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    schema TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL
);
```

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/agent/...
```

### Database Migrations

```bash
# Run migrations
goose -dir internal/db/migrations sqlite3:./ogcode.db up

# Create new migration
goose -dir internal/db/migrations sqlite3:./ogcode.db create migration_name sql
```

### Code Generation

```bash
# Generate code (if using generate directives)
go generate ./...
```

## Tech Stack

- **Language:** Go 1.26.1
- **Web Framework:** chi/v5
- **CLI:** spf13/cobra
- **Database:** modernc.org/sqlite
- **MCP:** mark3labs/mcp-go
- **ID Generation:** oklog/ulid/v2
- **Database Migrations:** pressly/goose/v3

## Contributing

Contributions are welcome! Please follow these steps:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Code Style

- Follow Go standard formatting (`go fmt`)
- Run `go vet` before committing
- Ensure all tests pass
- Add tests for new functionality

## License

MIT License - see [LICENSE](LICENSE) for details.

## Related Projects

- [Model Context Protocol](https://github.com/modelcontextprotocol) - MCP specification
- [mcp-go](https://github.com/mark3labs/mcp-go) - Go MCP implementation
- [chi](https://github.com/go-chi/chi) - Lightweight HTTP router

## Support

- Open an issue on GitHub for bugs or feature requests
- Join the community Discord for discussions