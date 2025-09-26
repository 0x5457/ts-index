package commands

import (
	"fmt"

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

			if transport != "stdio" {
				return fmt.Errorf(
					"only stdio transport is supported, other transports (%s) are not implemented",
					transport,
				)
			}

			return server.ServeStdio(s)
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "project path")
	cmd.Flags().StringVarP(&db, "db", "d", "", "SQLite database path")
	cmd.Flags().
		StringVar(&embedURL, "embed-url", "http://localhost:8000/embed", "embed API address")
	cmd.Flags().
		StringVarP(&transport, "transport", "t", "stdio", "transport (stdio, tcp, websocket)")
	cmd.Flags().StringVarP(&address, "address", "a", "", "server address (TCP/WebSocket mode)")

	return cmd
}
