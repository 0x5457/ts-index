package main

import (
	"log"

	"github.com/0x5457/ts-index/cmd/ts-index/commands"
	"github.com/spf13/cobra"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "ts-index",
		Short: "TypeScript code indexing and search tool",
		Long: `A powerful tool for indexing TypeScript projects 
		and performing semantic search with Language Server Protocol support.`,
	}

	// Add all command modules - now using Fx for dependency injection
	rootCmd.AddCommand(
		commands.NewIndexCommand(),
		commands.NewSearchCommand(),
		commands.NewLSPCommand(),
		commands.NewMCPServeCommand(),
		commands.NewMCPClientCommand(),
	)

	if err := rootCmd.Execute(); err != nil {
		log.Fatal(err)
	}
}
