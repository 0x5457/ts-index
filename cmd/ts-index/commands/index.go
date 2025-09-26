package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/constants"
	"github.com/0x5457/ts-index/internal/factory"
	"github.com/spf13/cobra"
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

			// Create component factory
			componentFactory := factory.NewComponentFactory(factory.ComponentConfig{
				DBPath:   dbPath,
				EmbedURL: embUrl,
			})

			// Create components
			components, err := componentFactory.CreateComponents()
			if err != nil {
				return fmt.Errorf("failed to create components: %w", err)
			}
			defer func() {
				if cleanupErr := components.Cleanup(); cleanupErr != nil {
					fmt.Printf("failed to cleanup components: %v\n", cleanupErr)
				}
			}()

			// Create indexer
			idx := componentFactory.CreateIndexerFromComponents(components)

			// Run indexing with progress
			progCh, errCh := idx.IndexProjectProgress(cmd.Context(), project)
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
				case <-cmd.Context().Done():
					fmt.Println()
					return cmd.Context().Err()
				}
			}
			fmt.Println()
			fmt.Println("index completed")
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
