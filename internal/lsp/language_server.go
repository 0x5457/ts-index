package lsp

import (
	"context"
	"path/filepath"
)

// LanguageServerInterface represents the interface for language server implementations
// Inspired by Zed's LanguageServer trait design
type LanguageServerInterface interface {
	// Start initializes and starts the language server
	Start(ctx context.Context, workspaceRoot string) error

	// Stop gracefully shuts down the language server
	Stop() error

	// IsRunning returns true if the server is running
	IsRunning() bool

	// Hover provides hover information for a position in a document
	Hover(ctx context.Context, params TextDocumentPositionParams) (*Hover, error)

	// Completion provides completion items for a position in a document
	Completion(ctx context.Context, params TextDocumentPositionParams) (*CompletionList, error)

	// GotoDefinition provides goto definition information
	GotoDefinition(ctx context.Context, params TextDocumentPositionParams) ([]Location, error)

	// FindReferences finds all references to the symbol at the given position
	FindReferences(ctx context.Context, params TextDocumentPositionParams) ([]Location, error)

	// WorkspaceSymbols returns workspace symbols matching the query
	WorkspaceSymbols(ctx context.Context, params WorkspaceSymbolParams) ([]SymbolInformation, error)

	// DocumentSymbols returns document symbols for the given document
	DocumentSymbols(ctx context.Context, uri string) ([]SymbolInformation, error)

	// GetDiagnostics returns diagnostics for the given document
	GetDiagnostics(ctx context.Context, uri string) ([]Diagnostic, error)

	// DidOpen notifies the server that a document was opened
	DidOpen(ctx context.Context, uri string, content string) error

	// DidChange notifies the server that a document was changed
	DidChange(ctx context.Context, uri string, content string) error

	// DidClose notifies the server that a document was closed
	DidClose(ctx context.Context, uri string) error
}

// LanguageServerConfig represents configuration for a language server
type LanguageServerConfig struct {
	// Command is the command to run the language server
	Command string

	// Args are the arguments to pass to the command
	Args []string

	// WorkspaceRoot is the root directory of the workspace
	WorkspaceRoot string

	// InitializationOptions are server-specific initialization options
	InitializationOptions map[string]interface{}

	// Environment variables to set for the server process
	Env map[string]string
}

// LanguageServerFactory creates language servers for specific languages
type LanguageServerFactory interface {
	// CreateLanguageServer creates a new language server instance
	CreateLanguageServer(config LanguageServerConfig) (LanguageServerInterface, error)

	// SupportedLanguages returns the list of supported languages
	SupportedLanguages() []string

	// GetDefaultConfig returns the default configuration for the given language
	GetDefaultConfig(language string, workspaceRoot string) LanguageServerConfig
}

// Helper function to convert file paths to URIs
func PathToURI(path string) string {
	abs, _ := filepath.Abs(path)
	return "file://" + abs
}

// Helper function to convert URIs to file paths
func URIToPath(uri string) string {
	if len(uri) > 7 && uri[:7] == "file://" {
		return uri[7:]
	}
	return uri
}
