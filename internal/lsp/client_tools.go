package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ClientTools provides high-level tools for interacting with language servers
// This is the main interface that applications should use
type ClientTools struct {
	manager *LanguageServerManager
}

// NewClientTools creates a new client tools instance
func NewClientTools() *ClientTools {
	// Create a simple delegate for basic functionality
	delegate := &SimpleDelegate{}
	manager := NewLanguageServerManager(delegate)

	return &ClientTools{
		manager: manager,
	}
}

// AnalyzeSymbolRequest represents a request to analyze a symbol
type AnalyzeSymbolRequest struct {
	WorkspaceRoot          string `json:"workspace_root"`
	FilePath               string `json:"file_path"`
	Line                   int    `json:"line"`      // 0-based
	Character              int    `json:"character"` // 0-based
	IncludeHover           bool   `json:"include_hover"`
	IncludeRefs            bool   `json:"include_refs"`
	IncludeDefs            bool   `json:"include_defs"`
	IncludeImplementations bool   `json:"include_implementations"`
	IncludeTypeDefinitions bool   `json:"include_type_definitions"`
	IncludeDeclarations    bool   `json:"include_declarations"`
}

// GotoRequest represents a generic goto request (implementation/type definition/declaration)
type GotoRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FilePath      string `json:"file_path"`
	Line          int    `json:"line"`      // 0-based
	Character     int    `json:"character"` // 0-based
}

// GotoResponse represents a goto response
type GotoResponse struct {
	Locations []LocationResult `json:"locations"`
	Error     string           `json:"error,omitempty"`
}

// AnalyzeSymbolResponse represents the response of symbol analysis
type AnalyzeSymbolResponse struct {
	Hover           *HoverResult     `json:"hover,omitempty"`
	Definitions     []LocationResult `json:"definitions,omitempty"`
	References      []LocationResult `json:"references,omitempty"`
	Implementations []LocationResult `json:"implementations,omitempty"`
	TypeDefinitions []LocationResult `json:"type_definitions,omitempty"`
	Declarations    []LocationResult `json:"declarations,omitempty"`
	Error           string           `json:"error,omitempty"`
}

// HoverResult represents hover information
type HoverResult struct {
	Contents string `json:"contents"`
	Range    *Range `json:"range,omitempty"`
}

// LocationResult represents a location
type LocationResult struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// CompletionRequest represents a request to get completions
type CompletionRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	FilePath      string `json:"file_path"`
	Line          int    `json:"line"`      // 0-based
	Character     int    `json:"character"` // 0-based
	MaxResults    int    `json:"max_results"`
}

// CompletionResponse represents the response of completion request
type CompletionResponse struct {
	Items []CompletionItemResult `json:"items"`
	Error string                 `json:"error,omitempty"`
}

// CompletionItemResult represents a completion item
type CompletionItemResult struct {
	Label      string `json:"label"`
	Kind       int    `json:"kind,omitempty"`
	Detail     string `json:"detail,omitempty"`
	InsertText string `json:"insert_text,omitempty"`
}

// SymbolSearchRequest represents a request to search symbols
type SymbolSearchRequest struct {
	WorkspaceRoot string `json:"workspace_root"`
	Query         string `json:"query"`
	MaxResults    int    `json:"max_results"`
}

// SymbolSearchResponse represents the response of symbol search
type SymbolSearchResponse struct {
	Symbols []SymbolResult `json:"symbols"`
	Error   string         `json:"error,omitempty"`
}

// SymbolResult represents a symbol
type SymbolResult struct {
	Name          string         `json:"name"`
	Kind          int            `json:"kind"`
	Location      LocationResult `json:"location"`
	ContainerName string         `json:"container_name,omitempty"`
}

// AnalyzeSymbol analyzes a symbol at a specific position
func (ct *ClientTools) AnalyzeSymbol(
	ctx context.Context,
	req AnalyzeSymbolRequest,
) AnalyzeSymbolResponse {
	// Determine language from file extension
	language := getLanguageFromPath(req.FilePath)
	if language == "" {
		return AnalyzeSymbolResponse{Error: "unsupported file type"}
	}

	// Get or create language server
	server, err := ct.manager.GetLanguageServer(ctx, req.WorkspaceRoot, language)
	if err != nil {
		return AnalyzeSymbolResponse{Error: fmt.Sprintf("failed to get language server: %v", err)}
	}

	// Make file path absolute
	absFilePath := req.FilePath
	if !filepath.IsAbs(absFilePath) {
		absRoot, _ := filepath.Abs(req.WorkspaceRoot)
		absFilePath = filepath.Join(absRoot, req.FilePath)
	}

	uri := PathToURI(absFilePath)
	position := Position{Line: req.Line, Character: req.Character}

	// Ensure document is open
	if err := ct.ensureDocumentOpen(ctx, server, uri, absFilePath); err != nil {
		return AnalyzeSymbolResponse{Error: fmt.Sprintf("failed to open document: %v", err)}
	}

	var response AnalyzeSymbolResponse

	// Get hover information if requested
	if req.IncludeHover {
		hover, err := server.Hover(ctx, uri, position)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get hover info: %v", err)
			return response
		}
		if hover != nil {
			response.Hover = &HoverResult{
				Contents: extractHoverContents(hover.Contents),
				Range:    hover.Range,
			}
		}
	}

	// Get definitions if requested
	if req.IncludeDefs {
		definitions, err := server.GotoDefinition(ctx, uri, position)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get definitions: %v", err)
			return response
		}
		response.Definitions = convertLocationsToResults(definitions)
	}

	// Get references if requested
	if req.IncludeRefs {
		references, err := server.FindReferences(ctx, uri, position, true)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get references: %v", err)
			return response
		}
		response.References = convertLocationsToResults(references)
	}

	// Get implementations if requested
	if req.IncludeImplementations {
		implementations, err := server.GotoImplementation(ctx, uri, position)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get implementations: %v", err)
			return response
		}
		response.Implementations = convertLocationsToResults(implementations)
	}

	// Get type definitions if requested
	if req.IncludeTypeDefinitions {
		typeDefinitions, err := server.GotoTypeDefinition(ctx, uri, position)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get type definitions: %v", err)
			return response
		}
		response.TypeDefinitions = convertLocationsToResults(typeDefinitions)
	}

	// Get declarations if requested
	if req.IncludeDeclarations {
		declarations, err := server.GotoDeclaration(ctx, uri, position)
		if err != nil {
			response.Error = fmt.Sprintf("failed to get declarations: %v", err)
			return response
		}
		response.Declarations = convertLocationsToResults(declarations)
	}

	return response
}

// GetCompletion gets completion items at a specific position
func (ct *ClientTools) GetCompletion(
	ctx context.Context,
	req CompletionRequest,
) CompletionResponse {
	// Determine language from file extension
	language := getLanguageFromPath(req.FilePath)
	if language == "" {
		return CompletionResponse{Error: "unsupported file type"}
	}

	// Get or create language server
	server, err := ct.manager.GetLanguageServer(ctx, req.WorkspaceRoot, language)
	if err != nil {
		return CompletionResponse{Error: fmt.Sprintf("failed to get language server: %v", err)}
	}

	// Make file path absolute
	absFilePath := req.FilePath
	if !filepath.IsAbs(absFilePath) {
		absRoot, _ := filepath.Abs(req.WorkspaceRoot)
		absFilePath = filepath.Join(absRoot, req.FilePath)
	}

	uri := PathToURI(absFilePath)
	position := Position{Line: req.Line, Character: req.Character}

	// Ensure document is open
	if err := ct.ensureDocumentOpen(ctx, server, uri, absFilePath); err != nil {
		return CompletionResponse{Error: fmt.Sprintf("failed to open document: %v", err)}
	}

	// Set default max results
	if req.MaxResults <= 0 {
		req.MaxResults = 20
	}

	completion, err := server.Completion(ctx, uri, position)
	if err != nil {
		return CompletionResponse{Error: fmt.Sprintf("failed to get completion: %v", err)}
	}

	items := make([]CompletionItemResult, 0, len(completion.Items))
	for i, item := range completion.Items {
		if i >= req.MaxResults {
			break
		}

		items = append(items, CompletionItemResult{
			Label:      item.Label,
			Kind:       getCompletionKindValue(item.Kind),
			Detail:     getStringValue(item.Detail),
			InsertText: getStringValue(item.InsertText),
		})
	}

	return CompletionResponse{Items: items}
}

// SearchSymbols searches for symbols in the workspace
func (ct *ClientTools) SearchSymbols(
	ctx context.Context,
	req SymbolSearchRequest,
) SymbolSearchResponse {
	// Try TypeScript first (most common case)
	language := "typescript"

	// Get or create language server
	server, err := ct.manager.GetLanguageServer(ctx, req.WorkspaceRoot, language)
	if err != nil {
		return SymbolSearchResponse{Error: fmt.Sprintf("failed to get language server: %v", err)}
	}

	// Set default max results
	if req.MaxResults <= 0 {
		req.MaxResults = 50
	}

	symbols, err := server.WorkspaceSymbols(ctx, req.Query)
	if err != nil {
		return SymbolSearchResponse{Error: fmt.Sprintf("failed to search symbols: %v", err)}
	}

	result := make([]SymbolResult, 0, len(symbols))
	for i, symbol := range symbols {
		if i >= req.MaxResults {
			break
		}

		result = append(result, SymbolResult{
			Name: symbol.Name,
			Kind: int(symbol.Kind),
			Location: LocationResult{
				URI:   symbol.Location.URI,
				Range: symbol.Location.Range,
			},
			ContainerName: getStringValue(symbol.ContainerName),
		})
	}

	return SymbolSearchResponse{Symbols: result}
}

// GotoImplementation finds implementations of the symbol at a specific position
func (ct *ClientTools) GotoImplementation(
	ctx context.Context,
	req GotoRequest,
) GotoResponse {
	return ct.performGoto(ctx, req, "implementation")
}

// GotoTypeDefinition finds type definitions of the symbol at a specific position
func (ct *ClientTools) GotoTypeDefinition(
	ctx context.Context,
	req GotoRequest,
) GotoResponse {
	return ct.performGoto(ctx, req, "typeDefinition")
}

// GotoDeclaration finds declarations of the symbol at a specific position
func (ct *ClientTools) GotoDeclaration(
	ctx context.Context,
	req GotoRequest,
) GotoResponse {
	return ct.performGoto(ctx, req, "declaration")
}

// performGoto is a helper method to perform goto operations
func (ct *ClientTools) performGoto(
	ctx context.Context,
	req GotoRequest,
	gotoType string,
) GotoResponse {
	// Determine language from file extension
	language := getLanguageFromPath(req.FilePath)
	if language == "" {
		return GotoResponse{Error: "unsupported file type"}
	}

	// Get or create language server
	server, err := ct.manager.GetLanguageServer(ctx, req.WorkspaceRoot, language)
	if err != nil {
		return GotoResponse{Error: fmt.Sprintf("failed to get language server: %v", err)}
	}

	// Make file path absolute
	absFilePath := req.FilePath
	if !filepath.IsAbs(absFilePath) {
		absRoot, _ := filepath.Abs(req.WorkspaceRoot)
		absFilePath = filepath.Join(absRoot, req.FilePath)
	}

	uri := PathToURI(absFilePath)
	position := Position{Line: req.Line, Character: req.Character}

	// Ensure document is open
	if err := ct.ensureDocumentOpen(ctx, server, uri, absFilePath); err != nil {
		return GotoResponse{Error: fmt.Sprintf("failed to open document: %v", err)}
	}

	var locations []Location
	var gotoErr error

	switch gotoType {
	case "implementation":
		locations, gotoErr = server.GotoImplementation(ctx, uri, position)
	case "typeDefinition":
		locations, gotoErr = server.GotoTypeDefinition(ctx, uri, position)
	case "declaration":
		locations, gotoErr = server.GotoDeclaration(ctx, uri, position)
	default:
		return GotoResponse{Error: fmt.Sprintf("unknown goto type: %s", gotoType)}
	}

	if gotoErr != nil {
		return GotoResponse{Error: fmt.Sprintf("failed to get %s: %v", gotoType, gotoErr)}
	}

	return GotoResponse{Locations: convertLocationsToResults(locations)}
}

// GetDocumentSymbols gets symbols for a specific document
func (ct *ClientTools) GetDocumentSymbols(
	ctx context.Context,
	workspaceRoot, filePath string,
) ([]SymbolResult, error) {
	// Determine language from file extension
	language := getLanguageFromPath(filePath)
	if language == "" {
		return nil, fmt.Errorf("unsupported file type")
	}

	// Get or create language server
	server, err := ct.manager.GetLanguageServer(ctx, workspaceRoot, language)
	if err != nil {
		return nil, fmt.Errorf("failed to get language server: %v", err)
	}

	// Make file path absolute
	absFilePath := filePath
	if !filepath.IsAbs(absFilePath) {
		absRoot, _ := filepath.Abs(workspaceRoot)
		absFilePath = filepath.Join(absRoot, filePath)
	}

	uri := PathToURI(absFilePath)

	// Ensure document is open
	if err := ct.ensureDocumentOpen(ctx, server, uri, absFilePath); err != nil {
		return nil, fmt.Errorf("failed to open document: %v", err)
	}

	symbols, err := server.DocumentSymbols(ctx, uri)
	if err != nil {
		return nil, err
	}

	result := make([]SymbolResult, len(symbols))
	for i, symbol := range symbols {
		result[i] = SymbolResult{
			Name: symbol.Name,
			Kind: int(symbol.Kind),
			Location: LocationResult{
				URI:   symbol.Location.URI,
				Range: symbol.Location.Range,
			},
			ContainerName: getStringValue(symbol.ContainerName),
		}
	}

	return result, nil
}

// Cleanup shuts down all language servers
func (ct *ClientTools) Cleanup() error {
	return ct.manager.StopAllServers()
}

// GetServerInfo returns information about running servers
func (ct *ClientTools) GetServerInfo() []ServerInfo {
	return ct.manager.GetRunningServers()
}

// GetAdapterInfo returns information about registered adapters
func (ct *ClientTools) GetAdapterInfo() []AdapterInfo {
	return ct.manager.GetRegisteredAdapters()
}

// ReadFileRequest represents a request to read file content
type ReadFileRequest struct {
	FilePath      string `json:"file_path"`
	WorkspaceRoot string `json:"workspace_root,omitempty"` // Project root path
	StartLine     int    `json:"start_line,omitempty"`     // 1-based line number, 0 means from beginning
	EndLine       int    `json:"end_line,omitempty"`       // 1-based line number, 0 means to end
}

// ReadFileResponse represents the response of reading a file
type ReadFileResponse struct {
	Content    string `json:"content"`
	Range      *Range `json:"range,omitempty"` // Range of lines that were read
	TotalLines int    `json:"total_lines"`     // Total lines in the file
	Error      string `json:"error,omitempty"`
}

// ReadFile reads file content, optionally within a specified range
func (ct *ClientTools) ReadFile(
	ctx context.Context,
	req ReadFileRequest,
) ReadFileResponse {
	// Make file path absolute if needed
	absFilePath := req.FilePath
	if !filepath.IsAbs(absFilePath) {
		if req.WorkspaceRoot != "" {
			// If workspace root is provided, make path relative to workspace root
			absWorkspaceRoot, err := filepath.Abs(req.WorkspaceRoot)
			if err != nil {
				return ReadFileResponse{
					Error: fmt.Sprintf("failed to get absolute workspace path: %v", err),
				}
			}
			absFilePath = filepath.Join(absWorkspaceRoot, req.FilePath)
		} else {
			// Fallback to current working directory
			var err error
			absFilePath, err = filepath.Abs(req.FilePath)
			if err != nil {
				return ReadFileResponse{Error: fmt.Sprintf("failed to get absolute path: %v", err)}
			}
		}
	}

	// Read entire file content
	content, err := readFileContent(absFilePath)
	if err != nil {
		return ReadFileResponse{Error: fmt.Sprintf("failed to read file: %v", err)}
	}

	lines := strings.Split(content, "\n")
	totalLines := len(lines)

	// If no range specified, return entire file
	if req.StartLine == 0 && req.EndLine == 0 {
		return ReadFileResponse{
			Content:    content,
			TotalLines: totalLines,
		}
	}

	// Handle range selection
	startIdx := 0
	endIdx := totalLines

	if req.StartLine > 0 {
		startIdx = req.StartLine - 1 // Convert to 0-based
		if startIdx >= totalLines {
			return ReadFileResponse{Error: "start line exceeds file length"}
		}
	}

	if req.EndLine > 0 {
		endIdx = req.EndLine // EndLine is inclusive, so we don't subtract 1
		if endIdx > totalLines {
			endIdx = totalLines
		}
	}

	if startIdx >= endIdx {
		return ReadFileResponse{Error: "invalid range: start line must be less than end line"}
	}

	// Extract the requested range
	selectedLines := lines[startIdx:endIdx]
	selectedContent := strings.Join(selectedLines, "\n")

	// Create range info
	rangeInfo := &Range{
		Start: Position{Line: startIdx, Character: 0},
		End:   Position{Line: endIdx - 1, Character: len(lines[endIdx-1])},
	}

	return ReadFileResponse{
		Content:    selectedContent,
		Range:      rangeInfo,
		TotalLines: totalLines,
	}
}

// Helper functions

func (ct *ClientTools) ensureDocumentOpen(
	ctx context.Context,
	server *LanguageServer,
	uri, filePath string,
) error {
	// Read file content
	content, err := readFileContent(filePath)
	if err != nil {
		return err
	}

	// Open document
	return server.DidOpen(ctx, uri, content)
}

func getLanguageFromPath(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".ts":
		return "typescript"
	case ".tsx":
		return "typescriptreact"
	case ".js":
		return "javascript"
	case ".jsx":
		return "javascriptreact"
	default:
		return ""
	}
}

func convertLocationsToResults(locations []Location) []LocationResult {
	result := make([]LocationResult, len(locations))
	for i, loc := range locations {
		result[i] = LocationResult(loc)
	}
	return result
}

func getStringValue(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func getCompletionKindValue(k *CompletionKind) int {
	if k == nil {
		return 0 // Default to 0 when nil (no specific kind)
	}
	return int(*k)
}

// SimpleDelegate provides a minimal implementation of LanguageServerDelegate
type SimpleDelegate struct{}

func (d *SimpleDelegate) ReadTextFile(path string) (string, error) {
	return readFileContent(path)
}

func (d *SimpleDelegate) Which(command string) (string, error) {
	return exec.LookPath(command)
}

func (d *SimpleDelegate) ShellEnv() map[string]string {
	env := make(map[string]string)
	for _, e := range os.Environ() {
		if i := strings.Index(e, "="); i >= 0 {
			env[e[:i]] = e[i+1:]
		}
	}
	return env
}

func (d *SimpleDelegate) WorkspaceRoot() string {
	return ""
}

// extractHoverContents extracts string content from hover contents
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
