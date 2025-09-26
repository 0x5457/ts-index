# ts-index

A powerful TypeScript code indexing and search tool with Language Server Protocol support and MCP (Model Context Protocol) integration.

## Key Features

- **Semantic Search**: Advanced code search using embeddings for natural language queries
- **Symbol Search**: Exact symbol name matching and lookup
- **LSP Integration**: Language Server Protocol support for TypeScript analysis
- **MCP Server**: Exposes indexing and search capabilities through Model Context Protocol
- **Vector Database**: SQLite-based storage with vector search capabilities
- **Multiple Transports**: Support for stdio, HTTP, and SSE communication protocols

## Usage

### Index a TypeScript project

```bash
ts-index index --project /path/to/project --db /path/to/index.db
```

### Search code semantically

```bash
ts-index search "function to parse JSON" --project /path/to/project --db /path/to/index.db
```

### Search by exact symbol name

```bash
ts-index search "parseJSON" --symbol --db /path/to/index.db
```

### Language Server Protocol commands

```bash
# Analyze symbol at position
ts-index lsp analyze src/utils.ts --project /path/to/project --line 10 --character 5

# Get code completions
ts-index lsp completion src/utils.ts --project /path/to/project --line 10 --character 5

# Search workspace symbols
ts-index lsp symbols --project /path/to/project --query "parse"

# Install language server
ts-index lsp install vtsls

# Check LSP health
ts-index lsp health
```

### Run MCP server

```bash
# stdio mode (default)
ts-index mcp --project /path/to/project --db /path/to/index.db

# HTTP mode
ts-index mcp --transport http --address :8080 --db /path/to/index.db

# SSE mode
ts-index mcp --transport sse --address :8080 --db /path/to/index.db
```

## Development

### Prerequisites

- Go 1.24.5 or later
- Make

### Commands

```bash
# Download dependencies
make deps

# Run tests
make test

# Run linter
make lint

# Run linter with auto-fix
make lint-fix

# Clean build artifacts
make clean

# Show all available commands
make help
```

### Building

```bash
go build -o bin/ts-index ./cmd/ts-index
```
