package cmdsfx

import (
	"context"
	"fmt"

	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// CommandRunner provides methods to run different application commands
type CommandRunner struct {
	config        *configfx.Config
	searchService *search.Service
	indexer       indexer.Indexer
	mcpServer     *server.MCPServer
}

// Params represents dependencies for command runner
type Params struct {
	fx.In

	Config        *configfx.Config
	SearchService *search.Service   `optional:"true"`
	Indexer       indexer.Indexer   `optional:"true"`
	MCPServer     *server.MCPServer `optional:"true"`
}

// NewCommandRunner creates a new command runner
func NewCommandRunner(params Params) *CommandRunner {
	return &CommandRunner{
		config:        params.Config,
		searchService: params.SearchService,
		indexer:       params.Indexer,
		mcpServer:     params.MCPServer,
	}
}

// RunIndex executes the index command
func (r *CommandRunner) RunIndex(ctx context.Context, projectPath string) error {
	if r.indexer == nil {
		return fmt.Errorf("indexer not available")
	}

	// Run indexing with progress
	progCh, errCh := r.indexer.IndexProjectProgress(ctx, projectPath)
	for progCh != nil || errCh != nil {
		select {
		case p, ok := <-progCh:
			if !ok {
				progCh = nil
				continue
			}
			fmt.Printf("\r[%3.0f%%] stage=%s files:%d/%d chunks:%d/%d %-40s",
				p.Percent*100,
				p.Stage,
				p.ParsedFiles, p.TotalFiles,
				p.EmbeddedChunks, p.TotalChunks,
				p.CurrentFile,
			)
		case err, ok := <-errCh:
			if !ok {
				errCh = nil
				continue
			}
			if err != nil {
				fmt.Println()
				return err
			}
		case <-ctx.Done():
			fmt.Println()
			return ctx.Err()
		}
	}
	fmt.Println()
	fmt.Println("index completed")
	return nil
}

// RunSearch executes semantic search
func (r *CommandRunner) RunSearch(ctx context.Context, query string, topK int) error {
	if r.searchService == nil {
		return fmt.Errorf("search service not available")
	}

	hits, err := r.searchService.Search(ctx, query, topK)
	if err != nil {
		return err
	}

	// Print results (you can customize the output format)
	for i, hit := range hits {
		fmt.Printf("Result %d (score: %.4f):\n", i+1, hit.Score)
		fmt.Printf("File: %s\n", hit.Chunk.File)
		fmt.Printf("Lines: %d-%d\n", hit.Chunk.StartLine, hit.Chunk.EndLine)
		fmt.Printf("Content: %s\n\n", hit.Chunk.Content)
	}

	return nil
}

// RunMCPServer executes the MCP server
func (r *CommandRunner) RunMCPServer(transport, address string) error {
	if r.mcpServer == nil {
		return fmt.Errorf("MCP server not available")
	}

	switch transport {
	case "stdio":
		return server.ServeStdio(r.mcpServer)
	case "http":
		// Streamable HTTP server on address, default ":8080" if empty
		addr := address
		if addr == "" {
			addr = ":8080"
		}
		httpSrv := server.NewStreamableHTTPServer(r.mcpServer)
		return httpSrv.Start(addr)
	case "sse":
		// SSE server exposes two endpoints; default base path "/mcp"
		addr := address
		if addr == "" {
			addr = ":8080"
		}
		sseSrv := server.NewSSEServer(r.mcpServer,
			server.WithBaseURL(""),
			server.WithStaticBasePath("/mcp"),
		)
		return sseSrv.Start(addr)
	default:
		return fmt.Errorf(
			"unsupported transport: %s (supported: stdio, http, sse)",
			transport,
		)
	}
}

// Module provides command runner
var Module = fx.Module("commands",
	fx.Provide(NewCommandRunner),
)
