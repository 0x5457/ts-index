package mcp

import (
	"context"
	"fmt"
	"log"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/factory"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/lsp"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ServerOptions contains configuration for the MCP server
type ServerOptions struct {
	Project  string // Project path to pre-index
	DB       string // SQLite database path
	EmbedURL string // Embedding API URL
}

// New returns an MCP server exposing search and LSP tools.
// It intentionally excludes index and LSP install commands.
func New() *server.MCPServer {
	return NewWithOptions(ServerOptions{})
}

// Server wraps an MCP server with configuration options
type Server struct {
	opts       ServerOptions
	server     *server.MCPServer
	factory    *factory.ComponentFactory
	components *factory.Components
}

// NewWithOptions returns an MCP server with the specified options.
// If Project is specified, the server will pre-index it on startup.
func NewWithOptions(opts ServerOptions) *server.MCPServer {
	// Set default values
	if opts.EmbedURL == "" {
		opts.EmbedURL = "http://localhost:8000/embed"
	}

	srv := &Server{
		opts: opts,
		server: server.NewMCPServer(
			"ts-index/mcp",
			"0.1.0",
			server.WithToolCapabilities(true),
		),
	}

	// Initialize shared components if DB is configured
	if opts.DB != "" {
		if err := srv.initComponents(); err != nil {
			log.Printf("initialize components failed: %v", err)
		}
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

	// Pre-index project if specified
	if opts.Project != "" {
		log.Printf("pre-index project: %s", opts.Project)
		if err := srv.preIndexProject(); err != nil {
			log.Printf("pre-index failed: %v", err)
		} else {
			log.Printf("pre-index completed")
		}
	}

	return srv.server
}

// initComponents initializes shared components that can be reused across requests
func (srv *Server) initComponents() error {
	if srv.opts.DB == "" {
		return fmt.Errorf("database path must be specified")
	}

	// Create component factory
	srv.factory = factory.NewComponentFactory(factory.ComponentConfig{
		DBPath:   srv.opts.DB,
		EmbedURL: srv.opts.EmbedURL,
	})

	// Create components
	components, err := srv.factory.CreateComponents()
	if err != nil {
		return fmt.Errorf("initialize components failed: %w", err)
	}
	srv.components = components

	return nil
}

// Cleanup releases resources held by the server
func (srv *Server) Cleanup() error {
	if srv.components != nil {
		return srv.components.Cleanup()
	}
	return nil
}

// preIndexProject indexes the configured project using shared components
func (srv *Server) preIndexProject() error {
	if srv.opts.Project == "" {
		return fmt.Errorf("project path must be specified")
	}

	// Ensure components are initialized
	if srv.factory == nil || srv.components == nil {
		return fmt.Errorf("components not initialized")
	}

	idx := srv.factory.CreateIndexer(srv.components)
	return idx.IndexProject(srv.opts.Project)
}

// Tool definitions
func newSemanticSearchTool() mcp.Tool {
	return mcp.NewTool(
		"semantic_search",
		mcp.WithDescription("Semantic code search by natural language query"),
		mcp.WithString("query", mcp.Description("Natural language query"), mcp.Required()),
		mcp.WithString("db", mcp.Description("SQLite DB path")),
		mcp.WithString("embed_url", mcp.Description("Embedding API URL")),
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
		mcp.WithString("db", mcp.Description("SQLite DB path")),
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

	// Check if we should use shared components or create new ones for custom config
	dbPath := req.GetString("db", srv.opts.DB)
	embURL := req.GetString("embed_url", srv.opts.EmbedURL)

	// If using custom config, create temporary components
	if dbPath != srv.opts.DB || embURL != srv.opts.EmbedURL {
		return srv.handleSemanticSearchWithCustomConfig(ctx, query, dbPath, embURL, project, topK)
	}

	// Use shared components for default config
	if srv.components == nil || srv.components.Searcher == nil {
		return mcp.NewToolResultError("search service not initialized"), nil
	}

	// Index project if specified and different from default
	if project != "" && project != srv.opts.Project {
		idx := srv.factory.CreateIndexer(srv.components)
		if err := idx.IndexProject(project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("index project failed: %v", err)), nil
		}
	}

	hits, err := srv.components.Searcher.Search(ctx, query, topK)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(hits), nil
}

// handleSemanticSearchWithCustomConfig handles semantic search with custom db/embed URL
func (srv *Server) handleSemanticSearchWithCustomConfig(
	ctx context.Context,
	query, dbPath, embURL, project string,
	topK int,
) (*mcp.CallToolResult, error) {
	if dbPath == "" {
		return mcp.NewToolResultError(
			"database path must be specified (through parameters or server configuration)",
		), nil
	}

	p := tsparser.New()
	emb := embeddings.NewApi(embURL)

	sym, err := sqlite.New(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open SQLite failed: %v", err)), nil
	}

	vecStore, err := sqlvec.New(dbPath, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open vector store failed: %v", err)), nil
	}
	defer vecStore.Close() //nolint:errcheck

	idx := pipeline.New(p, emb, sym, vecStore, pipeline.Options{})
	if project != "" {
		if err := idx.IndexProject(project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("index project failed: %v", err)), nil
		}
	}

	svc := &search.Service{Embedder: emb, Vector: vecStore}
	hits, err := svc.Search(ctx, query, topK)
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
	dbPath := req.GetString("db", srv.opts.DB)

	// If using custom DB, create temporary components
	if dbPath != srv.opts.DB {
		return srv.handleSymbolSearchWithCustomConfig(name, dbPath)
	}

	// Use shared components for default config
	if srv.components == nil {
		return mcp.NewToolResultError("components not initialized"), nil
	}

	// Create minimal indexer just for symbol search
	emb := embeddings.NewLocal(1) // Dummy embedder for symbol search
	idx := pipeline.New(
		srv.components.Parser,
		emb,
		srv.components.SymStore,
		srv.components.VecStore,
		pipeline.Options{},
	)
	res, err := idx.SearchSymbol(name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(res), nil
}

// handleSymbolSearchWithCustomConfig handles symbol search with custom db
func (srv *Server) handleSymbolSearchWithCustomConfig(
	name, dbPath string,
) (*mcp.CallToolResult, error) {
	if dbPath == "" {
		return mcp.NewToolResultError(
			"database path must be specified (through parameters or server configuration)",
		), nil
	}

	p := tsparser.New()
	emb := embeddings.NewLocal(1)
	sym, err := sqlite.New(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open SQLite failed: %v", err)), nil
	}

	vecStore, err := sqlvec.New(dbPath, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open vector store failed: %v", err)), nil
	}
	defer vecStore.Close() //nolint:errcheck

	idx := pipeline.New(p, emb, sym, vecStore, pipeline.Options{})
	res, err := idx.SearchSymbol(name)
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
	project := req.GetString("project", srv.opts.Project)
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
