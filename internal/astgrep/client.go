package astgrep

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Client wraps ast-grep command execution
type Client struct {
	executable string
}

// NewClient creates a new ast-grep client
func NewClient() *Client {
	return &Client{
		executable: "ast-grep", // assume ast-grep is in PATH
	}
}

// Match represents an ast-grep match result
type Match struct {
	Text     string            `json:"text"`
	File     string            `json:"file"`
	Range    Range             `json:"range"`
	Language string            `json:"language"`
	Meta     map[string]string `json:"meta,omitempty"`
}

// Range represents a code range
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in code
type Position struct {
	Line   int `json:"line"`
	Column int `json:"column"`
	Index  int `json:"index"`
}

// SearchRequest represents parameters for ast-grep search
type SearchRequest struct {
	// Pattern is the ast-grep pattern to search for
	Pattern string `json:"pattern"`

	// Language specifies the programming language
	Language string `json:"language,omitempty"`

	// ProjectPath is the root path to search in
	ProjectPath string `json:"project_path"`

	// MaxResults limits the number of results
	MaxResults int `json:"max_results,omitempty"`

	// IncludeContext adds surrounding context lines
	IncludeContext int `json:"include_context,omitempty"`
}

// RuleSearchRequest represents parameters for rule-based search
type RuleSearchRequest struct {
	// Rule is the YAML rule content
	Rule string `json:"rule"`

	// ProjectPath is the root path to search in
	ProjectPath string `json:"project_path"`

	// MaxResults limits the number of results
	MaxResults int `json:"max_results,omitempty"`
}

// TestRuleRequest represents parameters for testing a rule
type TestRuleRequest struct {
	// Rule is the YAML rule content
	Rule string `json:"rule"`

	// Code is the code snippet to test
	Code string `json:"code"`

	// Language specifies the programming language
	Language string `json:"language,omitempty"`
}

// SyntaxTreeRequest represents parameters for syntax tree dumping
type SyntaxTreeRequest struct {
	// Code is the code snippet to analyze
	Code string `json:"code"`

	// Language specifies the programming language
	Language string `json:"language,omitempty"`
}

// SearchResponse represents the result of a search operation
type SearchResponse struct {
	Matches []Match `json:"matches"`
	Error   string  `json:"error,omitempty"`
}

// TestRuleResponse represents the result of testing a rule
type TestRuleResponse struct {
	Matches []Match `json:"matches"`
	Success bool    `json:"success"`
	Error   string  `json:"error,omitempty"`
}

// SyntaxTreeResponse represents the result of syntax tree dumping
type SyntaxTreeResponse struct {
	Tree  string `json:"tree"`
	Error string `json:"error,omitempty"`
}

// Search performs pattern-based search using ast-grep
func (c *Client) Search(ctx context.Context, req SearchRequest) SearchResponse {
	args := []string{"run"}

	// Add pattern
	args = append(args, "--pattern", req.Pattern)

	// Add language if specified
	if req.Language != "" {
		args = append(args, "--lang", req.Language)
	}

	// Add JSON output
	args = append(args, "--json")

	// Add context if specified
	if req.IncludeContext > 0 {
		args = append(args, "--context", fmt.Sprintf("%d", req.IncludeContext))
	}

	// Add project path
	args = append(args, req.ProjectPath)

	return c.executeSearch(ctx, args, req.MaxResults)
}

// SearchByRule performs rule-based search using ast-grep
func (c *Client) SearchByRule(ctx context.Context, req RuleSearchRequest) SearchResponse {
	// Create temporary rule file
	ruleFile, err := c.createTempRuleFile(req.Rule)
	if err != nil {
		return SearchResponse{Error: fmt.Sprintf("failed to create rule file: %v", err)}
	}
	defer c.cleanupTempFile(ruleFile)

	args := []string{"run"}
	args = append(args, "--rule", ruleFile)
	args = append(args, "--json")
	args = append(args, req.ProjectPath)

	return c.executeSearch(ctx, args, req.MaxResults)
}

// TestRule tests a rule against code snippet
func (c *Client) TestRule(ctx context.Context, req TestRuleRequest) TestRuleResponse {
	// Create temporary rule file
	ruleFile, err := c.createTempRuleFile(req.Rule)
	if err != nil {
		return TestRuleResponse{Error: fmt.Sprintf("failed to create rule file: %v", err)}
	}
	defer c.cleanupTempFile(ruleFile)

	// Create temporary code file
	codeFile, err := c.createTempCodeFile(req.Code, req.Language)
	if err != nil {
		return TestRuleResponse{Error: fmt.Sprintf("failed to create code file: %v", err)}
	}
	defer c.cleanupTempFile(codeFile)

	args := []string{"run"}
	args = append(args, "--rule", ruleFile)
	args = append(args, "--json")
	args = append(args, codeFile)

	response := c.executeSearch(ctx, args, 0)
	return TestRuleResponse{
		Matches: response.Matches,
		Success: response.Error == "",
		Error:   response.Error,
	}
}

// DumpSyntaxTree dumps the syntax tree of code
func (c *Client) DumpSyntaxTree(ctx context.Context, req SyntaxTreeRequest) SyntaxTreeResponse {
	// Create temporary code file
	codeFile, err := c.createTempCodeFile(req.Code, req.Language)
	if err != nil {
		return SyntaxTreeResponse{Error: fmt.Sprintf("failed to create code file: %v", err)}
	}
	defer c.cleanupTempFile(codeFile)

	args := []string{"scan", codeFile}

	// Add language if specified
	if req.Language != "" {
		args = append(args, "--lang", req.Language)
	}

	cmd := exec.CommandContext(ctx, c.executable, args...)
	output, err := cmd.Output()
	if err != nil {
		return SyntaxTreeResponse{Error: fmt.Sprintf("ast-grep command failed: %v", err)}
	}

	return SyntaxTreeResponse{Tree: string(output)}
}

// executeSearch is a helper to execute search commands
func (c *Client) executeSearch(ctx context.Context, args []string, maxResults int) SearchResponse {
	cmd := exec.CommandContext(ctx, c.executable, args...)
	output, err := cmd.Output()
	if err != nil {
		return SearchResponse{Error: fmt.Sprintf("ast-grep command failed: %v", err)}
	}

	// Parse JSON output
	var matches []Match
	if len(output) > 0 {
		if err := json.Unmarshal(output, &matches); err != nil {
			return SearchResponse{Error: fmt.Sprintf("failed to parse ast-grep output: %v", err)}
		}
	}

	// Apply max results limit
	if maxResults > 0 && len(matches) > maxResults {
		matches = matches[:maxResults]
	}

	return SearchResponse{Matches: matches}
}

// createTempRuleFile creates a temporary YAML rule file
func (c *Client) createTempRuleFile(rule string) (string, error) {
	return c.createTempFile(rule, "rule-*.yml")
}

// createTempCodeFile creates a temporary code file
func (c *Client) createTempCodeFile(code, language string) (string, error) {
	ext := c.getFileExtension(language)
	return c.createTempFile(code, fmt.Sprintf("code-*%s", ext))
}

// createTempFile creates a temporary file with content
func (c *Client) createTempFile(content, pattern string) (string, error) {
	tmpDir := "/tmp"

	// For simplicity, create a unique filename
	filename := strings.Replace(pattern, "*", fmt.Sprintf("%d", len(content)+int(^uint(0)>>1)), 1)
	fullPath := filepath.Join(tmpDir, filename)

	if err := writeFile(fullPath, content); err != nil {
		return "", err
	}

	return fullPath, nil
}

// cleanupTempFile removes a temporary file
func (c *Client) cleanupTempFile(path string) {
	_ = removeFile(path)
}

// getFileExtension returns the appropriate file extension for a language
func (c *Client) getFileExtension(language string) string {
	switch strings.ToLower(language) {
	case "typescript", "ts":
		return ".ts"
	case "javascript", "js":
		return ".js"
	case "tsx":
		return ".tsx"
	case "jsx":
		return ".jsx"
	case "python", "py":
		return ".py"
	case "go":
		return ".go"
	case "rust", "rs":
		return ".rs"
	case "java":
		return ".java"
	case "c":
		return ".c"
	case "cpp", "c++":
		return ".cpp"
	default:
		return ".txt"
	}
}
