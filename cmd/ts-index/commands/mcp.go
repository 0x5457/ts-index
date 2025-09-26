package commands

import (
	"fmt"
	"net/http"

	appmcp "github.com/0x5457/ts-index/internal/mcp"
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
			opts := appmcp.ServerOptions{
				Project:  project,
				DB:       db,
				EmbedURL: embedURL,
			}
			s := appmcp.NewWithOptions(opts)

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
		StringVar(&embedURL, "embed-url", "http://localhost:8000/embed", "embed API address")
	cmd.Flags().
		StringVarP(&transport, "transport", "t", "stdio", "transport (stdio, http, sse, http-handler)")
	cmd.Flags().StringVarP(&address, "address", "a", "", "server address (http modes), e.g. :8080")

	return cmd
}
