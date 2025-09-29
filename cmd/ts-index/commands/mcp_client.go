package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0x5457/ts-index/internal/app/appfx"
	"github.com/0x5457/ts-index/internal/constants"
	"github.com/0x5457/ts-index/internal/indexer"
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

const (
	transportStdio  = "stdio"
	transportHTTP   = "http"
	transportSSE    = "sse"
	transportInproc = "inproc"
)

// NewMCPClientCommand creates commands for connecting to and interacting with MCP servers
func NewMCPClientCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "mcp-client",
		Short: "MCP client commands",
		Long:  "Commands for connecting to and interacting with MCP servers",
	}

	cmd.AddCommand(
		newMCPCallCommand(),
	)

	// Add global flags that will be inherited by all subcommands
	cmd.PersistentFlags().
		StringP("project", "p", "", "project path (will be used as server configuration)")

	return cmd
}

func newMCPCallCommand() *cobra.Command {
	var (
		project   string
		db        string
		embedURL  string
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "call <tool_name> [args...]",
		Short: "Call a specific MCP tool",
		Long: `Call a specific MCP tool with arguments.
Arguments should be provided as key=value pairs.

Examples:
  # Semantic search
  ts-index mcp-client call semantic_search query="async function" top_k=5

  # LSP analyze
  ts-index mcp-client call lsp_analyze file="src/index.ts" line=10 character=5

  # AST grep search
  ts-index mcp-client call ast_grep_search pattern="function $$name" language="typescript"

  # List available tools
  ts-index mcp-client call --list-tools`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName := args[0]
			toolArgs := make(map[string]any)

			// Parse key=value arguments
			for _, arg := range args[1:] {
				parts := strings.SplitN(arg, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid argument format: %s (expected key=value)", arg)
				}
				key, value := parts[0], parts[1]

				// Try to parse as number, bool, or keep as string
				if val, err := strconv.Atoi(value); err == nil {
					toolArgs[key] = val
				} else if val, err := strconv.ParseBool(value); err == nil {
					toolArgs[key] = val
				} else {
					toolArgs[key] = value
				}
			}

			// Add global options to args if not already specified
			if project != "" && toolArgs["project"] == nil {
				toolArgs["project"] = project
			}
			if db != "" && toolArgs["db"] == nil {
				toolArgs["db"] = db
			}
			if embedURL != "" && toolArgs["embed_url"] == nil {
				toolArgs["embed_url"] = embedURL
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project:  project,
				DB:       db,
				EmbedURL: embedURL,
			}
			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			result, err := client.Call(ctx, toolName, toolArgs)
			if err != nil {
				return fmt.Errorf("call tool failed: %w", err)
			}

			// Pretty print result
			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}

	cmd.Flags().StringVarP(&project, "project", "p", "", "project path")
	cmd.Flags().StringVarP(&db, "db", "d", "", "SQLite database path")
	cmd.Flags().StringVar(&embedURL, "embed-url", constants.DefaultEmbedURL, "embed API address")
	cmd.Flags().
		StringVarP(&transport, "transport", "t", transportStdio, "transport (stdio, http, sse, inproc)")
	cmd.Flags().
		StringVarP(&address, "address", "a", "", "server URL (http/sse), ignored for stdio/inproc")

	return cmd
}

func createMCPClient(
	ctx context.Context,
	transport, address string,
	config appmcp.ServerConfig,
) (*appmcp.Client, error) {
	switch transport {
	case transportStdio:
		return appmcp.NewStdioClientWithConfig(ctx, config)
	case transportHTTP:
		if address == "" {
			address = "http://127.0.0.1:8080/mcp"
		}
		return appmcp.NewHTTPClient(ctx, address)
	case transportSSE:
		if address == "" {
			address = "http://127.0.0.1:8080/mcp/sse"
		}
		return appmcp.NewSSEClient(ctx, address)
	case transportInproc:
		var searchService *search.Service
		var indexer indexer.Indexer
		if config.DB != "" {
			// Create Fx app to get components
			app := fx.New(
				appfx.Module,
				fx.Supply(
					fx.Annotate(config.DB, fx.ResultTags(`name:"dbPath"`)),
					fx.Annotate(config.EmbedURL, fx.ResultTags(`name:"embedURL"`)),
					fx.Annotate(config.Project, fx.ResultTags(`name:"project"`)),
				),
				fx.Populate(&searchService, &indexer),
			)
			if err := app.Start(ctx); err != nil {
				return nil, fmt.Errorf("initialize components failed: %w", err)
			}
			// Note: We don't stop the app here as it's needed for the client lifetime
		}
		return appmcp.NewInProcessClient(ctx, searchService, indexer)
	default:
		return nil, fmt.Errorf(
			"unsupported transport: %s (supported: stdio, http, sse, inproc)",
			transport,
		)
	}
}
