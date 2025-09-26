package mcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// TestStreamableHTTPTransport verifies initialize and list-tools via streamable-http
func TestStreamableHTTPTransport(t *testing.T) {
	s := New(nil, nil)
	h := server.NewStreamableHTTPServer(s)
	ts := httptest.NewServer(h)
	t.Cleanup(ts.Close)

	cliTr, err := transport.NewStreamableHTTP(ts.URL)
	if err != nil {
		t.Fatalf("new streamable http: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	if err := cliTr.Start(ctx); err != nil {
		t.Fatalf("start transport: %v", err)
	}
	cli := client.NewClient(cliTr)

	if err := cli.Start(ctx); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() { _ = cli.Close() }()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "0.0.1"}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	res, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatalf("expected some tools")
	}
}

// TestSSETransport verifies initialize and list-tools via SSE
func TestSSETransport(t *testing.T) {
	s := New(nil, nil)
	sse := server.NewSSEServer(s,
		server.WithStaticBasePath("/mcp"),
	)
	mux := http.NewServeMux()
	mux.Handle("/mcp/sse", sse.SSEHandler())
	mux.Handle("/mcp/message", sse.MessageHandler())
	ts := httptest.NewServer(mux)
	t.Cleanup(ts.Close)

	cliTr, err := transport.NewSSE(ts.URL + "/mcp/sse")
	if err != nil {
		t.Fatalf("new sse: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	t.Cleanup(cancel)
	cli := client.NewClient(cliTr)
	if err := cli.Start(ctx); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() { _ = cli.Close() }()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "0.0.1"}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	res, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatalf("expected some tools")
	}
}

// TestInProcessTransport verifies initialize and list-tools via in-process
func TestInProcessTransport(t *testing.T) {
	s := New(nil, nil)
	tr := transport.NewInProcessTransport(s)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	if err := tr.Start(ctx); err != nil {
		t.Fatalf("start inproc: %v", err)
	}
	cli := client.NewClient(tr)
	if err := cli.Start(ctx); err != nil {
		t.Fatalf("start client: %v", err)
	}
	defer func() { _ = cli.Close() }()

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{Name: "test", Version: "0.0.1"}
	initReq.Params.Capabilities = mcp.ClientCapabilities{}
	if _, err := cli.Initialize(ctx, initReq); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	res, err := cli.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(res.Tools) == 0 {
		t.Fatalf("expected some tools")
	}
}
