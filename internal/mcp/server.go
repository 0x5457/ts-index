package mcp

import (
	"context"
	"fmt"

	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/lsp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps an MCP server with direct interface dependencies
type Server struct {
	server        *server.MCPServer
	searchService *search.Service // Search service (can be nil)
	indexer       indexer.Indexer // Indexer (can be nil)
}

// New returns an MCP server with the given services.
func New(searchService *search.Service, indexer indexer.Indexer) *server.MCPServer {
	srv := &Server{
		searchService: searchService,
		indexer:       indexer,
		server: server.NewMCPServer(
			"ts-index/mcp",
			"0.1.0",
			server.WithToolCapabilities(true),
		),
	}

	// Search tools
	srv.server.AddTool(newSemanticSearchTool(), srv.handleSemanticSearch)
	srv.server.AddTool(newSymbolSearchTool(), srv.handleSymbolSearch)

	// LSP tools
	srv.server.AddTool(newLSPInfoTool(), srv.handleLSPInfo)
	srv.server.AddTool(newLSPAnalyzeTool(), srv.handleLSPAnalyze)
	srv.server.AddTool(newLSPCompletionTool(), srv.handleLSPCompletion)
	srv.server.AddTool(newLSPSymbolsTool(), srv.handleLSPSymbols)
	srv.server.AddTool(newLSPListTool(), srv.handleLSPList)
	srv.server.AddTool(newLSPHealthTool(), srv.handleLSPHealth)

	return srv.server
}

// Tool definitions
func newSemanticSearchTool() mcp.Tool {
	return mcp.NewTool(
		"semantic_search",
		mcp.WithDescription("Semantic code search by natural language query"),
		mcp.WithString("query", mcp.Description("Natural language query"), mcp.Required()),
		mcp.WithString(
			"db",
			mcp.Description("SQLite DB path (optional, uses server default if not provided)"),
		),
		mcp.WithString(
			"embed_url",
			mcp.Description("Embedding API URL (optional, uses server default if not provided)"),
		),
		mcp.WithNumber("top_k", mcp.Description("Top K results"), mcp.DefaultNumber(5)),
		mcp.WithString(
			"project",
			mcp.Description("Optional project path to index into memory before searching"),
		),
	)
}

func newSymbolSearchTool() mcp.Tool {
	return mcp.NewTool(
		"symbol_search",
		mcp.WithDescription("Exact symbol name search in symbol store"),
		mcp.WithString("name", mcp.Description("Symbol name"), mcp.Required()),
		mcp.WithString(
			"db",
			mcp.Description("SQLite DB path (optional, uses server default if not provided)"),
		),
	)
}

func newLSPInfoTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_info",
		mcp.WithDescription("Show LSP adapters and running servers"),
	)
}

func newLSPAnalyzeTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_analyze",
		mcp.WithDescription("Analyze symbol at position using LSP"),
		mcp.WithString(
			"project",
			mcp.Description("Workspace root (optional, fallback to server config)"),
		),
		mcp.WithString("file", mcp.Description("File path"), mcp.Required()),
		mcp.WithNumber("line", mcp.Description("0-based line"), mcp.Required()),
		mcp.WithNumber("character", mcp.Description("0-based character"), mcp.Required()),
		mcp.WithBoolean("hover", mcp.Description("Include hover"), mcp.DefaultBool(true)),
		mcp.WithBoolean("refs", mcp.Description("Include references"), mcp.DefaultBool(false)),
		mcp.WithBoolean("defs", mcp.Description("Include definitions"), mcp.DefaultBool(true)),
	)
}

func newLSPCompletionTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_completion",
		mcp.WithDescription("Get completion items at position"),
		mcp.WithString("project", mcp.Description("Workspace root"), mcp.Required()),
		mcp.WithString("file", mcp.Description("File path"), mcp.Required()),
		mcp.WithNumber("line", mcp.Description("0-based line"), mcp.Required()),
		mcp.WithNumber("character", mcp.Description("0-based character"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Max results"), mcp.DefaultNumber(20)),
	)
}

func newLSPSymbolsTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_symbols",
		mcp.WithDescription("Search workspace symbols via LSP"),
		mcp.WithString("project", mcp.Description("Workspace root"), mcp.Required()),
		mcp.WithString("query", mcp.Description("Symbol query"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Max results"), mcp.DefaultNumber(50)),
	)
}

func newLSPListTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_list",
		mcp.WithDescription("List locally installed LSP servers"),
		mcp.WithString("dir", mcp.Description("Installation directory override")),
	)
}

func newLSPHealthTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_health",
		mcp.WithDescription("Check LSP health and availability"),
		mcp.WithString("dir", mcp.Description("Installation directory override")),
	)
}

// Handlers
func (srv *Server) handleSemanticSearch(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	topK := req.GetInt("top_k", 5)
	project := req.GetString("project", "")

	// Check for custom config parameters
	dbPath := req.GetString("db", "")
	embURL := req.GetString("embed_url", "")

	if dbPath != "" || embURL != "" {
		return mcp.NewToolResultError(
			"Custom database path and embedding URL are not supported in this server instance. " +
				"Please start server with the desired configuration.",
		), nil
	}

	// Use default search service
	if srv.searchService == nil {
		return mcp.NewToolResultError("search service not initialized"), nil
	}

	// Index project if specified
	if project != "" {
		if srv.indexer == nil {
			return mcp.NewToolResultError("indexer not available for project indexing"), nil
		}
		if err := srv.indexer.IndexProject(project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("index project failed: %v", err)), nil
		}
	}

	hits, err := srv.searchService.Search(ctx, query, topK)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(hits), nil
}

func (srv *Server) handleSymbolSearch(
	_ context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	// Check for custom config parameters
	dbPath := req.GetString("db", "")
	if dbPath != "" {
		return mcp.NewToolResultError(
			"Custom database path is not supported in this server instance. " +
				"Please start server with the desired database configuration.",
		), nil
	}

	// Use default indexer for symbol search
	if srv.indexer == nil {
		return mcp.NewToolResultError("indexer not initialized"), nil
	}

	res, err := srv.indexer.SearchSymbol(name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(res), nil
}

func (srv *Server) handleLSPInfo(
	_ context.Context,
	_ mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	clientTools := lsp.NewClientTools()
	adapters := clientTools.GetAdapterInfo()
	servers := clientTools.GetServerInfo()
	resp := map[string]any{
		"adapters": adapters,
		"servers":  servers,
	}
	_ = clientTools.Cleanup()
	return mcp.NewToolResultStructuredOnly(resp), nil
}

func (srv *Server) handleLSPAnalyze(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	project := req.GetString("project", "")
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified (through parameters or server configuration)",
		), nil
	}
	file, err := req.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ch, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	hover := req.GetBool("hover", true)
	refs := req.GetBool("refs", false)
	defs := req.GetBool("defs", true)

	clientTools := lsp.NewClientTools()
	defer func() { _ = clientTools.Cleanup() }()
	result := clientTools.AnalyzeSymbol(ctx, lsp.AnalyzeSymbolRequest{
		WorkspaceRoot: project,
		FilePath:      file,
		Line:          line,
		Character:     ch,
		IncludeHover:  hover,
		IncludeRefs:   refs,
		IncludeDefs:   defs,
	})
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleLSPCompletion(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	file, err := req.RequireString("file")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	line, err := req.RequireInt("line")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	ch, err := req.RequireInt("character")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	max := req.GetInt("max_results", 20)

	clientTools := lsp.NewClientTools()
	defer func() { _ = clientTools.Cleanup() }()
	result := clientTools.GetCompletion(ctx, lsp.CompletionRequest{
		WorkspaceRoot: project,
		FilePath:      file,
		Line:          line,
		Character:     ch,
		MaxResults:    max,
	})
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleLSPSymbols(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	project, err := req.RequireString("project")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	max := req.GetInt("max_results", 50)

	clientTools := lsp.NewClientTools()
	defer func() { _ = clientTools.Cleanup() }()
	result := clientTools.SearchSymbols(ctx, lsp.SymbolSearchRequest{
		WorkspaceRoot: project,
		Query:         query,
		MaxResults:    max,
	})
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleLSPList(
	_ context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	installDir := req.GetString("dir", "")
	var mgr *lsp.InstallationManager
	if installDir != "" {
		mgr = lsp.NewInstallationManager(installDir)
	} else {
		mgr = lsp.NewInstallationManager("")
	}
	delegate := &lsp.SimpleDelegate{}
	servers, err := mgr.GetInstalledServers(delegate)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(map[string]any{"installed": servers}), nil
}

func (srv *Server) handleLSPHealth(
	_ context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	installDir := req.GetString("dir", "")
	health := map[string]any{}
	health["vtsls_system"] = lsp.IsVTSLSInstalled()
	health["tsls_system"] = lsp.IsTypeScriptLanguageServerInstalled()

	var mgr *lsp.InstallationManager
	if installDir != "" {
		mgr = lsp.NewInstallationManager(installDir)
	} else {
		mgr = lsp.NewInstallationManager("")
	}
	delegate := &lsp.SimpleDelegate{}
	servers, err := mgr.GetInstalledServers(delegate)
	if err == nil {
		health["local"] = servers
	}
	return mcp.NewToolResultStructuredOnly(health), nil
}
