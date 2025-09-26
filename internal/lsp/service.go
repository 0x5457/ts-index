package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

// LSPService provides HTTP endpoints for LSP functionality
type LSPService struct {
	tools *LSPTools
}

// NewLSPService creates a new LSP service
func NewLSPService() *LSPService {
	return &LSPService{
		tools: NewLSPTools(),
	}
}

// RegisterHandlers registers HTTP handlers for LSP endpoints
func (s *LSPService) RegisterHandlers(mux *http.ServeMux) {
	mux.HandleFunc("/lsp/analyze-symbol", s.handleAnalyzeSymbol)
	mux.HandleFunc("/lsp/completion", s.handleCompletion)
	mux.HandleFunc("/lsp/search-symbols", s.handleSearchSymbols)
	mux.HandleFunc("/lsp/document-symbols", s.handleDocumentSymbols)
	mux.HandleFunc("/lsp/health", s.handleHealth)
}

// handleAnalyzeSymbol handles symbol analysis requests
func (s *LSPService) handleAnalyzeSymbol(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req AnalyzeSymbolRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response := s.tools.AnalyzeSymbol(r.Context(), req)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleCompletion handles completion requests
func (s *LSPService) handleCompletion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetCompletionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response := s.tools.GetCompletion(r.Context(), req)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleSearchSymbols handles symbol search requests
func (s *LSPService) handleSearchSymbols(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req SearchSymbolsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response := s.tools.SearchSymbols(r.Context(), req)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleDocumentSymbols handles document symbols requests
func (s *LSPService) handleDocumentSymbols(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req GetDocumentSymbolsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	response := s.tools.GetDocumentSymbols(r.Context(), req)
	
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("Error encoding response: %v", err)
	}
}

// handleHealth handles health check requests
func (s *LSPService) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]interface{}{
		"status": "ok",
		"vtsls_available": IsVTSLSInstalled(),
	}

	if !IsVTSLSInstalled() {
		status["install_command"] = InstallVTSLSCommand()
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(status); err != nil {
		log.Printf("Error encoding health response: %v", err)
	}
}

// Cleanup shuts down the service
func (s *LSPService) Cleanup() error {
	return s.tools.Cleanup()
}

// LSPCommand provides command-line integration for LSP functionality
type LSPCommand struct {
	tools *LSPTools
}

// NewLSPCommand creates a new LSP command
func NewLSPCommand() *LSPCommand {
	return &LSPCommand{
		tools: NewLSPTools(),
	}
}

// AnalyzeSymbolJSON analyzes a symbol and returns JSON output
func (c *LSPCommand) AnalyzeSymbolJSON(ctx context.Context, workspaceRoot, filePath string, line, character int, includeHover, includeRefs, includeDefs bool) (string, error) {
	req := AnalyzeSymbolRequest{
		WorkspaceRoot: workspaceRoot,
		FilePath:      filePath,
		Line:          line,
		Character:     character,
		IncludeHover:  includeHover,
		IncludeRefs:   includeRefs,
		IncludeDefs:   includeDefs,
	}

	response := c.tools.AnalyzeSymbol(ctx, req)
	
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return string(data), nil
}

// GetCompletionJSON gets completion and returns JSON output
func (c *LSPCommand) GetCompletionJSON(ctx context.Context, workspaceRoot, filePath string, line, character, maxResults int) (string, error) {
	req := GetCompletionRequest{
		WorkspaceRoot: workspaceRoot,
		FilePath:      filePath,
		Line:          line,
		Character:     character,
		MaxResults:    maxResults,
	}

	response := c.tools.GetCompletion(ctx, req)
	
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return string(data), nil
}

// SearchSymbolsJSON searches symbols and returns JSON output
func (c *LSPCommand) SearchSymbolsJSON(ctx context.Context, workspaceRoot, query string, maxResults int) (string, error) {
	req := SearchSymbolsRequest{
		WorkspaceRoot: workspaceRoot,
		Query:         query,
		MaxResults:    maxResults,
	}

	response := c.tools.SearchSymbols(ctx, req)
	
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal response: %w", err)
	}
	
	return string(data), nil
}

// Cleanup shuts down the command
func (c *LSPCommand) Cleanup() error {
	return c.tools.Cleanup()
}