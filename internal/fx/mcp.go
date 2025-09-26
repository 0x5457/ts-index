package fx

import (
	"context"
	"fmt"

	"github.com/0x5457/ts-index/internal/indexer"
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// MCPParams represents dependencies for MCP server
type MCPParams struct {
	fx.In

	SearchService *search.Service
	Indexer       indexer.Indexer
	Config        *Config
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(params MCPParams) *server.MCPServer {
	return appmcp.New(params.SearchService, params.Indexer)
}

// MCPLifecycle manages MCP server lifecycle
type MCPLifecycle struct {
	server  *server.MCPServer
	indexer indexer.Indexer
	config  *Config
}

// NewMCPLifecycle creates a new MCP lifecycle manager
func NewMCPLifecycle(srv *server.MCPServer, indexer indexer.Indexer, config *Config) *MCPLifecycle {
	return &MCPLifecycle{
		server:  srv,
		indexer: indexer,
		config:  config,
	}
}

// Start initializes the MCP server and handles pre-indexing
func (m *MCPLifecycle) Start(ctx context.Context) error {
	// Pre-index project if specified
	if m.config.Project != "" {
		if err := m.indexer.IndexProject(m.config.Project); err != nil {
			return fmt.Errorf("pre-index project failed: %w", err)
		}
	}
	return nil
}

// Stop handles graceful shutdown
func (m *MCPLifecycle) Stop(ctx context.Context) error {
	// MCP server cleanup is handled by the framework
	return nil
}

// MCPModule provides MCP server components
var MCPModule = fx.Module("mcp",
	fx.Provide(
		NewMCPServer,
		NewMCPLifecycle,
	),
)
