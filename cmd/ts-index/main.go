package main

import (
	"fmt"
	"log"
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

func main() {
	var (
		project string
		dbPath  string
	)

	embUrl := "http://localhost:8000/embed"
	rootCmd := &cobra.Command{Use: "ts-index"}

	indexCmd := &cobra.Command{
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
	indexCmd.Flags().StringVar(&project, "project", "", "Path to project root")
	indexCmd.Flags().
		StringVar(&dbPath, "db", filepath.Join(os.TempDir(), "ts_index.db"), "SQLite DB path")
	indexCmd.Flags().StringVar(&embUrl, "embed-url", embUrl, "Embedding API URL")

	var topK int
	var symbol bool
	searchCmd := &cobra.Command{
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
	searchCmd.Flags().
		StringVar(&project, "project", "", "Path to project root (optional to build memory index)")
	searchCmd.Flags().
		StringVar(&dbPath, "db", filepath.Join(os.TempDir(), "ts_index.db"), "SQLite DB path")
	searchCmd.Flags().IntVar(&topK, "top-k", 5, "Top K results")
	searchCmd.Flags().BoolVar(&symbol, "symbol", false, "Use exact symbol name search")
	searchCmd.Flags().StringVar(&embUrl, "embed-url", embUrl, "Embedding API URL")

	rootCmd.AddCommand(indexCmd, searchCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
