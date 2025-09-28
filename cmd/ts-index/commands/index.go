package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/cmd/cmdsfx"
	"github.com/0x5457/ts-index/internal/app/appfx"
	"github.com/0x5457/ts-index/internal/constants"
	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

func NewIndexCommand() *cobra.Command {
	var (
		project string
		dbPath  string
		embUrl  string
	)

	cmd := &cobra.Command{
		Use:   "index",
		Short: "Index a TypeScript project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			// Create Fx app with configuration
			app := fx.New(
				appfx.Module,
				fx.Supply(
					fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
					fx.Annotate(embUrl, fx.ResultTags(`name:"embedURL"`)),
					fx.Annotate("", fx.ResultTags(`name:"project"`)),
				),
				fx.Invoke(func(runner *cmdsfx.CommandRunner) error {
					return runner.RunIndex(cmd.Context(), project)
				}),
			)

			// Start and wait for the app to finish
			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			if err := app.Start(ctx); err != nil {
				return fmt.Errorf("failed to start application: %w", err)
			}

			ctx, cancel = context.WithTimeout(context.Background(), fx.DefaultTimeout)
			defer cancel()

			if err := app.Stop(ctx); err != nil {
				return fmt.Errorf("failed to stop application: %w", err)
			}

			return nil
		},
	}

	defaultEmbUrl := constants.DefaultEmbedURL
	defaultDbPath := filepath.Join(os.TempDir(), "ts_index.db")

	cmd.Flags().StringVar(&project, "project", "", "Path to project root")
	cmd.Flags().StringVar(&dbPath, "db", defaultDbPath, "SQLite DB path")
	cmd.Flags().StringVar(&embUrl, "embed-url", defaultEmbUrl, "Embedding API URL")

	return cmd
}
