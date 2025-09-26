package mcp

import (
	"context"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	server := New()
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
		{"lsp_info", newLSPInfoTool, "lsp_info"},
		{"lsp_analyze", newLSPAnalyzeTool, "lsp_analyze"},
		{"lsp_completion", newLSPCompletionTool, "lsp_completion"},
		{"lsp_symbols", newLSPSymbolsTool, "lsp_symbols"},
		{"lsp_list", newLSPListTool, "lsp_list"},
		{"lsp_health", newLSPHealthTool, "lsp_health"},
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
	requiredParams := []string{"project", "file", "line", "character"}
	for _, param := range requiredParams {
		assert.Contains(t, tool.InputSchema.Properties, param)
	}
}

func TestHandleSemanticSearchError(t *testing.T) {
	ctx := context.Background()
	srv := &Server{opts: ServerOptions{}}

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

	srv := &Server{opts: ServerOptions{}}
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

	srv := &Server{opts: ServerOptions{}}
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

	srv := &Server{opts: ServerOptions{}}
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

	srv := &Server{opts: ServerOptions{}}
	result, err := srv.handleLSPSymbols(ctx, req)
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.NotEmpty(t, result.Content) // check error content
}

func TestHandleLSPInfo(t *testing.T) {
	ctx := context.Background()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_info",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{opts: ServerOptions{}}
	result, err := srv.handleLSPInfo(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotNil(t, result.StructuredContent)

	// check return structure
	content := result.StructuredContent.(map[string]any)
	assert.Contains(t, content, "adapters")
	assert.Contains(t, content, "servers")
}

func TestHandleLSPList(t *testing.T) {
	ctx := context.Background()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_list",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{opts: ServerOptions{}}
	result, err := srv.handleLSPList(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotNil(t, result.StructuredContent)

	// check return structure
	content := result.StructuredContent.(map[string]any)
	assert.Contains(t, content, "installed")
}

func TestHandleLSPHealth(t *testing.T) {
	ctx := context.Background()

	req := mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Name:      "lsp_health",
			Arguments: map[string]any{},
		},
	}

	srv := &Server{opts: ServerOptions{}}
	result, err := srv.handleLSPHealth(ctx, req)
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.NotNil(t, result.StructuredContent)

	// check return structure
	content := result.StructuredContent.(map[string]any)
	assert.Contains(t, content, "vtsls_system")
	assert.Contains(t, content, "tsls_system")
}
