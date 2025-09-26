package lsp

import (
	"context"
	"fmt"
)

// LspAdapter represents a language-specific LSP adapter, inspired by Zed's design
// This is the main interface that language-specific implementations should fulfill
type LspAdapter interface {
	// Name returns the name of this language server
	Name() string

	// LanguageIds returns a mapping of language names to LSP language identifiers
	LanguageIds() map[string]string

	// ServerCommand returns the command and arguments to start the language server
	ServerCommand(workspaceRoot string) (string, []string, error)

	// InitializationOptions returns options to send during LSP initialization
	InitializationOptions(workspaceRoot string) (map[string]interface{}, error)

	// WorkspaceConfiguration returns workspace-specific configuration
	WorkspaceConfiguration(workspaceRoot string) (map[string]interface{}, error)

	// ProcessDiagnostics allows the adapter to modify diagnostics before they're used
	ProcessDiagnostics(diagnostics []Diagnostic) []Diagnostic

	// ProcessCompletions allows the adapter to modify completion items
	ProcessCompletions(items []CompletionItem) []CompletionItem

	// CanInstall returns true if this adapter can install its language server
	CanInstall() bool

	// Install installs the language server for this adapter
	Install(ctx context.Context) error

	// IsInstalled checks if the language server is already installed
	IsInstalled() bool
}

// LanguageServerBinary represents a language server executable
type LanguageServerBinary struct {
	Path string
	Args []string
	Env  map[string]string
}

// LanguageServerDelegate provides services that adapters can use
type LanguageServerDelegate interface {
	// ReadTextFile reads a text file from the workspace
	ReadTextFile(path string) (string, error)

	// Which finds an executable in PATH
	Which(command string) (string, error)

	// ShellEnv returns the shell environment
	ShellEnv() map[string]string

	// WorkspaceRoot returns the workspace root path
	WorkspaceRoot() string
}

// LanguageServer represents an active LSP connection, similar to Zed's LanguageServer
type LanguageServer struct {
	adapter    LspAdapter
	client     *LSPClient
	delegate   LanguageServerDelegate
	rootPath   string
	serverName string
}

// NewLanguageServer creates a new language server instance
func NewLanguageServer(
	adapter LspAdapter,
	delegate LanguageServerDelegate,
	rootPath string,
) *LanguageServer {
	return &LanguageServer{
		adapter:    adapter,
		delegate:   delegate,
		rootPath:   rootPath,
		serverName: adapter.Name(),
	}
}

// Start initializes and starts the language server
func (ls *LanguageServer) Start(ctx context.Context) error {
	// Get server command from adapter
	command, args, err := ls.adapter.ServerCommand(ls.rootPath)
	if err != nil {
		return err
	}

	// Get configuration from adapter
	initOptions, err := ls.adapter.InitializationOptions(ls.rootPath)
	if err != nil {
		return err
	}

	// Create LSP client configuration
	config := LanguageServerConfig{
		Command:               command,
		Args:                  args,
		WorkspaceRoot:         ls.rootPath,
		InitializationOptions: initOptions,
		Env:                   ls.delegate.ShellEnv(),
	}

	// Create and start client
	ls.client = NewLSPClient(config)
	return ls.client.Start(ctx, ls.rootPath)
}

// Stop shuts down the language server
func (ls *LanguageServer) Stop() error {
	if ls.client != nil {
		return ls.client.Stop()
	}
	return nil
}

// IsRunning returns true if the server is running
func (ls *LanguageServer) IsRunning() bool {
	return ls.client != nil && ls.client.IsRunning()
}

// Hover provides hover information
func (ls *LanguageServer) Hover(
	ctx context.Context,
	uri string,
	position Position,
) (*Hover, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}

	return ls.client.Hover(ctx, params)
}

// Completion provides code completion
func (ls *LanguageServer) Completion(
	ctx context.Context,
	uri string,
	position Position,
) (*CompletionList, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}

	result, err := ls.client.Completion(ctx, params)
	if err != nil {
		return nil, err
	}

	// Process completions through adapter
	if result != nil {
		result.Items = ls.adapter.ProcessCompletions(result.Items)
	}

	return result, nil
}

// GotoDefinition finds symbol definitions
func (ls *LanguageServer) GotoDefinition(
	ctx context.Context,
	uri string,
	position Position,
) ([]Location, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}

	return ls.client.GotoDefinition(ctx, params)
}

// FindReferences finds symbol references
func (ls *LanguageServer) FindReferences(
	ctx context.Context,
	uri string,
	position Position,
	includeDeclaration bool,
) ([]Location, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	params := TextDocumentPositionParams{
		TextDocument: TextDocumentIdentifier{URI: uri},
		Position:     position,
	}

	return ls.client.FindReferences(ctx, params)
}

// WorkspaceSymbols searches for symbols in the workspace
func (ls *LanguageServer) WorkspaceSymbols(
	ctx context.Context,
	query string,
) ([]SymbolInformation, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	params := WorkspaceSymbolParams{Query: query}
	return ls.client.WorkspaceSymbols(ctx, params)
}

// DocumentSymbols gets symbols for a document
func (ls *LanguageServer) DocumentSymbols(
	ctx context.Context,
	uri string,
) ([]SymbolInformation, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	return ls.client.DocumentSymbols(ctx, uri)
}

// DidOpen notifies the server that a document was opened
func (ls *LanguageServer) DidOpen(ctx context.Context, uri string, content string) error {
	if ls.client == nil {
		return ErrServerNotRunning
	}

	return ls.client.DidOpen(ctx, uri, content)
}

// DidChange notifies the server that a document was changed
func (ls *LanguageServer) DidChange(ctx context.Context, uri string, content string) error {
	if ls.client == nil {
		return ErrServerNotRunning
	}

	return ls.client.DidChange(ctx, uri, content)
}

// DidClose notifies the server that a document was closed
func (ls *LanguageServer) DidClose(ctx context.Context, uri string) error {
	if ls.client == nil {
		return ErrServerNotRunning
	}

	return ls.client.DidClose(ctx, uri)
}

// GetDiagnostics returns diagnostics for a document
func (ls *LanguageServer) GetDiagnostics(ctx context.Context, uri string) ([]Diagnostic, error) {
	if ls.client == nil {
		return nil, ErrServerNotRunning
	}

	diagnostics, err := ls.client.GetDiagnostics(ctx, uri)
	if err != nil {
		return nil, err
	}

	// Process diagnostics through adapter
	return ls.adapter.ProcessDiagnostics(diagnostics), nil
}

// Adapter returns the underlying adapter
func (ls *LanguageServer) Adapter() LspAdapter {
	return ls.adapter
}

// Name returns the server name
func (ls *LanguageServer) Name() string {
	return ls.serverName
}

// RootPath returns the workspace root path
func (ls *LanguageServer) RootPath() string {
	return ls.rootPath
}

var ErrServerNotRunning = fmt.Errorf("language server is not running")
