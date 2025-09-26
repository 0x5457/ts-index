package commands

import (
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
)

// NewMCPServeCommand starts an MCP stdio server that exposes search and LSP tools.
func NewMCPServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp",
		Short: "Run MCP stdio server",
		RunE: func(cmd *cobra.Command, args []string) error {
			s := appmcp.New()
			return server.ServeStdio(s)
		},
	}
	return cmd
}
