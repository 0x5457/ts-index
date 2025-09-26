package mcp

import (
    "context"
    "fmt"
    "time"

    "github.com/mark3labs/mcp-go/client"
    "github.com/mark3labs/mcp-go/client/transport"
    "github.com/mark3labs/mcp-go/mcp"
)

// Client wraps an MCP stdio client aimed at our own server executable.
type Client struct { c *client.Client }

// NewStdioClient creates and initializes an MCP client that launches this binary with serve-mcp.
func NewStdioClient(ctx context.Context) (*Client, error) {
    tr := transport.NewStdio("ts-index", nil, "serve-mcp")
    cli := client.NewClient(tr)

    ctxStart, cancel := context.WithTimeout(ctx, 10*time.Second)
    defer cancel()
    if err := cli.Start(ctxStart); err != nil {
        return nil, fmt.Errorf("start mcp client: %w", err)
    }

    initReq := mcp.InitializeRequest{}
    initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
    initReq.Params.ClientInfo = mcp.Implementation{Name: "ts-index-cli", Version: "0.1.0"}
    initReq.Params.Capabilities = mcp.ClientCapabilities{}

    if _, err := cli.Initialize(ctx, initReq); err != nil {
        _ = cli.Close()
        return nil, fmt.Errorf("init mcp client: %w", err)
    }

    return &Client{c: cli}, nil
}

func (c *Client) Close() error { return c.c.Close() }

func (c *Client) Call(ctx context.Context, name string, args map[string]any) (*mcp.CallToolResult, error) {
    return c.c.CallTool(ctx, mcp.CallToolRequest{Params: mcp.CallToolParams{Name: name, Arguments: args}})
}

