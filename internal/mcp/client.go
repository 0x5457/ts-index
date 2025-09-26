package mcp

import (
	"context"
	"fmt"
	"os"
	"time"

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

	// Use the same executable to launch the MCP server
	tr := transport.NewStdio(exePath, nil, args...)
	if err := tr.Start(ctx); err != nil {
		return nil, fmt.Errorf("start mcp transport: %w", err)
	}
	cli := client.NewClient(tr)

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
