package commands

import (
    "encoding/json"
    "fmt"

    "github.com/0x5457/ts-index/internal/lsp"
    mcpclient "github.com/0x5457/ts-index/internal/mcp"
    "github.com/spf13/cobra"
)

func NewLSPCommand() *cobra.Command {
	lspCmd := &cobra.Command{
		Use:   "lsp",
		Short: "Language Server Protocol commands",
	}

	lspCmd.AddCommand(
		newLSPInfoCommand(),
		newLSPAnalyzeCommand(),
		newLSPCompletionCommand(),
		newLSPSymbolCommand(),
		newLSPInstallCommand(),
		newLSPInstallByLanguageCommand(),
		newLSPListCommand(),
		newLSPHealthCommand(),
	)

	return lspCmd
}

func newLSPInfoCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "info",
		Short: "Show LSP server information",
		RunE: func(cmd *cobra.Command, args []string) error {
            cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_info", nil)
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}
}

func newLSPAnalyzeCommand() *cobra.Command {
	var (
		project      string
		lspLine      int
		lspCharacter int
		includeHover bool
		includeRefs  bool
		includeDefs  bool
	)

	cmd := &cobra.Command{
		Use:   "analyze [file-path]",
		Short: "Analyze symbol at position using LSP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

            cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_analyze", map[string]any{
				"project":   project,
				"file":      args[0],
				"line":      lspLine,
				"character": lspCharacter,
				"hover":     includeHover,
				"refs":      includeRefs,
				"defs":      includeDefs,
			})
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Path to project root")
	cmd.Flags().IntVar(&lspLine, "line", 0, "Line number (0-based)")
	cmd.Flags().IntVar(&lspCharacter, "character", 0, "Character number (0-based)")
	cmd.Flags().BoolVar(&includeHover, "hover", true, "Include hover information")
	cmd.Flags().BoolVar(&includeRefs, "refs", false, "Include references")
	cmd.Flags().BoolVar(&includeDefs, "defs", true, "Include definitions")

	return cmd
}

func newLSPCompletionCommand() *cobra.Command {
	var (
		project      string
		lspLine      int
		lspCharacter int
		maxResults   int
	)

	cmd := &cobra.Command{
		Use:   "completion [file-path]",
		Short: "Get completion items at position using LSP",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}

            cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_completion", map[string]any{
				"project":     project,
				"file":        args[0],
				"line":        lspLine,
				"character":   lspCharacter,
				"max_results": maxResults,
			})
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Path to project root")
	cmd.Flags().IntVar(&lspLine, "line", 0, "Line number (0-based)")
	cmd.Flags().IntVar(&lspCharacter, "character", 0, "Character number (0-based)")
	cmd.Flags().IntVar(&maxResults, "max-results", 20, "Maximum number of results")

	return cmd
}

func newLSPSymbolCommand() *cobra.Command {
	var (
		project    string
		query      string
		maxResults int
	)

	cmd := &cobra.Command{
		Use:   "symbols",
		Short: "Search workspace symbols using LSP",
		RunE: func(cmd *cobra.Command, args []string) error {
			if project == "" {
				return fmt.Errorf("--project is required")
			}
			if query == "" {
				return fmt.Errorf("--query is required")
			}

            cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_symbols", map[string]any{
				"project":     project,
				"query":       query,
				"max_results": maxResults,
			})
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().StringVar(&project, "project", "", "Path to project root")
	cmd.Flags().StringVar(&query, "query", "", "Search query")
	cmd.Flags().IntVar(&maxResults, "max-results", 50, "Maximum number of results")

	return cmd
}

func newLSPInstallCommand() *cobra.Command {
	var (
		installVersion string
		installDir     string
	)

	cmd := &cobra.Command{
		Use:   "install [server-name]",
		Short: "Install a language server to local directory",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			var serverName string
			if len(args) > 0 {
				serverName = args[0]
			} else {
				serverName = "vtsls" // Default to vtsls
			}

			// Create installation manager
			var installManager *lsp.InstallationManager
			if installDir != "" {
				installManager = lsp.NewInstallationManager(installDir)
			} else {
				installManager = lsp.NewInstallationManager("")
			}

			delegate := &lsp.SimpleDelegate{}

			fmt.Printf("Installing %s", serverName)
			if installVersion != "" {
				fmt.Printf(" version %s", installVersion)
			}
			fmt.Printf("...\n")

			binary, err := installManager.InstallServer(
				cmd.Context(),
				serverName,
				installVersion,
				delegate,
			)
			if err != nil {
				return fmt.Errorf("installation failed: %v", err)
			}

			fmt.Printf("✓ Successfully installed %s\n", serverName)
			fmt.Printf("  Binary: %s\n", binary.Path)
			fmt.Printf("  Args: %v\n", binary.Args)

			return nil
		},
	}

	cmd.Flags().StringVar(&installVersion, "version", "", "Specific version to install")
	cmd.Flags().
		StringVar(&installDir, "dir", "", "Installation directory (default: ~/.cache/ts-index/lsp-servers)")

	return cmd
}

func newLSPInstallByLanguageCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install-by-language [language]",
		Short: "Install language server by language type",
		Long: `Install a language server using the language server manager.
This command installs the appropriate LSP server for the specified language.

Supported languages: typescript, javascript`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			language := args[0]

			// Create language server manager with delegate
			delegate := &lsp.SimpleDelegate{}
			manager := lsp.NewLanguageServerManager(delegate)

			fmt.Printf("Installing language server for language '%s'...\n", language)

			err := manager.InstallLanguageServer(cmd.Context(), language)
			if err != nil {
				return fmt.Errorf("installation failed: %v", err)
			}

			fmt.Printf("✓ Successfully installed language server for language '%s'\n", language)
			return nil
		},
	}

	return cmd
}

func newLSPListCommand() *cobra.Command {
	var installDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed language servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_list", map[string]any{"dir": installDir})
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().
		StringVar(&installDir, "dir", "", "Installation directory (default: ~/.cache/ts-index/lsp-servers)")

	return cmd
}

func newLSPHealthCommand() *cobra.Command {
	var installDir string

	cmd := &cobra.Command{
		Use:   "health",
		Short: "Check LSP health and language server availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			cli, err := mcpclient.NewStdioClient(cmd.Context())
			if err != nil {
				return err
			}
			defer func() { _ = cli.Close() }()
			res, err := cli.Call(cmd.Context(), "lsp_health", map[string]any{"dir": installDir})
			if err != nil {
				return err
			}
			data, _ := json.MarshalIndent(res.StructuredContent, "", "  ")
			fmt.Println(string(data))
			return nil
		},
	}

	cmd.Flags().
		StringVar(&installDir, "dir", "", "Installation directory (default: ~/.cache/ts-index/lsp-servers)")

	return cmd
}
