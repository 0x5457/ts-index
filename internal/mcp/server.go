package mcp

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/0x5457/ts-index/internal/astgrep"
	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/lsp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Server wraps an MCP server with direct interface dependencies
type Server struct {
	server         *server.MCPServer
	searchService  *search.Service  // Search service (can be nil)
	indexer        indexer.Indexer  // Indexer (can be nil)
	config         ServerConfig     // Server configuration
	lspClientTools *lsp.ClientTools // Pre-initialized LSP client tools
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

	// Pre-initialize LSP client tools if we have a project configured
	if config.Project != "" {
		srv.initializeLSPClient()
	}

	// Search tools
	srv.server.AddTool(newSemanticSearchTool(), srv.handleSemanticSearch)

	// LSP tools
	srv.server.AddTool(newLSPAnalyzeTool(), srv.handleLSPAnalyze)
	srv.server.AddTool(newLSPSymbolsTool(), srv.handleLSPSymbols)
	srv.server.AddTool(newLSPImplementationTool(), srv.handleLSPImplementation)
	srv.server.AddTool(newLSPTypeDefinitionTool(), srv.handleLSPTypeDefinition)
	srv.server.AddTool(newLSPDeclarationTool(), srv.handleLSPDeclaration)

	// AST-grep tools
	srv.server.AddTool(newAstGrepSearchTool(), srv.handleAstGrepSearch)

	// File tools
	srv.server.AddTool(newReadFileTool(), srv.handleReadFile)

	return srv.server
}

// initializeLSPClient pre-initializes the LSP client to catch errors early
func (srv *Server) initializeLSPClient() {
	fmt.Printf("Initializing LSP client for project: %s\n", srv.config.Project)

	srv.lspClientTools = lsp.NewClientTools()

	// Test LSP connection by trying to create a language server
	ctx := context.Background()

	// Try to get adapter info to validate the setup
	adapters := srv.lspClientTools.GetAdapterInfo()
	if len(adapters) == 0 {
		fmt.Fprintf(os.Stderr, "[LSP WARNING] No LSP adapters available\n")
		return
	}

	// Try to test the server startup with a simple operation
	// We'll test this by checking if we can start a language server
	go func() {
		// Create a test request to warm up the language server
		result := srv.lspClientTools.SearchSymbols(ctx, lsp.SymbolSearchRequest{
			WorkspaceRoot: srv.config.Project,
			Query:         "", // Empty query to test connection
			MaxResults:    1,
		})

		if result.Error != "" {
			fmt.Fprintf(
				os.Stderr,
				"[LSP ERROR] Language server initialization failed: %s\n",
				result.Error,
			)
			fmt.Fprintf(
				os.Stderr,
				"[LSP ERROR] This may cause LSP tools to fail during operation\n",
			)
		} else {
			fmt.Printf("LSP client initialized successfully\n")
		}
	}()
}

// getLSPClientTools returns the pre-initialized LSP client tools or creates new ones as fallback
func (srv *Server) getLSPClientTools() *lsp.ClientTools {
	if srv.lspClientTools != nil {
		return srv.lspClientTools
	}

	// Fallback: create new client tools if pre-initialization failed
	fmt.Fprintf(
		os.Stderr,
		"[LSP WARNING] Using fallback LSP client tools (pre-initialization may have failed)\n",
	)
	return lsp.NewClientTools()
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

	// Wrap the hits array in an object to satisfy MCP protocol expectations
	result := map[string]interface{}{
		"hits":  hits,
		"query": query,
		"total": len(hits),
	}
	return mcp.NewToolResultStructuredOnly(result), nil
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

	// Use pre-initialized client tools or create new ones
	clientTools := srv.getLSPClientTools()
	if clientTools == nil {
		return mcp.NewToolResultError("LSP client not available"), nil
	}

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
	// Use server config project
	project := srv.config.Project
	if project == "" {
		return mcp.NewToolResultError(
			"workspace path must be specified in server configuration",
		), nil
	}

	filePath, err := req.RequireString("file_path")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	startLine := req.GetInt("start_line", 0)
	endLine := req.GetInt("end_line", 0)

	// Use pre-initialized client tools or create new ones
	clientTools := srv.getLSPClientTools()
	if clientTools == nil {
		return mcp.NewToolResultError("LSP client not available"), nil
	}

	result := clientTools.ReadFile(ctx, lsp.ReadFileRequest{
		FilePath:      filePath,
		WorkspaceRoot: project,
		StartLine:     startLine,
		EndLine:       endLine,
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

	// Use pre-initialized client tools or create new ones
	clientTools := srv.getLSPClientTools()
	if clientTools == nil {
		return mcp.NewToolResultError("LSP client not available"), nil
	}

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

	// Use pre-initialized client tools or create new ones
	clientTools := srv.getLSPClientTools()
	if clientTools == nil {
		return mcp.NewToolResultError("LSP client not available"), nil
	}

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
		mcp.WithString(
			"globs",
			mcp.Description(
				"Comma-separated glob patterns for file inclusion/exclusion. Patterns starting with ! are exclusions.",
			),
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

	// Parse globs from comma-separated string
	var globs []string
	if globsStr := req.GetString("globs", ""); globsStr != "" {
		// Split by comma and trim spaces
		for _, glob := range strings.Split(globsStr, ",") {
			if trimmed := strings.TrimSpace(glob); trimmed != "" {
				globs = append(globs, trimmed)
			}
		}
	}

	client := astgrep.NewClient(project)
	result := client.Search(ctx, astgrep.SearchRequest{
		Pattern:        pattern,
		Language:       language,
		MaxResults:     maxResults,
		IncludeContext: context,
		Globs:          globs,
	})

	if result.Error != "" {
		return mcp.NewToolResultError(result.Error), nil
	}

	return mcp.NewToolResultStructuredOnly(result), nil
}
