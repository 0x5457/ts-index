package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/featurizer"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser/tsparser"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage/memory"
	"github.com/0x5457/ts-index/internal/storage/sqlite"

	"github.com/spf13/cobra"
)

func main() {
	var (
		project string
		dbPath  string
	)

	rootCmd := &cobra.Command{Use: "ts-index"}

	indexCmd := &cobra.Command{
		Use:   "index",
		Short: "Index a TypeScript project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			p := tsparser.New()
			emb := embeddings.NewLocal(256)
			sym, err := sqlite.New(dbPath)
			if err != nil {
				return err
			}
			vec := memory.NewInMemoryVectorStore()
			idx := pipeline.New(p, emb, sym, vec, pipeline.Options{})
			if err := idx.IndexProject(project); err != nil {
				return err
			}
			fmt.Println("index completed")
			return nil
		},
	}
	indexCmd.Flags().StringVar(&project, "project", "", "Path to project root")
	indexCmd.Flags().
		StringVar(&dbPath, "db", filepath.Join(os.TempDir(), "ts_index.db"), "SQLite DB path")

	searchSymCmd := &cobra.Command{
		Use:   "search-symbol [name]",
		Short: "Search symbol by exact name",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			p := tsparser.New()
			emb := embeddings.NewLocal(256)
			sym, err := sqlite.New(dbPath)
			if err != nil {
				return err
			}
			vec := memory.NewInMemoryVectorStore()
			idx := pipeline.New(p, emb, sym, vec, pipeline.Options{})
			// no index build here; assumes DB already populated
			hits, err := idx.SearchSymbol(name)
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
		},
	}
	searchSymCmd.Flags().
		StringVar(&dbPath, "db", filepath.Join(os.TempDir(), "ts_index.db"), "SQLite DB path")

	var topK int
	searchSemCmd := &cobra.Command{
		Use:   "search-semantic [query]",
		Short: "Semantic search code chunks",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			p := tsparser.New()
			emb := embeddings.NewLocal(256)
			sym, err := sqlite.New(dbPath)
			if err != nil {
				return err
			}
			vec := memory.NewInMemoryVectorStore()
			idx := pipeline.New(p, emb, sym, vec, pipeline.Options{})
			// no persistence for vector store in PoC; re-index to fill memory
			if project != "" {
				if err := idx.IndexProject(project); err != nil {
					return err
				}
			}
			fz := &featurizer.Featurizer{
				SystemPrompt:  "You are a feature extractor",
				MessagePrefix: "Analyze the query and output boolean features",
				Features: []featurizer.Feature{
					{Identifier: "has_code", Description: "query includes code or api names"},
					{Identifier: "needs_debug", Description: "query implies debugging needed"},
				},
				CreateLLM: func(cfg featurizer.LLMConfig) featurizer.LLM { return &featurizer.MockLLM{} },
			}
			svc := &search.Service{
				Embedder:   emb,
				Vector:     vec,
				Featurizer: fz,
				LLMConfig:  featurizer.LLMConfig{Model: "mock"},
			}
			enriched, _, err := svc.SearchWithFeatures(cmd.Context(), query, topK, 2, 0.7)
			if err != nil {
				return err
			}
			for _, e := range enriched {
				fmt.Printf(
					"[%.3f] %s %s:%d-%d features=%v\n",
					e.Hit.Score,
					e.Hit.Chunk.Name,
					e.Hit.Chunk.File,
					e.Hit.Chunk.StartLine,
					e.Hit.Chunk.EndLine,
					e.Features,
				)
			}
			return nil
		},
	}
	searchSemCmd.Flags().
		StringVar(&project, "project", "", "Path to project root (optional to build memory index)")
	searchSemCmd.Flags().
		StringVar(&dbPath, "db", filepath.Join(os.TempDir(), "ts_index.db"), "SQLite DB path")
	searchSemCmd.Flags().IntVar(&topK, "top-k", 5, "Top K results")

	rootCmd.AddCommand(indexCmd, searchSymCmd, searchSemCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
