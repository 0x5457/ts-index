package mcp

import (
	"context"
	"fmt"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/lsp"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// New returns an MCP server exposing search and LSP tools.
// It intentionally excludes index and LSP install commands.
func New() *server.MCPServer {
	s := server.NewMCPServer(
		"ts-index/mcp",
		"0.1.0",
		server.WithToolCapabilities(true),
	)

	// Search tools
	s.AddTool(newSemanticSearchTool(), handleSemanticSearch)
	s.AddTool(newSymbolSearchTool(), handleSymbolSearch)

	// LSP tools
	s.AddTool(newLSPInfoTool(), handleLSPInfo)
	s.AddTool(newLSPAnalyzeTool(), handleLSPAnalyze)
	s.AddTool(newLSPCompletionTool(), handleLSPCompletion)
	s.AddTool(newLSPSymbolsTool(), handleLSPSymbols)
	s.AddTool(newLSPListTool(), handleLSPList)
	s.AddTool(newLSPHealthTool(), handleLSPHealth)

	return s
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
		mcp.WithString("project", mcp.Description("Workspace root"), mcp.Required()),
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
func handleSemanticSearch(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	query, err := req.RequireString("query")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbPath := req.GetString("db", "")
	embURL := req.GetString("embed_url", "http://localhost:8000/embed")
	project := req.GetString("project", "")
	topK := req.GetInt("top_k", 5)

	p := tsparser.New()
	emb := embeddings.NewApi(embURL)

	if dbPath == "" {
		return mcp.NewToolResultError("db path is required"), nil
	}

	sym, err := sqlite.New(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open sqlite: %v", err)), nil
	}
	vecStore, err := sqlvec.New(dbPath, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open vec store: %v", err)), nil
	}
	idx := pipeline.New(p, emb, sym, vecStore, pipeline.Options{})
	if project != "" {
		if err := idx.IndexProject(project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("index project: %v", err)), nil
		}
	}

	svc := &search.Service{Embedder: emb, Vector: vecStore}
	hits, err := svc.Search(ctx, query, topK)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(hits), nil
}

func handleSymbolSearch(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := req.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	dbPath := req.GetString("db", "")
	if dbPath == "" {
		return mcp.NewToolResultError("db path is required"), nil
	}

	// Re-use pipeline.Indexer for symbol search only
	p := tsparser.New()
	emb := embeddings.NewLocal(1)
	sym, err := sqlite.New(dbPath)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open sqlite: %v", err)), nil
	}
	vecStore, err := sqlvec.New(dbPath, 0)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("open vec store: %v", err)), nil
	}
	idx := pipeline.New(p, emb, sym, vecStore, pipeline.Options{})
	res, err := idx.SearchSymbol(name)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultStructuredOnly(res), nil
}

func handleLSPInfo(_ context.Context, _ mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleLSPAnalyze(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleLSPCompletion(
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

func handleLSPSymbols(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleLSPList(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

func handleLSPHealth(_ context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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
