package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
	"github.com/spf13/cobra"
)

func NewSearchCommand() *cobra.Command {
	var (
		project string
		dbPath  string
		embUrl  string
		topK    int
		symbol  bool
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search code: semantic (default) or exact symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			p := tsparser.New()
			emb := embeddings.NewApi(embUrl)
			sym, err := sqlite.New(dbPath)
			if err != nil {
				return err
			}
			vecStore, err := sqlvec.New(dbPath, 0)
			if err != nil {
				return err
			}
			vec := vecStore
			idx := pipeline.New(p, emb, sym, vec, pipeline.Options{})

			if symbol {
				// exact symbol search
				hits, err := idx.SearchSymbol(query)
				if err != nil {
					return err
				}
				for _, h := range hits {
					fmt.Printf(
						"%s %s:%d-%d\n",
						h.Symbol.Name,
						h.Symbol.File,
						h.Symbol.StartLine,
						h.Symbol.EndLine,
					)
				}
				return nil
			}

			// semantic search
			if project != "" {
				if err := idx.IndexProject(project); err != nil {
					return err
				}
			}
			svc := &search.Service{
				Embedder: emb,
				Vector:   vec,
			}
			hits, err := svc.Search(cmd.Context(), query, topK)
			if err != nil {
				return err
			}
			for _, hit := range hits {
				fmt.Printf(
					"[%.3f] %s %s:%d-%d\n",
					hit.Score,
					hit.Chunk.Name,
					hit.Chunk.File,
					hit.Chunk.StartLine,
					hit.Chunk.EndLine,
				)
			}
			return nil
		},
	}

	defaultEmbUrl := "http://localhost:8000/embed"
	defaultDbPath := filepath.Join(os.TempDir(), "ts_index.db")

	cmd.Flags().
		StringVar(&project, "project", "", "Path to project root (optional to build memory index)")
	cmd.Flags().StringVar(&dbPath, "db", defaultDbPath, "SQLite DB path")
	cmd.Flags().IntVar(&topK, "top-k", 5, "Top K results")
	cmd.Flags().BoolVar(&symbol, "symbol", false, "Use exact symbol name search")
	cmd.Flags().StringVar(&embUrl, "embed-url", defaultEmbUrl, "Embedding API URL")

	return cmd
}
