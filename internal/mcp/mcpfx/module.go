package mcpfx

import (
	"context"
	"fmt"

	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/indexer"
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// Params represents dependencies for MCP server
type Params struct {
	fx.In

	SearchService *search.Service
	Indexer       indexer.Indexer
	Config        *configfx.Config
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer(params Params) *server.MCPServer {
	return appmcp.New(params.SearchService, params.Indexer)
}

// Lifecycle manages MCP server lifecycle
type Lifecycle struct {
	server  *server.MCPServer
	indexer indexer.Indexer
	config  *configfx.Config
}

// NewLifecycle creates a new MCP lifecycle manager
func NewLifecycle(
	srv *server.MCPServer,
	indexer indexer.Indexer,
	config *configfx.Config,
) *Lifecycle {
	return &Lifecycle{
		server:  srv,
		indexer: indexer,
		config:  config,
	}
}

// Start initializes the MCP server and handles pre-indexing
func (m *Lifecycle) Start(ctx context.Context) error {
	// Pre-index project if specified
	if m.config.Project != "" {
		if err := m.indexer.IndexProject(m.config.Project); err != nil {
			return fmt.Errorf("pre-index project failed: %w", err)
		}
	}
	return nil
}

// Stop handles graceful shutdown
func (m *Lifecycle) Stop(ctx context.Context) error {
	// MCP server cleanup is handled by the framework
	return nil
}

// Module provides MCP server components
var Module = fx.Module("mcp",
	fx.Provide(
		NewMCPServer,
		NewLifecycle,
	),
)
