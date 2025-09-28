package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

// Client wraps an MCP stdio client aimed at our own server executable.
type Client struct {
	c *client.Client
}

// ServerConfig contains configuration for launching the MCP server
type ServerConfig struct {
	Project  string
	DB       string
	EmbedURL string
}

// NewStdioClient creates and initializes an MCP client that launches this binary with mcp.
func NewStdioClient(ctx context.Context) (*Client, error) {
	return NewStdioClientWithConfig(ctx, ServerConfig{})
}

// NewStdioClientWithConfig creates and initializes an MCP client with server configuration.
func NewStdioClientWithConfig(ctx context.Context, config ServerConfig) (*Client, error) {
	// Get the path of current executable
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("get executable path: %w", err)
	}

	// Build server arguments
	args := []string{"mcp"}
	if config.Project != "" {
		args = append(args, "--project", config.Project)
	}
	if config.DB != "" {
		args = append(args, "--db", config.DB)
	}
	if config.EmbedURL != "" {
		args = append(args, "--embed-url", config.EmbedURL)
	}

	// First, test if the server can start properly by running it briefly
	testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	var stderrBuf bytes.Buffer
	cmd := exec.CommandContext(testCtx, exePath, args...)
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start MCP server process: %w", err)
	}

	// Wait a moment to see if the process exits with an error
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case err := <-done:
		// Process exited quickly, likely due to an error
		if err != nil {
			stderrContent := stderrBuf.String()
			if stderrContent != "" {
				return nil, fmt.Errorf(
					"MCP server failed to start: %w\nServer output: %s",
					err,
					stderrContent,
				)
			}
			return nil, fmt.Errorf("MCP server failed to start: %w", err)
		}
	case <-testCtx.Done():
		// Process is still running after timeout, kill it and proceed
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}

	// Now create the actual transport
	tr := transport.NewStdio(exePath, nil, args...)
	if err := tr.Start(ctx); err != nil {
		return nil, fmt.Errorf("start mcp transport: %w", err)
	}
	cli := client.NewClient(tr)
	return initializeClient(ctx, cli)
}

// NewHTTPClient creates an MCP client using Streamable HTTP transport to a serverURL,
// for example: http://127.0.0.1:8080/mcp
func NewHTTPClient(ctx context.Context, serverURL string) (*Client, error) {
	tr, err := transport.NewStreamableHTTP(serverURL)
	if err != nil {
		return nil, fmt.Errorf("create http transport: %w", err)
	}
	if err := tr.Start(ctx); err != nil {
		return nil, fmt.Errorf("start http transport: %w", err)
	}
	cli := client.NewClient(tr)
	return initializeClient(ctx, cli)
}

// NewSSEClient creates an MCP client using SSE transport to the SSE endpoint,
// for example: http://127.0.0.1:8080/sse
func NewSSEClient(ctx context.Context, sseURL string) (*Client, error) {
	tr, err := transport.NewSSE(sseURL)
	if err != nil {
		return nil, fmt.Errorf("create sse transport: %w", err)
	}
	if err := tr.Start(ctx); err != nil {
		return nil, fmt.Errorf("start sse transport: %w", err)
	}
	cli := client.NewClient(tr)
	return initializeClient(ctx, cli)
}

// NewInProcessClient creates an MCP client connected to an in-process server instance.
func NewInProcessClient(
	ctx context.Context,
	searchService *search.Service,
	indexer indexer.Indexer,
) (*Client, error) {
	srv := New(searchService, indexer, ServerConfig{})
	tr := transport.NewInProcessTransport(srv)
	if err := tr.Start(ctx); err != nil {
		return nil, fmt.Errorf("start in-process transport: %w", err)
	}
	cli := client.NewClient(tr)
	return initializeClient(ctx, cli)
}

// initializeClient starts and initializes the MCP client with default capabilities.
func initializeClient(ctx context.Context, cli *client.Client) (*Client, error) {
	ctxStart, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := cli.Start(ctxStart); err != nil {
		return nil, fmt.Errorf("start mcp client: %w", err)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "ts-index", Version: "0.1.0"}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}

	if _, err := cli.Initialize(ctx, initReq); err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("init mcp client: %w", err)
	}
	return &Client{c: cli}, nil
}

func (c *Client) Close() error {
	return c.c.Close()
}

func (c *Client) Call(
	ctx context.Context,
	name string,
	args map[string]any,
) (*mcp.CallToolResult, error) {
	return c.c.CallTool(
		ctx,
		mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: args}},
	)
}

// ListTools returns the list of available tools from the MCP server
func (c *Client) ListTools(
	ctx context.Context,
	req mcp.ListToolsRequest,
) (*mcp.ListToolsResult, error) {
	return c.c.ListTools(ctx, req)
}
