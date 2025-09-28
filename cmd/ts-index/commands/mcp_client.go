package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/0x5457/ts-index/internal/app/appfx"
	"github.com/0x5457/ts-index/internal/constants"
	"github.com/0x5457/ts-index/internal/indexer"
	appmcp "github.com/0x5457/ts-index/internal/mcp"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/mcp"
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
		newMCPListToolsCommand(),
		newMCPSearchCommand(),
		newMCPLSPCommand(),
		newMCPAstGrepCommand(),
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

Example:
  ts-index mcp-client call semantic_search query="async function" project=/path/to/project`,
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

	cmd.Flags().StringP("language", "l", "typescript", "Programming language")
	cmd.Flags().IntP("max-results", "m", 50, "Maximum number of results")
	cmd.Flags().IntP("context", "c", 0, "Number of context lines")

	return cmd
}

func newMCPListToolsCommand() *cobra.Command {
	var (
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "list-tools",
		Short: "List available MCP tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// Use minimal config for list-tools
			config := appmcp.ServerConfig{}
			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			// Use standard MCP ListTools API
			result, err := client.ListTools(ctx, mcp.ListToolsRequest{})
			if err != nil {
				return fmt.Errorf("failed to list tools: %w", err)
			}

			// Pretty print the tools
			if len(result.Tools) == 0 {
				fmt.Println("No tools available")
				return nil
			}

			fmt.Printf("Available MCP tools (%d):\n\n", len(result.Tools))
			for i, tool := range result.Tools {
				fmt.Printf("%d. %s\n", i+1, tool.Name)
				if tool.Description != "" {
					fmt.Printf("   Description: %s\n", tool.Description)
				}
				if len(tool.InputSchema.Properties) > 0 {
					fmt.Printf("   Parameters:\n")
					for name, prop := range tool.InputSchema.Properties {
						required := ""
						if slices.Contains(tool.InputSchema.Required, name) {
							required = " (required)"
						}
						if propMap, ok := prop.(map[string]any); ok {
							if desc, ok := propMap["description"].(string); ok {
								fmt.Printf("     - %s%s: %s\n", name, required, desc)
							} else {
								fmt.Printf("     - %s%s\n", name, required)
							}
						} else {
							fmt.Printf("     - %s%s\n", name, required)
						}
					}
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().
		StringVarP(&transport, "transport", "t", transportStdio, "transport (stdio, http, sse, inproc)")
	cmd.Flags().
		StringVarP(&address, "address", "a", "", "server URL (http/sse), ignored for stdio/inproc")
	return cmd
}

func newMCPSearchCommand() *cobra.Command {
	var (
		project   string
		db        string
		embedURL  string
		topK      int
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "semantic search code",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]

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

			toolArgs := map[string]any{
				"query": query,
				"top_k": topK,
			}
			if project != "" {
				toolArgs["project"] = project
			}
			if db != "" {
				toolArgs["db"] = db
			}
			if embedURL != "" {
				toolArgs["embed_url"] = embedURL
			}

			result, err := client.Call(ctx, "semantic_search", toolArgs)
			if err != nil {
				return fmt.Errorf("search failed: %w", err)
			}

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
	cmd.Flags().
		StringVar(&embedURL, "embed-url", constants.DefaultEmbedURL, "embed API address")
	cmd.Flags().IntVarP(&topK, "top-k", "k", 5, "number of results")
	cmd.Flags().
		StringVarP(&transport, "transport", "t", transportStdio, "transport (stdio, http, sse, inproc)")
	cmd.Flags().
		StringVarP(&address, "address", "a", "", "server URL (http/sse), ignored for stdio/inproc")

	return cmd
}

func newMCPLSPCommand() *cobra.Command {
	var (
		transport string
		address   string
	)

	cmd := &cobra.Command{
		Use:   "lsp",
		Short: "LSP related operations",
	}

	analyzeCmd := &cobra.Command{
		Use:   "analyze <file> <line> <character>",
		Short: "Analyze the symbol at the specified position",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			line, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid line number: %s", args[1])
			}
			character, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid character position: %s", args[2])
			}

			// Get global project parameter
			project, err := cmd.Flags().GetString("project")
			if err != nil {
				return fmt.Errorf("failed to get project parameter: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project: project,
			}
			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			toolArgs := map[string]any{
				"file":      file,
				"line":      line,
				"character": character,
			}

			result, err := client.Call(ctx, "lsp_analyze", toolArgs)
			if err != nil {
				return fmt.Errorf("analyze failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}

	// Create the new goto commands
	implementationCmd := newMCPLSPGotoCommand(
		"implementation",
		"Find implementations of symbol at position",
		"lsp_implementation",
		&transport, &address,
	)
	typeDefinitionCmd := newMCPLSPGotoCommand(
		"type-definition",
		"Find type definitions of symbol at position",
		"lsp_type_definition",
		&transport, &address,
	)
	declarationCmd := newMCPLSPGotoCommand(
		"declaration",
		"Find declarations of symbol at position",
		"lsp_declaration",
		&transport, &address,
	)

	cmd.AddCommand(analyzeCmd, implementationCmd, typeDefinitionCmd, declarationCmd)

	cmd.PersistentFlags().
		StringVarP(&transport, "transport", "t", transportStdio, "transport (stdio, http, sse, inproc)")
	cmd.PersistentFlags().
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

// astGrepCommandBuilder is a function type for building ast-grep command arguments
type astGrepCommandBuilder func(args []string, cmd *cobra.Command) (toolName string, toolArgs map[string]any, err error)

// newAstGrepCommand creates a generic ast-grep command to reduce code duplication
func newAstGrepCommand(
	use, short string,
	expectedArgs int,
	builder astGrepCommandBuilder,
) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(expectedArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolName, toolArgs, err := builder(args, cmd)
			if err != nil {
				return err
			}

			// Get global flags
			project, _ := cmd.Flags().GetString("project")
			transport, _ := cmd.Flags().GetString("transport")
			address, _ := cmd.Flags().GetString("address")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project: project,
			}

			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			result, err := client.Call(ctx, toolName, toolArgs)
			if err != nil {
				return fmt.Errorf("%s failed: %w", toolName, err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}
}

// newMCPLSPGotoCommand creates a generic LSP goto command for MCP client
func newMCPLSPGotoCommand(
	use, short, mcpTool string,
	transport, address *string,
) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <file> <line> <character>",
		Short: short,
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			file := args[0]
			line, err := strconv.Atoi(args[1])
			if err != nil {
				return fmt.Errorf("invalid line number: %s", args[1])
			}
			character, err := strconv.Atoi(args[2])
			if err != nil {
				return fmt.Errorf("invalid character position: %s", args[2])
			}

			// Get global project parameter
			project, err := cmd.Flags().GetString("project")
			if err != nil {
				return fmt.Errorf("failed to get project parameter: %w", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project: project,
			}
			client, err := createMCPClient(ctx, *transport, *address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			toolArgs := map[string]any{
				"file":      file,
				"line":      line,
				"character": character,
			}

			result, err := client.Call(ctx, mcpTool, toolArgs)
			if err != nil {
				return fmt.Errorf("%s failed: %w", use, err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}
}

// newMCPAstGrepCommand creates ast-grep subcommands
func newMCPAstGrepCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ast-grep",
		Short: "AST-grep structural code search",
		Long:  "Commands for structural code search using ast-grep",
	}

	cmd.AddCommand(
		newAstGrepSearchCommand(),
		newAstGrepRuleCommand(),
		newAstGrepTestCommand(),
		newAstGrepSyntaxTreeCommand(),
	)

	return cmd
}

// newAstGrepSearchCommand creates the ast-grep search command
func newAstGrepSearchCommand() *cobra.Command {
	cmd := newAstGrepCommand("search <pattern>", "Search code patterns using ast-grep", 1,
		func(args []string, cmd *cobra.Command) (string, map[string]any, error) {
			pattern := args[0]
			language, _ := cmd.Flags().GetString("language")
			maxResults, _ := cmd.Flags().GetInt("max-results")
			contextLines, _ := cmd.Flags().GetInt("context")

			toolArgs := map[string]any{
				"pattern":     pattern,
				"language":    language,
				"max_results": maxResults,
				"context":     contextLines,
			}

			return "ast_grep_search", toolArgs, nil
		})

	cmd.Flags().StringP("language", "l", "typescript", "Programming language")
	cmd.Flags().IntP("max-results", "m", 50, "Maximum number of results")
	cmd.Flags().IntP("context", "c", 0, "Number of context lines")

	return cmd
}

// newAstGrepRuleCommand creates the ast-grep rule search command
func newAstGrepRuleCommand() *cobra.Command {
	return newAstGrepCommand("rule <rule>", "Search using ast-grep YAML rule", 1,
		func(args []string, cmd *cobra.Command) (string, map[string]any, error) {
			rule := args[0]
			maxResults, _ := cmd.Flags().GetInt("max-results")

			toolArgs := map[string]any{
				"rule":        rule,
				"max_results": maxResults,
			}

			return "ast_grep_rule", toolArgs, nil
		})
}

// newAstGrepTestCommand creates the ast-grep test command
func newAstGrepTestCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "test <rule> <code>",
		Short: "Test ast-grep rule against code snippet",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			rule := args[0]
			code := args[1]
			language, _ := cmd.Flags().GetString("language")

			// Get global flags
			project, _ := cmd.Flags().GetString("project")
			transport, _ := cmd.Flags().GetString("transport")
			address, _ := cmd.Flags().GetString("address")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project: project,
			}

			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			toolArgs := map[string]any{
				"rule":     rule,
				"code":     code,
				"language": language,
			}

			result, err := client.Call(ctx, "ast_grep_test", toolArgs)
			if err != nil {
				return fmt.Errorf("ast-grep test failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}
}

// newAstGrepSyntaxTreeCommand creates the ast-grep syntax tree command
func newAstGrepSyntaxTreeCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "syntax-tree <code>",
		Short: "Dump syntax tree of code snippet",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			code := args[0]
			language, _ := cmd.Flags().GetString("language")

			// Get global flags
			project, _ := cmd.Flags().GetString("project")
			transport, _ := cmd.Flags().GetString("transport")
			address, _ := cmd.Flags().GetString("address")

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			config := appmcp.ServerConfig{
				Project: project,
			}

			client, err := createMCPClient(ctx, transport, address, config)
			if err != nil {
				return fmt.Errorf("create MCP client failed: %w", err)
			}
			defer client.Close() //nolint:errcheck

			toolArgs := map[string]any{
				"code":     code,
				"language": language,
			}

			result, err := client.Call(ctx, "ast_grep_syntax_tree", toolArgs)
			if err != nil {
				return fmt.Errorf("ast-grep syntax tree failed: %w", err)
			}

			output, err := json.MarshalIndent(result, "", "  ")
			if err != nil {
				return fmt.Errorf("format result failed: %w", err)
			}
			fmt.Println(string(output))
			return nil
		},
	}
}
