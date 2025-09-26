package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
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
			p := tsparser.New()
			emb := embeddings.NewApi(embUrl)
			sym, err := sqlite.New(dbPath)
			if err != nil {
				return err
			}
			// dimension is inferred at first insert if set to 0
			vecStore, err := sqlvec.New(dbPath, 0)
			if err != nil {
				return err
			}
			vec := vecStore
			idx := pipeline.New(p, emb, sym, vec, pipeline.Options{})
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

	defaultEmbUrl := "http://localhost:8000/embed"
	defaultDbPath := filepath.Join(os.TempDir(), "ts_index.db")

	cmd.Flags().StringVar(&project, "project", "", "Path to project root")
	cmd.Flags().StringVar(&dbPath, "db", defaultDbPath, "SQLite DB path")
	cmd.Flags().StringVar(&embUrl, "embed-url", defaultEmbUrl, "Embedding API URL")

	return cmd
}
