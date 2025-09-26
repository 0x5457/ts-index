package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/0x5457/ts-index/internal/models"
)

// LSPTools provides high-level tools for LLMs to interact with TypeScript code via LSP
type LSPTools struct {
	tsFeatures *TypeScriptFeatures
}

// NewLSPTools creates a new LSP tools instance
func NewLSPTools() *LSPTools {
	return &LSPTools{
		tsFeatures: NewTypeScriptFeatures(),
	}
}

// AnalyzeSymbolRequest represents a request to analyze a symbol
type AnalyzeSymbolRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FilePath      string `json:"file_path"`
	Line          int    `json:"line"`          // 0-based
	Character     int    `json:"character"`     // 0-based
	IncludeHover  bool   `json:"include_hover"`
	IncludeRefs   bool   `json:"include_refs"`
	IncludeDefs   bool   `json:"include_defs"`
}

// AnalyzeSymbolResponse represents the response of symbol analysis
type AnalyzeSymbolResponse struct {
	Symbol      *models.Symbol          `json:"symbol,omitempty"`
	Hover       *models.LSPHoverInfo    `json:"hover,omitempty"`
	Definitions []models.LSPLocation    `json:"definitions,omitempty"`
	References  []models.LSPLocation    `json:"references,omitempty"`
	Error       string                  `json:"error,omitempty"`
}

// GetCompletionRequest represents a request to get completions
type GetCompletionRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FilePath      string `json:"file_path"`
	Line          int    `json:"line"`      // 0-based
	Character     int    `json:"character"` // 0-based
	MaxResults    int    `json:"max_results"`
}

// GetCompletionResponse represents the response of completion request
type GetCompletionResponse struct {
	Items []models.LSPCompletionItem `json:"items"`
	Error string                     `json:"error,omitempty"`
}

// SearchSymbolsRequest represents a request to search symbols
type SearchSymbolsRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Query         string `json:"query"`
	MaxResults    int    `json:"max_results"`
}

// SearchSymbolsResponse represents the response of symbol search
type SearchSymbolsResponse struct {
	Symbols []models.LSPSymbolInfo `json:"symbols"`
	Error   string                 `json:"error,omitempty"`
}

// GetDocumentSymbolsRequest represents a request to get document symbols
type GetDocumentSymbolsRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FilePath      string `json:"file_path"`
}

// GetDocumentSymbolsResponse represents the response of document symbols request
type GetDocumentSymbolsResponse struct {
	Symbols []models.LSPSymbolInfo `json:"symbols"`
	Error   string                 `json:"error,omitempty"`
}

// AnalyzeSymbol analyzes a symbol at a specific position in a TypeScript file
func (t *LSPTools) AnalyzeSymbol(ctx context.Context, req AnalyzeSymbolRequest) AnalyzeSymbolResponse {
	// Validate input
	if req.WorkspaceRoot == "" || req.FilePath == "" {
		return AnalyzeSymbolResponse{
			Error: "workspace_root and file_path are required",
		}
	}

	// Make workspace path absolute
	absWorkspace, err := filepath.Abs(req.WorkspaceRoot)
	if err != nil {
		return AnalyzeSymbolResponse{Error: fmt.Sprintf("failed to get absolute workspace path: %v", err)}
	}

	// Make file path absolute
	if !filepath.IsAbs(req.FilePath) {
		req.FilePath = filepath.Join(absWorkspace, req.FilePath)
	}

	var response AnalyzeSymbolResponse

	// Get hover information if requested
	if req.IncludeHover {
		hover, err := t.tsFeatures.GetHoverInfo(ctx, absWorkspace, req.FilePath, req.Line, req.Character)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get hover info: %v", err)
			return response
		}
		if hover != nil {
			response.Hover = &models.LSPHoverInfo{
				Contents: extractHoverContents(hover.Contents),
				Range:    convertRange(hover.Range),
			}
		}
	}

	// Get definitions if requested
	if req.IncludeDefs {
		definitions, err := t.tsFeatures.GetDefinition(ctx, absWorkspace, req.FilePath, req.Line, req.Character)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get definitions: %v", err)
			return response
		}
		response.Definitions = convertLocations(definitions)
	}

	// Get references if requested
	if req.IncludeRefs {
		references, err := t.tsFeatures.GetReferences(ctx, absWorkspace, req.FilePath, req.Line, req.Character)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get references: %v", err)
			return response
		}
		response.References = convertLocations(references)
	}

	return response
}

// GetCompletion gets completion items at a specific position in a TypeScript file
func (t *LSPTools) GetCompletion(ctx context.Context, req GetCompletionRequest) GetCompletionResponse {
	// Validate input
	if req.WorkspaceRoot == "" || req.FilePath == "" {
		return GetCompletionResponse{
			Error: "workspace_root and file_path are required",
		}
	}

	// Make workspace path absolute
	absWorkspace, err := filepath.Abs(req.WorkspaceRoot)
	if err != nil {
		return GetCompletionResponse{Error: fmt.Sprintf("failed to get absolute workspace path: %v", err)}
	}

	// Make file path absolute
	if !filepath.IsAbs(req.FilePath) {
		req.FilePath = filepath.Join(absWorkspace, req.FilePath)
	}

	// Set default max results
	if req.MaxResults <= 0 {
		req.MaxResults = 20
	}

	completion, compErr := t.tsFeatures.GetCompletion(ctx, absWorkspace, req.FilePath, req.Line, req.Character)
	if compErr != nil {
		return GetCompletionResponse{
			Error: fmt.Sprintf("failed to get completion: %v", compErr),
		}
	}

	items := make([]models.LSPCompletionItem, 0, len(completion.Items))
	for i, item := range completion.Items {
		if i >= req.MaxResults {
			break
		}
		
		items = append(items, models.LSPCompletionItem{
			Label:      item.Label,
			Kind:       (*int)(item.Kind),
			Detail:     getStringPtr(item.Detail),
			InsertText: getStringPtr(item.InsertText),
		})
	}

	return GetCompletionResponse{Items: items}
}

// SearchSymbols searches for symbols in the workspace
func (t *LSPTools) SearchSymbols(ctx context.Context, req SearchSymbolsRequest) SearchSymbolsResponse {
	// Validate input
	if req.WorkspaceRoot == "" || req.Query == "" {
		return SearchSymbolsResponse{
			Error: "workspace_root and query are required",
		}
	}

	// Set default max results
	if req.MaxResults <= 0 {
		req.MaxResults = 50
	}

	symbols, err := t.tsFeatures.GetWorkspaceSymbols(ctx, req.WorkspaceRoot, req.Query)
	if err != nil {
		return SearchSymbolsResponse{
			Error: fmt.Sprintf("failed to search symbols: %v", err),
		}
	}

	result := make([]models.LSPSymbolInfo, 0, len(symbols))
	for i, symbol := range symbols {
		if i >= req.MaxResults {
			break
		}
		
		result = append(result, models.LSPSymbolInfo{
			Name:          symbol.Name,
			Kind:          int(symbol.Kind),
			Location:      convertLocation(symbol.Location),
			ContainerName: getStringPtr(symbol.ContainerName),
		})
	}

	return SearchSymbolsResponse{Symbols: result}
}

// GetDocumentSymbols gets symbols for a specific document
func (t *LSPTools) GetDocumentSymbols(ctx context.Context, req GetDocumentSymbolsRequest) GetDocumentSymbolsResponse {
	// Validate input
	if req.WorkspaceRoot == "" || req.FilePath == "" {
		return GetDocumentSymbolsResponse{
			Error: "workspace_root and file_path are required",
		}
	}

	// Make file path absolute
	if !filepath.IsAbs(req.FilePath) {
		req.FilePath = filepath.Join(req.WorkspaceRoot, req.FilePath)
	}

	symbols, err := t.tsFeatures.GetDocumentSymbols(ctx, req.WorkspaceRoot, req.FilePath)
	if err != nil {
		return GetDocumentSymbolsResponse{
			Error: fmt.Sprintf("failed to get document symbols: %v", err),
		}
	}

	result := make([]models.LSPSymbolInfo, len(symbols))
	for i, symbol := range symbols {
		result[i] = models.LSPSymbolInfo{
			Name:          symbol.Name,
			Kind:          int(symbol.Kind),
			Location:      convertLocation(symbol.Location),
			ContainerName: getStringPtr(symbol.ContainerName),
		}
	}

	return GetDocumentSymbolsResponse{Symbols: result}
}

// Cleanup shuts down all language servers
func (t *LSPTools) Cleanup() error {
	return t.tsFeatures.Cleanup()
}

// Helper functions

func extractHoverContents(contents json.RawMessage) string {
	// Try to extract string content from the hover contents
	var str string
	if err := json.Unmarshal(contents, &str); err == nil {
		return str
	}

	// Try to extract from MarkupContent
	var markup struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}
	if err := json.Unmarshal(contents, &markup); err == nil {
		return markup.Value
	}

	// Try to extract from array of strings
	var strs []string
	if err := json.Unmarshal(contents, &strs); err == nil {
		return strings.Join(strs, "\n")
	}

	// Fallback: return raw JSON
	return string(contents)
}

func convertRange(r *Range) *models.Range {
	if r == nil {
		return nil
	}
	return &models.Range{
		Start: models.Position{Line: r.Start.Line, Character: r.Start.Character},
		End:   models.Position{Line: r.End.Line, Character: r.End.Character},
	}
}

func convertLocation(l Location) models.LSPLocation {
	return models.LSPLocation{
		URI: l.URI,
		Range: models.Range{
			Start: models.Position{Line: l.Range.Start.Line, Character: l.Range.Start.Character},
			End:   models.Position{Line: l.Range.End.Line, Character: l.Range.End.Character},
		},
	}
}

func convertLocations(locations []Location) []models.LSPLocation {
	result := make([]models.LSPLocation, len(locations))
	for i, loc := range locations {
		result[i] = convertLocation(loc)
	}
	return result
}

func getStringPtr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}