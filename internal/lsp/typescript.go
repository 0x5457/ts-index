package lsp

import (
	"context"
	"os/exec"
	"path/filepath"
)

// TypeScriptLanguageServerFactory creates TypeScript language servers
type TypeScriptLanguageServerFactory struct{}

// NewTypeScriptLanguageServerFactory creates a new TypeScript language server factory
func NewTypeScriptLanguageServerFactory() *TypeScriptLanguageServerFactory {
	return &TypeScriptLanguageServerFactory{}
}

// CreateLanguageServer implements LanguageServerFactory.CreateLanguageServer
func (f *TypeScriptLanguageServerFactory) CreateLanguageServer(config LanguageServerConfig) (LanguageServer, error) {
	return NewLSPClient(config), nil
}

// SupportedLanguages implements LanguageServerFactory.SupportedLanguages
func (f *TypeScriptLanguageServerFactory) SupportedLanguages() []string {
	return []string{"typescript", "javascript", "typescriptreact", "javascriptreact"}
}

// GetDefaultConfig implements LanguageServerFactory.GetDefaultConfig
func (f *TypeScriptLanguageServerFactory) GetDefaultConfig(language string, workspaceRoot string) LanguageServerConfig {
	// Try vtsls first, fallback to typescript-language-server
	command := "vtsls"
	args := []string{"--stdio"}
	
	if !IsVTSLSInstalled() && IsTypeScriptLanguageServerInstalled() {
		command = "typescript-language-server"
		args = []string{"--stdio"}
	}
	
	return LanguageServerConfig{
		Command:       command,
		Args:          args,
		WorkspaceRoot: workspaceRoot,
		InitializationOptions: map[string]interface{}{
			"preferences": map[string]interface{}{
				"includeCompletionsForModuleExports": true,
				"includeCompletionsWithInsertText": true,
			},
			"typescript": map[string]interface{}{
				"suggest": map[string]interface{}{
					"autoImports": true,
				},
				"inlayHints": map[string]interface{}{
					"includeInlayParameterNameHints":     "all",
					"includeInlayParameterNameHintsWhenArgumentMatchesName": false,
					"includeInlayFunctionParameterTypeHints": true,
					"includeInlayVariableTypeHints": false,
					"includeInlayPropertyDeclarationTypeHints": true,
					"includeInlayFunctionLikeReturnTypeHints": true,
					"includeInlayEnumMemberValueHints": true,
				},
			},
		},
		Env: map[string]string{
			// Add any required environment variables
		},
	}
}

// TypeScriptLanguageServerManager manages TypeScript language servers for different projects
type TypeScriptLanguageServerManager struct {
	factory *TypeScriptLanguageServerFactory
	servers map[string]LanguageServer // map from workspace root to server
}

// NewTypeScriptLanguageServerManager creates a new TypeScript language server manager
func NewTypeScriptLanguageServerManager() *TypeScriptLanguageServerManager {
	return &TypeScriptLanguageServerManager{
		factory: NewTypeScriptLanguageServerFactory(),
		servers: make(map[string]LanguageServer),
	}
}

// GetOrCreateServer gets or creates a language server for the given workspace
func (m *TypeScriptLanguageServerManager) GetOrCreateServer(ctx context.Context, workspaceRoot string) (LanguageServer, error) {
	// Normalize workspace root path
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	// Check if server already exists
	if server, exists := m.servers[absRoot]; exists && server.IsRunning() {
		return server, nil
	}
	
	// Create new server
	config := m.factory.GetDefaultConfig("typescript", absRoot)
	server, err := m.factory.CreateLanguageServer(config)
	if err != nil {
		return nil, err
	}
	
	// Start the server
	if err := server.Start(ctx, absRoot); err != nil {
		return nil, err
	}
	
	m.servers[absRoot] = server
	return server, nil
}

// StopServer stops the language server for the given workspace
func (m *TypeScriptLanguageServerManager) StopServer(workspaceRoot string) error {
	absRoot, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return err
	}
	
	if server, exists := m.servers[absRoot]; exists {
		err := server.Stop()
		delete(m.servers, absRoot)
		return err
	}
	
	return nil
}

// StopAllServers stops all running language servers
func (m *TypeScriptLanguageServerManager) StopAllServers() error {
	var lastErr error
	for root, server := range m.servers {
		if err := server.Stop(); err != nil {
			lastErr = err
		}
		delete(m.servers, root)
	}
	return lastErr
}

// IsVTSLSInstalled checks if vtsls is installed and available
func IsVTSLSInstalled() bool {
	_, err := exec.LookPath("vtsls")
	return err == nil
}

// InstallVTSLSCommand returns the command to install vtsls
func InstallVTSLSCommand() string {
	return "npm install -g @vtsls/language-server"
}

// IsTypeScriptLanguageServerInstalled checks if typescript-language-server is installed and available
func IsTypeScriptLanguageServerInstalled() bool {
	_, err := exec.LookPath("typescript-language-server")
	return err == nil
}

// InstallTypeScriptLanguageServerCommand returns the command to install typescript-language-server
func InstallTypeScriptLanguageServerCommand() string {
	return "npm install -g typescript-language-server typescript"
}

// TypeScriptFeatures provides high-level TypeScript-specific features
type TypeScriptFeatures struct {
	manager *TypeScriptLanguageServerManager
}

// NewTypeScriptFeatures creates a new TypeScript features instance
func NewTypeScriptFeatures() *TypeScriptFeatures {
	return &TypeScriptFeatures{
		manager: NewTypeScriptLanguageServerManager(),
	}
}

// GetHoverInfo gets hover information for a TypeScript file at a specific position
func (f *TypeScriptFeatures) GetHoverInfo(ctx context.Context, workspaceRoot, filePath string, line, character int) (*Hover, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	uri := PathToURI(filePath)
	
	// Ensure the document is open
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	
	if err := server.DidOpen(ctx, uri, content); err != nil {
		return nil, err
	}
	
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}
	
	return server.Hover(ctx, params)
}

// GetDefinition gets definition locations for a symbol in a TypeScript file
func (f *TypeScriptFeatures) GetDefinition(ctx context.Context, workspaceRoot, filePath string, line, character int) ([]Location, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	uri := PathToURI(filePath)
	
	// Ensure the document is open
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	
	if err := server.DidOpen(ctx, uri, content); err != nil {
		return nil, err
	}
	
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}
	
	return server.GotoDefinition(ctx, params)
}

// GetReferences gets reference locations for a symbol in a TypeScript file
func (f *TypeScriptFeatures) GetReferences(ctx context.Context, workspaceRoot, filePath string, line, character int) ([]Location, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	uri := PathToURI(filePath)
	
	// Ensure the document is open
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	
	if err := server.DidOpen(ctx, uri, content); err != nil {
		return nil, err
	}
	
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}
	
	return server.FindReferences(ctx, params)
}

// GetCompletion gets completion items for a TypeScript file at a specific position
func (f *TypeScriptFeatures) GetCompletion(ctx context.Context, workspaceRoot, filePath string, line, character int) (*CompletionList, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	uri := PathToURI(filePath)
	
	// Ensure the document is open
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	
	if err := server.DidOpen(ctx, uri, content); err != nil {
		return nil, err
	}
	
	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     Position{Line: line, Character: character},
	}
	
	return server.Completion(ctx, params)
}

// GetWorkspaceSymbols gets workspace symbols matching a query
func (f *TypeScriptFeatures) GetWorkspaceSymbols(ctx context.Context, workspaceRoot, query string) ([]SymbolInformation, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	params := WorkspaceSymbolParams{Query: query}
	return server.WorkspaceSymbols(ctx, params)
}

// GetDocumentSymbols gets document symbols for a TypeScript file
func (f *TypeScriptFeatures) GetDocumentSymbols(ctx context.Context, workspaceRoot, filePath string) ([]SymbolInformation, error) {
	server, err := f.manager.GetOrCreateServer(ctx, workspaceRoot)
	if err != nil {
		return nil, err
	}
	
	uri := PathToURI(filePath)
	
	// Ensure the document is open
	content, err := readFileContent(filePath)
	if err != nil {
		return nil, err
	}
	
	if err := server.DidOpen(ctx, uri, content); err != nil {
		return nil, err
	}
	
	return server.DocumentSymbols(ctx, uri)
}

// Cleanup shuts down all language servers
func (f *TypeScriptFeatures) Cleanup() error {
	return f.manager.StopAllServers()
}