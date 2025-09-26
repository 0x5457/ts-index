package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LSPClient implements a Language Server Protocol client
type LSPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	running   int32
	requestID int32

	// Channels for handling responses and notifications
	responses    map[int]chan json.RawMessage
	responsesMux sync.RWMutex

	// Configuration
	config LanguageServerConfig

	// Workspace state
	workspaceRoot string
	openDocuments map[string]bool
	documentsMux  sync.RWMutex
}

// LSPRequest represents a JSON-RPC 2.0 request
type LSPRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// LSPResponse represents a JSON-RPC 2.0 response
type LSPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      *int            `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *LSPError       `json:"error,omitempty"`
}

// LSPError represents a JSON-RPC 2.0 error
type LSPError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// LSPNotification represents a JSON-RPC 2.0 notification
type LSPNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params"`
}

// NewLSPClient creates a new LSP client
func NewLSPClient(config LanguageServerConfig) *LSPClient {
	return &LSPClient{
		config:        config,
		responses:     make(map[int]chan json.RawMessage),
		openDocuments: make(map[string]bool),
		workspaceRoot: config.WorkspaceRoot,
	}
}

// Start implements LanguageServer.Start
func (c *LSPClient) Start(ctx context.Context, workspaceRoot string) error {
	if atomic.LoadInt32(&c.running) == 1 {
		return fmt.Errorf("language server is already running")
	}

	c.workspaceRoot = workspaceRoot
	c.config.WorkspaceRoot = workspaceRoot

	// Create command
	c.cmd = exec.CommandContext(ctx, c.config.Command, c.config.Args...)
	c.cmd.Dir = workspaceRoot

	// Set environment variables
	c.cmd.Env = os.Environ()
	for key, value := range c.config.Env {
		c.cmd.Env = append(c.cmd.Env, fmt.Sprintf("%s=%s", key, value))
	}

	// Setup pipes
	var err error
	c.stdin, err = c.cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	c.stdout, err = c.cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	c.stderr, err = c.cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the process
	if err := c.cmd.Start(); err != nil {
		return fmt.Errorf("failed to start language server: %w", err)
	}

	log.Printf("Started language server process: %s %v", c.config.Command, c.config.Args)

	atomic.StoreInt32(&c.running, 1)

	// Start goroutines to handle I/O
	go c.handleStdout()
	go c.handleStderr()

	// Initialize the server
	if err := c.initialize(ctx); err != nil {
		c.Stop()
		return fmt.Errorf("failed to initialize language server: %w", err)
	}

	// Check if process is still running after initialization
	if c.cmd.ProcessState != nil && c.cmd.ProcessState.Exited() {
		return fmt.Errorf(
			"language server process exited during initialization: %s",
			c.cmd.ProcessState.String(),
		)
	}

	return nil
}

// Stop implements LanguageServer.Stop
func (c *LSPClient) Stop() error {
	if atomic.LoadInt32(&c.running) == 0 {
		return nil
	}

	atomic.StoreInt32(&c.running, 0)

	// Send shutdown request
	c.sendNotification("shutdown", nil)
	c.sendNotification("exit", nil)

	// Close pipes
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}

	// Wait for process to exit
	if c.cmd != nil && c.cmd.Process != nil {
		c.cmd.Wait()
	}

	return nil
}

// IsRunning implements LanguageServer.IsRunning
func (c *LSPClient) IsRunning() bool {
	return atomic.LoadInt32(&c.running) == 1
}

// sendRequest sends a request and waits for response
func (c *LSPClient) sendRequest(
	ctx context.Context,
	method string,
	params interface{},
) (json.RawMessage, error) {
	if !c.IsRunning() {
		return nil, fmt.Errorf("language server is not running")
	}

	id := int(atomic.AddInt32(&c.requestID, 1))

	// Create response channel
	respChan := make(chan json.RawMessage, 1)
	c.responsesMux.Lock()
	c.responses[id] = respChan
	c.responsesMux.Unlock()

	defer func() {
		c.responsesMux.Lock()
		delete(c.responses, id)
		c.responsesMux.Unlock()
	}()

	// Send request
	req := LSPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.sendMessage(req); err != nil {
		return nil, err
	}

	// Wait for response
	select {
	case response := <-respChan:
		return response, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendNotification sends a notification (no response expected)
func (c *LSPClient) sendNotification(method string, params interface{}) error {
	if !c.IsRunning() {
		return fmt.Errorf("language server is not running")
	}

	notif := LSPNotification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}

	return c.sendMessage(notif)
}

// sendMessage sends a JSON-RPC message using the LSP protocol
func (c *LSPClient) sendMessage(message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}

	content := fmt.Sprintf("Content-Length: %d\r\n\r\n%s", len(data), data)
	log.Printf("Sending LSP message: %s", content)
	_, err = c.stdin.Write([]byte(content))
	return err
}

// handleStdout handles stdout from the language server
func (c *LSPClient) handleStdout() {
	reader := bufio.NewReader(c.stdout)

	for c.IsRunning() {
		// Read headers
		headers := make(map[string]string)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("Error reading from language server stdout: %v", err)
				}
				return
			}

			line = strings.TrimSpace(line)
			if line == "" {
				break // End of headers
			}

			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				headers[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
			}
		}

		// Read content
		contentLengthStr, ok := headers["Content-Length"]
		if !ok {
			continue
		}

		contentLength, err := strconv.Atoi(contentLengthStr)
		if err != nil {
			log.Printf("Invalid Content-Length: %s", contentLengthStr)
			continue
		}

		content := make([]byte, contentLength)
		_, err = io.ReadFull(reader, content)
		if err != nil {
			log.Printf("Error reading content: %v", err)
			continue
		}

		// Parse JSON-RPC message
		var response LSPResponse
		if err := json.Unmarshal(content, &response); err != nil {
			log.Printf("Error parsing JSON-RPC response: %v", err)
			continue
		}

		// Handle response
		if response.ID != nil {
			c.responsesMux.RLock()
			respChan, ok := c.responses[*response.ID]
			c.responsesMux.RUnlock()

			if ok && respChan != nil {
				if response.Error != nil {
					// Handle error response
					log.Printf("LSP Error: %s", response.Error.Message)
				} else {
					select {
					case respChan <- response.Result:
					default:
					}
				}
			}
		}
		// Note: We could handle notifications here if needed
	}
}

// handleStderr handles stderr from the language server
func (c *LSPClient) handleStderr() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() && c.IsRunning() {
		line := scanner.Text()
		log.Printf("LSP stderr: %s", line)
	}
	if err := scanner.Err(); err != nil {
		log.Printf("Error reading stderr: %v", err)
	}
}

// initialize sends the initialize request to the language server
func (c *LSPClient) initialize(ctx context.Context) error {
	params := map[string]interface{}{
		"processId": os.Getpid(),
		"rootUri":   PathToURI(c.workspaceRoot),
		"capabilities": map[string]interface{}{
			"textDocument": map[string]interface{}{
				"hover": map[string]interface{}{
					"contentFormat": []string{"markdown", "plaintext"},
				},
				"completion": map[string]interface{}{
					"completionItem": map[string]interface{}{
						"snippetSupport": true,
					},
				},
				"definition": map[string]interface{}{
					"linkSupport": true,
				},
				"references":     map[string]interface{}{},
				"documentSymbol": map[string]interface{}{},
			},
			"workspace": map[string]interface{}{
				"symbol": map[string]interface{}{},
			},
		},
		"initializationOptions": c.config.InitializationOptions,
	}

	// Create timeout context for initialization
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	_, err := c.sendRequest(initCtx, "initialize", params)
	if err != nil {
		return err
	}

	// Send initialized notification
	return c.sendNotification("initialized", map[string]interface{}{})
}

// Hover implements LanguageServer.Hover
func (c *LSPClient) Hover(ctx context.Context, params TextDocumentPositionParams) (*Hover, error) {
	response, err := c.sendRequest(ctx, "textDocument/hover", params)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return nil, nil
	}

	var hover Hover
	if err := json.Unmarshal(response, &hover); err != nil {
		return nil, err
	}

	return &hover, nil
}

// Completion implements LanguageServer.Completion
func (c *LSPClient) Completion(
	ctx context.Context,
	params TextDocumentPositionParams,
) (*CompletionList, error) {
	response, err := c.sendRequest(ctx, "textDocument/completion", params)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return &CompletionList{IsIncomplete: false, Items: []CompletionItem{}}, nil
	}

	// Try to parse as CompletionList first
	var completionList CompletionList
	if err := json.Unmarshal(response, &completionList); err == nil {
		return &completionList, nil
	}

	// Fallback: try to parse as array of CompletionItem
	var items []CompletionItem
	if err := json.Unmarshal(response, &items); err != nil {
		return nil, err
	}

	return &CompletionList{
		IsIncomplete: false,
		Items:        items,
	}, nil
}

// GotoDefinition implements LanguageServer.GotoDefinition
func (c *LSPClient) GotoDefinition(
	ctx context.Context,
	params TextDocumentPositionParams,
) ([]Location, error) {
	response, err := c.sendRequest(ctx, "textDocument/definition", params)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return []Location{}, nil
	}

	// Try to parse as array of Location first
	var locations []Location
	if err := json.Unmarshal(response, &locations); err == nil {
		return locations, nil
	}

	// Fallback: try to parse as single Location
	var location Location
	if err := json.Unmarshal(response, &location); err != nil {
		return nil, err
	}

	return []Location{location}, nil
}

// FindReferences implements LanguageServer.FindReferences
func (c *LSPClient) FindReferences(
	ctx context.Context,
	params TextDocumentPositionParams,
) ([]Location, error) {
	refParams := struct {
		TextDocumentPositionParams
		Context struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		} `json:"context"`
	}{
		TextDocumentPositionParams: params,
		Context: struct {
			IncludeDeclaration bool `json:"includeDeclaration"`
		}{
			IncludeDeclaration: true,
		},
	}

	response, err := c.sendRequest(ctx, "textDocument/references", refParams)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return []Location{}, nil
	}

	var locations []Location
	if err := json.Unmarshal(response, &locations); err != nil {
		return nil, err
	}

	return locations, nil
}

// WorkspaceSymbols implements LanguageServer.WorkspaceSymbols
func (c *LSPClient) WorkspaceSymbols(
	ctx context.Context,
	params WorkspaceSymbolParams,
) ([]SymbolInformation, error) {
	response, err := c.sendRequest(ctx, "workspace/symbol", params)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return []SymbolInformation{}, nil
	}

	var symbols []SymbolInformation
	if err := json.Unmarshal(response, &symbols); err != nil {
		return nil, err
	}

	return symbols, nil
}

// DocumentSymbols implements LanguageServer.DocumentSymbols
func (c *LSPClient) DocumentSymbols(ctx context.Context, uri string) ([]SymbolInformation, error) {
	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	response, err := c.sendRequest(ctx, "textDocument/documentSymbol", params)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "null" {
		return []SymbolInformation{}, nil
	}

	var symbols []SymbolInformation
	if err := json.Unmarshal(response, &symbols); err != nil {
		return nil, err
	}

	return symbols, nil
}

// GetDiagnostics implements LanguageServer.GetDiagnostics
func (c *LSPClient) GetDiagnostics(ctx context.Context, uri string) ([]Diagnostic, error) {
	// Note: LSP doesn't have a direct "get diagnostics" request
	// Diagnostics are typically sent as notifications from the server
	// For now, we return an empty slice
	// In a full implementation, we would store diagnostics from notifications
	return []Diagnostic{}, nil
}

// DidOpen implements LanguageServer.DidOpen
func (c *LSPClient) DidOpen(ctx context.Context, uri string, content string) error {
	c.documentsMux.Lock()
	c.openDocuments[uri] = true
	c.documentsMux.Unlock()

	params := struct {
		TextDocument struct {
			URI        string `json:"uri"`
			LanguageID string `json:"languageId"`
			Version    int    `json:"version"`
			Text       string `json:"text"`
		} `json:"textDocument"`
	}{
		TextDocument: struct {
			URI        string `json:"uri"`
			LanguageID string `json:"languageId"`
			Version    int    `json:"version"`
			Text       string `json:"text"`
		}{
			URI:        uri,
			LanguageID: c.getLanguageID(uri),
			Version:    1,
			Text:       content,
		},
	}

	return c.sendNotification("textDocument/didOpen", params)
}

// DidChange implements LanguageServer.DidChange
func (c *LSPClient) DidChange(ctx context.Context, uri string, content string) error {
	params := struct {
		TextDocument struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		} `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}{
		TextDocument: struct {
			URI     string `json:"uri"`
			Version int    `json:"version"`
		}{
			URI:     uri,
			Version: 2, // In a real implementation, track version numbers
		},
		ContentChanges: []struct {
			Text string `json:"text"`
		}{
			{Text: content},
		},
	}

	return c.sendNotification("textDocument/didChange", params)
}

// DidClose implements LanguageServer.DidClose
func (c *LSPClient) DidClose(ctx context.Context, uri string) error {
	c.documentsMux.Lock()
	delete(c.openDocuments, uri)
	c.documentsMux.Unlock()

	params := struct {
		TextDocument TextDocumentIdentifier `json:"textDocument"`
	}{
		TextDocument: TextDocumentIdentifier{URI: uri},
	}

	return c.sendNotification("textDocument/didClose", params)
}

// getLanguageID determines the language ID from the URI
func (c *LSPClient) getLanguageID(uri string) string {
	path := URIToPath(uri)
	if strings.HasSuffix(path, ".ts") {
		return "typescript"
	}
	if strings.HasSuffix(path, ".tsx") {
		return "typescriptreact"
	}
	if strings.HasSuffix(path, ".js") {
		return "javascript"
	}
	if strings.HasSuffix(path, ".jsx") {
		return "javascriptreact"
	}
	return "typescript" // default
}
