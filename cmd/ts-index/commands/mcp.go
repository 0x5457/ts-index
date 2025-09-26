package commands

import (
	"context"
	"fmt"
	"net/http"

	"github.com/0x5457/ts-index/cmd/cmdsfx"
	"github.com/0x5457/ts-index/internal/app/appfx"
	"github.com/0x5457/ts-index/internal/constants"
	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
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

			// Create result channel for server errors
			resultCh := make(chan error, 1)

			// Create Fx app with configuration
			app := fx.New(
				appfx.Module,
				fx.Supply(
					fx.Annotate(db, fx.ResultTags(`name:"dbPath"`)),
					fx.Annotate(embedURL, fx.ResultTags(`name:"embedURL"`)),
					fx.Annotate(project, fx.ResultTags(`name:"project"`)),
				),
				fx.Invoke(func(lc fx.Lifecycle, runner *cmdsfx.CommandRunner) {
					lc.Append(fx.Hook{
						OnStart: func(ctx context.Context) error {
							go func() {
								resultCh <- runner.RunMCPServer(transport, address)
							}()
							return nil
						},
					})
				}),
			)

			// Handle http-handler case separately as it needs special handling
			if transport == "http-handler" {
				if address == "" {
					return fmt.Errorf("--address is required for http-handler mode, e.g. :8080")
				}

				// For http-handler, we need to register the handler during app construction
				app = fx.New(
					appfx.Module,
					fx.Supply(
						fx.Annotate(db, fx.ResultTags(`name:"dbPath"`)),
						fx.Annotate(embedURL, fx.ResultTags(`name:"embedURL"`)),
						fx.Annotate(project, fx.ResultTags(`name:"project"`)),
					),
					fx.Invoke(func(srv *server.MCPServer) {
						sh := server.NewStreamableHTTPServer(srv)
						http.Handle("/mcp", sh)
					}),
				)

				ctx, cancel := context.WithCancel(cmd.Context())
				defer cancel()

				if err := app.Start(ctx); err != nil {
					return fmt.Errorf("failed to start application: %w", err)
				}

				return http.ListenAndServe(address, nil)
			}

			// Start the app
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			if err := app.Start(ctx); err != nil {
				return fmt.Errorf("failed to start application: %w", err)
			}

			// Wait for server result or context cancellation
			select {
			case err := <-resultCh:
				return err
			case <-cmd.Context().Done():
				return cmd.Context().Err()
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
