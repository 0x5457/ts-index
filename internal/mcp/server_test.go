package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	server := New(nil, nil, ServerConfig{}) // nil services for basic functionality test
	assert.NotNil(t, server)
}

func TestToolDefinitions(t *testing.T) {
	tests := []struct {
		name     string
		toolFunc func() mcp.Tool
		toolName string
	}{
		{"semantic_search", newSemanticSearchTool, "semantic_search"},
		{"symbol_search", newSymbolSearchTool, "symbol_search"},
		{"lsp_analyze", newLSPAnalyzeTool, "lsp_analyze"},
		{"lsp_completion", newLSPCompletionTool, "lsp_completion"},
		{"lsp_symbols", newLSPSymbolsTool, "lsp_symbols"},
		{"lsp_implementation", newLSPImplementationTool, "lsp_implementation"},
		{"lsp_type_definition", newLSPTypeDefinitionTool, "lsp_type_definition"},
		{"lsp_declaration", newLSPDeclarationTool, "lsp_declaration"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool := tt.toolFunc()
			assert.Equal(t, tt.toolName, tool.Name)
			assert.NotEmpty(t, tool.Description)
		})
	}
}

func TestSemanticSearchTool(t *testing.T) {
	tool := newSemanticSearchTool()
	assert.Equal(t, "semantic_search", tool.Name)
	assert.Contains(t, tool.Description, "Semantic code search")

	// check required params
	assert.Contains(t, tool.InputSchema.Properties, "query")
	queryProp := tool.InputSchema.Properties["query"].(map[string]interface{})
	assert.Equal(t, "string", queryProp["type"])
}

func TestSymbolSearchTool(t *testing.T) {
	tool := newSymbolSearchTool()
	assert.Equal(t, "symbol_search", tool.Name)
	assert.Contains(t, tool.Description, "Exact symbol name search")

	// check required params
	assert.Contains(t, tool.InputSchema.Properties, "name")
	nameProp := tool.InputSchema.Properties["name"].(map[string]interface{})
	assert.Equal(t, "string", nameProp["type"])
}

func TestLSPAnalyzeTool(t *testing.T) {
	tool := newLSPAnalyzeTool()
	assert.Equal(t, "lsp_analyze", tool.Name)
	assert.Contains(t, tool.Description, "Analyze symbol at position")

	// check required params
	requiredParams := []string{"file", "line", "character"}
	for _, param := range requiredParams {
		assert.Contains(t, tool.InputSchema.Properties, param)
	}

	// check optional params
	optionalParams := []string{"defs", "hover", "refs"}
	for _, param := range optionalParams {
		assert.Contains(t, tool.InputSchema.Properties, param)
	}

	// project parameter should no longer exist
	assert.NotContains(t, tool.InputSchema.Properties, "project")
}

func TestHandleSemanticSearchError(t *testing.T) {
	ctx := context.Background()
	srv := &Server{searchService: nil, indexer: nil} // nil components to test error handling

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "semantic_search",
			Arguments: map[string]any{},
		},
	}

	result, err := srv.handleSemanticSearch(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleSymbolSearchError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "symbol_search",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleSymbolSearch(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPAnalyzeError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_analyze",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPAnalyze(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPCompletionError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_completion",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPCompletion(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPSymbolsError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_symbols",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPSymbols(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPImplementationError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_implementation",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPImplementation(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPTypeDefinitionError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_type_definition",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPTypeDefinition(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPDeclarationError(t *testing.T) {
	ctx := context.Background()

	// test missing required params
	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_declaration",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{searchService: nil, indexer: nil}
	result, err := srv.handleLSPDeclaration(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}
