package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/lsp"
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

	// LSP commands
	lspCmd := &cobra.Command{
		Use:   "lsp",
		Short: "Language Server Protocol commands",
	}

	// LSP info command
	lspInfoCmd := &cobra.Command{
		Use:   "info",
		Short: "Show LSP server information",
		RunE: func(cmd *cobra.Command, args []string) error {
			clientTools := lsp.NewClientTools()
			defer clientTools.Cleanup()

			fmt.Println("Registered Language Server Adapters:")
			adapters := clientTools.GetAdapterInfo()
			for _, adapter := range adapters {
				status := "❌ Not Installed"
				if adapter.IsInstalled {
					status = "✅ Installed"
				}
				fmt.Printf("  %s (%s): %s\n", adapter.Language, adapter.Name, status)
			}

			fmt.Println("\nRunning Language Servers:")
			servers := clientTools.GetServerInfo()
			if len(servers) == 0 {
				fmt.Println("  None")
			} else {
				for _, server := range servers {
					fmt.Printf("  %s: %s (%s)\n", server.Name, server.WorkspaceRoot, server.AdapterName)
				}
			}

			return nil
		},
	}

	// LSP analyze command
	var (
		lspLine        int
		lspCharacter   int
		includeHover   bool
		includeRefs    bool
		includeDefs    bool
	)
	lspAnalyzeCmd := &cobra.Command{
		Use:   "analyze [file-path]",
		Short: "Analyze symbol at position using LSP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			clientTools := lsp.NewClientTools()
			defer clientTools.Cleanup()

			req := lsp.AnalyzeSymbolRequest{
				WorkspaceRoot: project,
				FilePath:      args[0],
				Line:          lspLine,
				Character:     lspCharacter,
				IncludeHover:  includeHover,
				IncludeRefs:   includeRefs,
				IncludeDefs:   includeDefs,
			}

			result := clientTools.AnalyzeSymbol(cmd.Context(), req)
			
			// Convert to JSON for output
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			
			fmt.Println(string(data))
			return nil
		},
	}
	lspAnalyzeCmd.Flags().StringVar(&project, "project", "", "Path to project root")
	lspAnalyzeCmd.Flags().IntVar(&lspLine, "line", 0, "Line number (0-based)")
	lspAnalyzeCmd.Flags().IntVar(&lspCharacter, "character", 0, "Character number (0-based)")
	lspAnalyzeCmd.Flags().BoolVar(&includeHover, "hover", true, "Include hover information")
	lspAnalyzeCmd.Flags().BoolVar(&includeRefs, "refs", false, "Include references")
	lspAnalyzeCmd.Flags().BoolVar(&includeDefs, "defs", true, "Include definitions")

	// LSP completion command
	var maxResults int
	lspCompletionCmd := &cobra.Command{
		Use:   "completion [file-path]",
		Short: "Get completion items at position using LSP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

			clientTools := lsp.NewClientTools()
			defer clientTools.Cleanup()

			req := lsp.CompletionRequest{
				WorkspaceRoot: project,
				FilePath:      args[0],
				Line:          lspLine,
				Character:     lspCharacter,
				MaxResults:    maxResults,
			}

			result := clientTools.GetCompletion(cmd.Context(), req)
			
			// Convert to JSON for output
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			
			fmt.Println(string(data))
			return nil
		},
	}
	lspCompletionCmd.Flags().StringVar(&project, "project", "", "Path to project root")
	lspCompletionCmd.Flags().IntVar(&lspLine, "line", 0, "Line number (0-based)")
	lspCompletionCmd.Flags().IntVar(&lspCharacter, "character", 0, "Character number (0-based)")
	lspCompletionCmd.Flags().IntVar(&maxResults, "max-results", 20, "Maximum number of results")

	// LSP symbol search command
	var query string
	lspSymbolCmd := &cobra.Command{
		Use:   "symbols",
		Short: "Search workspace symbols using LSP",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if query == "" {
				return fmt.Errorf("--query is required")
			}

			clientTools := lsp.NewClientTools()
			defer clientTools.Cleanup()

			req := lsp.SymbolSearchRequest{
				WorkspaceRoot: project,
				Query:         query,
				MaxResults:    maxResults,
			}

			result := clientTools.SearchSymbols(cmd.Context(), req)
			
			// Convert to JSON for output
			data, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return err
			}
			
			fmt.Println(string(data))
			return nil
		},
	}
	lspSymbolCmd.Flags().StringVar(&project, "project", "", "Path to project root")
	lspSymbolCmd.Flags().StringVar(&query, "query", "", "Search query")
	lspSymbolCmd.Flags().IntVar(&maxResults, "max-results", 50, "Maximum number of results")

	// LSP health command
	lspHealthCmd := &cobra.Command{
		Use:   "health",
		Short: "Check LSP health and language server availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			if lsp.IsVTSLSInstalled() {
				fmt.Println("✓ vtsls is installed and available")
			} else {
				fmt.Println("✗ vtsls is not installed")
				fmt.Printf("Install with: %s\n", lsp.InstallVTSLSCommand())
			}
			
			if lsp.IsTypeScriptLanguageServerInstalled() {
				fmt.Println("✓ typescript-language-server is installed and available")
			} else {
				fmt.Println("✗ typescript-language-server is not installed")
				fmt.Printf("Install with: %s\n", lsp.InstallTypeScriptLanguageServerCommand())
			}
			
			if !lsp.IsVTSLSInstalled() && !lsp.IsTypeScriptLanguageServerInstalled() {
				fmt.Println("\n⚠️  No TypeScript language servers are available")
				fmt.Println("Please install at least one of the above language servers to use LSP functionality")
			}
			
			return nil
		},
	}

	lspCmd.AddCommand(lspInfoCmd, lspAnalyzeCmd, lspCompletionCmd, lspSymbolCmd, lspHealthCmd)
	rootCmd.AddCommand(indexCmd, searchCmd, lspCmd)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
