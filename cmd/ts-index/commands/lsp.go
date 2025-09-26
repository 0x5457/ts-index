package commands

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/0x5457/ts-index/internal/lsp"
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
			clientTools := lsp.NewClientTools()
			defer func() {
				if err := clientTools.Cleanup(); err != nil {
					log.Printf("Failed to cleanup client tools: %v", err)
				}
			}()

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

			clientTools := lsp.NewClientTools()
			defer func() {
				if err := clientTools.Cleanup(); err != nil {
					log.Printf("Failed to cleanup client tools: %v", err)
				}
			}()

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

			clientTools := lsp.NewClientTools()
			defer func() {
				if err := clientTools.Cleanup(); err != nil {
					log.Printf("Failed to cleanup client tools: %v", err)
				}
			}()

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

			clientTools := lsp.NewClientTools()
			defer func() {
				if err := clientTools.Cleanup(); err != nil {
					log.Printf("Failed to cleanup client tools: %v", err)
				}
			}()

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

func newLSPListCommand() *cobra.Command {
	var installDir string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installed language servers",
		RunE: func(cmd *cobra.Command, args []string) error {
			var installManager *lsp.InstallationManager
			if installDir != "" {
				installManager = lsp.NewInstallationManager(installDir)
			} else {
				installManager = lsp.NewInstallationManager("")
			}

			delegate := &lsp.SimpleDelegate{}
			servers, err := installManager.GetInstalledServers(delegate)
			if err != nil {
				return err
			}

			if len(servers) == 0 {
				fmt.Println("No language servers installed locally")
				fmt.Println("Use 'ts-index lsp install' to install a language server")
				return nil
			}

			fmt.Println("Installed Language Servers:")
			for _, server := range servers {
				fmt.Printf("  %s:\n", server.Name)
				for _, version := range server.Versions {
					fmt.Printf("    - %s\n", version)
				}
				fmt.Printf("    Path: %s\n", server.Path)
			}

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
			fmt.Println("System-wide installations:")
			if lsp.IsVTSLSInstalled() {
				fmt.Println("  ✓ vtsls is installed and available")
			} else {
				fmt.Println("  ✗ vtsls is not installed")
				fmt.Printf("    Install globally with: %s\n", lsp.InstallVTSLSCommand())
			}

			if lsp.IsTypeScriptLanguageServerInstalled() {
				fmt.Println("  ✓ typescript-language-server is installed and available")
			} else {
				fmt.Println("  ✗ typescript-language-server is not installed")
				fmt.Printf("    Install globally with: %s\n", lsp.InstallTypeScriptLanguageServerCommand())
			}

			// Check local installations
			var installManager *lsp.InstallationManager
			if installDir != "" {
				installManager = lsp.NewInstallationManager(installDir)
			} else {
				installManager = lsp.NewInstallationManager("")
			}

			delegate := &lsp.SimpleDelegate{}
			servers, err := installManager.GetInstalledServers(delegate)
			if err == nil && len(servers) > 0 {
				fmt.Println("\nLocal installations:")
				for _, server := range servers {
					fmt.Printf("  ✓ %s (versions: %v)\n", server.Name, server.Versions)
				}
			} else {
				fmt.Println("\nLocal installations:")
				fmt.Println("  None found")
				fmt.Println("  Use 'ts-index lsp install' to install language servers locally")
			}

			return nil
		},
	}

	cmd.Flags().
		StringVar(&installDir, "dir", "", "Installation directory (default: ~/.cache/ts-index/lsp-servers)")

	return cmd
}
