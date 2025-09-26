package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0x5457/ts-index/internal/constants"
	mcpclient "github.com/0x5457/ts-index/internal/mcp"
	"github.com/spf13/cobra"
)

func NewSearchCommand() *cobra.Command {
	var (
		project   string
		dbPath    string
		embUrl    string
		topK      int
		symbol    bool
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "search [query]",
		Short: "Search code: semantic (default) or exact symbol",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			// choose transport
			var cli *mcpclient.Client
			var err error
			switch transport {
			case "", "stdio":
				cli, err = mcpclient.NewStdioClient(cmd.Context())
			case "http":
				addr := address
				if addr == "" {
					addr = "http://127.0.0.1:8080/mcp"
				}
				cli, err = mcpclient.NewHTTPClient(cmd.Context(), addr)
			case "sse":
				addr := address
				if addr == "" {
					addr = "http://127.0.0.1:8080/mcp/sse"
				}
				cli, err = mcpclient.NewSSEClient(cmd.Context(), addr)
			default:
				return fmt.Errorf(
					"unsupported transport: %s (supported: stdio, http, sse)",
					transport,
				)
			}
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()

			if symbol {
				res, err := cli.Call(cmd.Context(), "symbol_search", map[string]any{
					"name": query,
					"db":   dbPath,
				})
				if err != nil {
					return err
				}
				if res.IsError {
					b, _ := json.Marshal(res.StructuredContent)
					return fmt.Errorf("%s", string(b))
				}
				b, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
				fmt.Println(string(b))
				return nil
			}

			res, err := cli.Call(cmd.Context(), "semantic_search", map[string]any{
				"query":     query,
				"db":        dbPath,
				"embed_url": embUrl,
				"top_k":     topK,
				"project":   project,
			})
			if err != nil {
				return err
			}
			if res.IsError {
				b, _ := json.Marshal(res.StructuredContent)
				return fmt.Errorf("%s", string(b))
			}
			b, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(b))
			return nil
		},
	}

	defaultEmbUrl := constants.DefaultEmbedURL
	defaultDbPath := filepath.Join(os.TempDir(), "ts_index.db")

	cmd.Flags().
		StringVar(&project, "project", "", "Path to project root (optional to build memory index)")
	cmd.Flags().StringVar(&dbPath, "db", defaultDbPath, "SQLite DB path")
	cmd.Flags().IntVar(&topK, "top-k", 5, "Top K results")
	cmd.Flags().BoolVar(&symbol, "symbol", false, "Use exact symbol name search")
	cmd.Flags().StringVar(&embUrl, "embed-url", defaultEmbUrl, "Embedding API URL")
	cmd.Flags().StringVarP(&transport, "transport", "t", "stdio", "transport (stdio, http, sse)")
	cmd.Flags().StringVarP(&address, "address", "a", "", "server URL (http/sse)")

	return cmd
}
