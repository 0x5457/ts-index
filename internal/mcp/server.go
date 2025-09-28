package mcp

import (
	"context"

	"github.com/0x5457/ts-index/internal/astgrep"
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
	config        ServerConfig    // Server configuration
}

// New returns an MCP server with the given services and configuration.
func New(
	searchService *search.Service,
	indexer indexer.Indexer,
	config ServerConfig,
) *server.MCPServer {
	srv := &Server{
		searchService: searchService,
		indexer:       indexer,
		config:        config,
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
	srv.server.AddTool(newLSPAnalyzeTool(), srv.handleLSPAnalyze)
	srv.server.AddTool(newLSPCompletionTool(), srv.handleLSPCompletion)
	srv.server.AddTool(newLSPSymbolsTool(), srv.handleLSPSymbols)
	srv.server.AddTool(newLSPImplementationTool(), srv.handleLSPImplementation)
	srv.server.AddTool(newLSPTypeDefinitionTool(), srv.handleLSPTypeDefinition)
	srv.server.AddTool(newLSPDeclarationTool(), srv.handleLSPDeclaration)

	// AST-grep tools
	srv.server.AddTool(newAstGrepSearchTool(), srv.handleAstGrepSearch)
	srv.server.AddTool(newAstGrepRuleTool(), srv.handleAstGrepRule)
	srv.server.AddTool(newAstGrepTestTool(), srv.handleAstGrepTest)
	srv.server.AddTool(newAstGrepSyntaxTreeTool(), srv.handleAstGrepSyntaxTree)

	// File tools
	srv.server.AddTool(newReadFileTool(), srv.handleReadFile)

	return srv.server
}

// Tool definitions
func newSemanticSearchTool() mcp.Tool {
	return mcp.NewTool(
		"semantic_search",
		mcp.WithDescription("Semantic code search by natural language query"),
		mcp.WithString("query", mcp.Description("Natural language query"), mcp.Required()),
		mcp.WithNumber("top_k", mcp.Description("Top K results"), mcp.DefaultNumber(5)),
	)
}

func newSymbolSearchTool() mcp.Tool {
	return mcp.NewTool(
		"symbol_search",
		mcp.WithDescription("Exact symbol name search in symbol store"),
		mcp.WithString("name", mcp.Description("Symbol name"), mcp.Required()),
	)
}

func newLSPAnalyzeTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_analyze",
		mcp.WithDescription("Analyze symbol at position using LSP"),
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
		mcp.WithString("query", mcp.Description("Symbol query"), mcp.Required()),
		mcp.WithNumber("max_results", mcp.Description("Max results"), mcp.DefaultNumber(50)),
	)
}

func newLSPImplementationTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_implementation",
		mcp.WithDescription("Find implementations of symbol at position"),
		mcp.WithString("file", mcp.Description("File path"), mcp.Required()),
		mcp.WithNumber("line", mcp.Description("0-based line"), mcp.Required()),
		mcp.WithNumber("character", mcp.Description("0-based character"), mcp.Required()),
	)
}

func newLSPTypeDefinitionTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_type_definition",
		mcp.WithDescription("Find type definitions of symbol at position"),
		mcp.WithString("file", mcp.Description("File path"), mcp.Required()),
		mcp.WithNumber("line", mcp.Description("0-based line"), mcp.Required()),
		mcp.WithNumber("character", mcp.Description("0-based character"), mcp.Required()),
	)
}

func newLSPDeclarationTool() mcp.Tool {
	return mcp.NewTool(
		"lsp_declaration",
		mcp.WithDescription("Find declarations of symbol at position"),
		mcp.WithString("file", mcp.Description("File path"), mcp.Required()),
		mcp.WithNumber("line", mcp.Description("0-based line"), mcp.Required()),
		mcp.WithNumber("character", mcp.Description("0-based character"), mcp.Required()),
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

	// Use default search service
	if srv.searchService == nil {
		return mcp.NewToolResultError("search service not initialized"), nil
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

func (srv *Server) handleLSPAnalyze(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
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

func (srv *Server) handleReadFile(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	filePath, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	startLine := req.GetInt("start_line", 0)
	endLine := req.GetInt("end_line", 0)

	clientTools := lsp.NewClientTools()
	defer func() { _ = clientTools.Cleanup() }()

	result := clientTools.ReadFile(ctx, lsp.ReadFileRequest{
		FilePath:  filePath,
		StartLine: startLine,
		EndLine:   endLine,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleLSPCompletion(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
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
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
		), nil
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

// handleLSPGoto is a generic handler for goto operations
func (srv *Server) handleLSPGoto(
	ctx context.Context,
	req mcp.CallToolRequest,
	gotoFunc func(*lsp.ClientTools, context.Context, lsp.GotoRequest) lsp.GotoResponse,
) (*mcp.CallToolResult, error) {
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
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

	clientTools := lsp.NewClientTools()
	defer func() { _ = clientTools.Cleanup() }()
	result := gotoFunc(clientTools, ctx, lsp.GotoRequest{
		WorkspaceRoot: project,
		FilePath:      file,
		Line:          line,
		Character:     ch,
	})
	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleLSPImplementation(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return srv.handleLSPGoto(ctx, req, (*lsp.ClientTools).GotoImplementation)
}

func (srv *Server) handleLSPTypeDefinition(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return srv.handleLSPGoto(ctx, req, (*lsp.ClientTools).GotoTypeDefinition)
}

func (srv *Server) handleLSPDeclaration(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	return srv.handleLSPGoto(ctx, req, (*lsp.ClientTools).GotoDeclaration)
}

// AST-grep tool definitions
func newAstGrepSearchTool() mcp.Tool {
	return mcp.NewTool(
		"ast_grep_search",
		mcp.WithDescription("Search code patterns using ast-grep"),
		mcp.WithString("pattern", mcp.Description("AST pattern to search for"), mcp.Required()),
		mcp.WithString(
			"language",
			mcp.Description("Programming language (typescript, javascript, python, etc.)"),
			mcp.DefaultString("typescript"),
		),
		mcp.WithNumber(
			"max_results",
			mcp.Description("Maximum number of results"),
			mcp.DefaultNumber(50),
		),
		mcp.WithNumber(
			"context",
			mcp.Description("Number of context lines to include"),
			mcp.DefaultNumber(0),
		),
	)
}

func newAstGrepRuleTool() mcp.Tool {
	return mcp.NewTool(
		"ast_grep_rule",
		mcp.WithDescription("Search using ast-grep YAML rule"),
		mcp.WithString("rule", mcp.Description("YAML rule content"), mcp.Required()),
		mcp.WithNumber(
			"max_results",
			mcp.Description("Maximum number of results"),
			mcp.DefaultNumber(50),
		),
	)
}

func newAstGrepTestTool() mcp.Tool {
	return mcp.NewTool(
		"ast_grep_test",
		mcp.WithDescription("Test ast-grep rule against code snippet"),
		mcp.WithString("rule", mcp.Description("YAML rule content"), mcp.Required()),
		mcp.WithString("code", mcp.Description("Code snippet to test"), mcp.Required()),
		mcp.WithString(
			"language",
			mcp.Description("Programming language"),
			mcp.DefaultString("typescript"),
		),
	)
}

func newAstGrepSyntaxTreeTool() mcp.Tool {
	return mcp.NewTool(
		"ast_grep_syntax_tree",
		mcp.WithDescription("Dump syntax tree of code snippet"),
		mcp.WithString("code", mcp.Description("Code snippet to analyze"), mcp.Required()),
		mcp.WithString(
			"language",
			mcp.Description("Programming language"),
			mcp.DefaultString("typescript"),
		),
	)
}

func newReadFileTool() mcp.Tool {
	return mcp.NewTool(
		"read_file",
		mcp.WithDescription("Read file content with optional line range"),
		mcp.WithString("file_path", mcp.Description("Path to the file to read"), mcp.Required()),
		mcp.WithNumber(
			"start_line",
			mcp.Description("Start line (1-based, optional)"),
			mcp.DefaultNumber(0),
		),
		mcp.WithNumber(
			"end_line",
			mcp.Description("End line (1-based, optional)"),
			mcp.DefaultNumber(0),
		),
	)
}

// AST-grep handlers
func (srv *Server) handleAstGrepSearch(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
		), nil
	}

	pattern, err := req.RequireString("pattern")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	language := req.GetString("language", "typescript")
	maxResults := req.GetInt("max_results", 50)
	context := req.GetInt("context", 0)

	client := astgrep.NewClient()
	result := client.Search(ctx, astgrep.SearchRequest{
		Pattern:        pattern,
		Language:       language,
		ProjectPath:    project,
		MaxResults:     maxResults,
		IncludeContext: context,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleAstGrepRule(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
		), nil
	}

	rule, err := req.RequireString("rule")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	maxResults := req.GetInt("max_results", 50)

	client := astgrep.NewClient()
	result := client.SearchByRule(ctx, astgrep.RuleSearchRequest{
		Rule:        rule,
		ProjectPath: project,
		MaxResults:  maxResults,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleAstGrepTest(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	rule, err := req.RequireString("rule")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	language := req.GetString("language", "typescript")

	client := astgrep.NewClient()
	result := client.TestRule(ctx, astgrep.TestRuleRequest{
		Rule:     rule,
		Code:     code,
		Language: language,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}

func (srv *Server) handleAstGrepSyntaxTree(
	ctx context.Context,
	req mcp.CallToolRequest,
) (*mcp.CallToolResult, error) {
	code, err := req.RequireString("code")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	language := req.GetString("language", "typescript")

	client := astgrep.NewClient()
	result := client.DumpSyntaxTree(ctx, astgrep.SyntaxTreeRequest{
		Code:     code,
		Language: language,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}
