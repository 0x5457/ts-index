package commands

import (
	"fmt"
	"net/http"

	"github.com/0x5457/ts-index/internal/constants"
	"github.com/0x5457/ts-index/internal/factory"
	"github.com/0x5457/ts-index/internal/indexer"
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// NewMCPServeCommand starts an MCP stdio server that exposes search and LSP tools.
func NewMCPServeCommand() *cobra.Command {
	var (
		project   string
		db        string
		embedURL  string
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP server",
		Long:  "Run MCP server, provide search and LSP tools.",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Set default values
			if embedURL == "" {
				embedURL = constants.DefaultEmbedURL
			}

			// Create factory and components in the outer layer
			componentFactory := factory.NewComponentFactory(factory.ComponentConfig{
				DBPath:   db,
				EmbedURL: embedURL,
			})

			var searchService *search.Service
			var indexer indexer.Indexer

			if db != "" {
				components, err := componentFactory.CreateComponents()
				if err != nil {
					return fmt.Errorf("initialize components failed: %w", err)
				}

				searchService = components.Searcher
				indexer = componentFactory.CreateIndexerFromComponents(components)

				// If Project is specified, pre-index it
				if project != "" {
					if err := indexer.IndexProject(project); err != nil {
						return fmt.Errorf("pre-index project failed: %w", err)
					}
				}
			}

			// Create server with injected dependencies
			s := appmcp.New(searchService, indexer)

			switch transport {
			case "stdio":
				return server.ServeStdio(s)
			case "http":
				// Streamable HTTP server on address, default ":8080" if empty
				addr := address
				if addr == "" {
					addr = ":8080"
				}
				httpSrv := server.NewStreamableHTTPServer(s)
				return httpSrv.Start(addr)
			case "sse":
				// SSE server exposes two endpoints; default base path "/mcp"
				addr := address
				if addr == "" {
					addr = ":8080"
				}
				sseSrv := server.NewSSEServer(s,
					server.WithBaseURL(""),
					server.WithStaticBasePath("/mcp"),
				)
				return sseSrv.Start(addr)
			case "http-handler":
				// Advanced: mount handlers on default net/http mux
				// address must be provided
				if address == "" {
					return fmt.Errorf("--address is required for http-handler mode, e.g. :8080")
				}
				sh := server.NewStreamableHTTPServer(s)
				http.Handle("/mcp", sh)
				return http.ListenAndServe(address, nil)
			default:
				return fmt.Errorf(
					"unsupported transport: %s (supported: stdio, http, sse, http-handler)",
					transport,
				)
			}
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "project path")
	cmd.Flags().StringVarP(&db, "db", "d", "", "SQLite database path")
	cmd.Flags().
		StringVar(&embedURL, "embed-url", constants.DefaultEmbedURL, "embed API address")
	cmd.Flags().
		StringVarP(&transport, "transport", "t", "stdio", "transport (stdio, http, sse, http-handler)")
	cmd.Flags().StringVarP(&address, "address", "a", "", "server address (http modes), e.g. :8080")

	return cmd
}
