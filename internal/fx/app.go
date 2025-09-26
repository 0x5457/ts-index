package fx

import (
	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/mark3labs/mcp-go/server"
	"go.uber.org/fx"
)

// AppModule combines all application modules
var AppModule = fx.Options(
	ConfigModule,
	ParserModule,
	EmbeddingsModule,
	StorageModule,
	SearchModule,
	IndexerModule,
	MCPModule,
	CommandModule,
)

// Components holds all the main components for external access
type Components struct {
	fx.In

	Config        *Config
	SearchService *search.Service   `optional:"true"`
	Indexer       indexer.Indexer   `optional:"true"`
	MCPServer     *server.MCPServer `optional:"true"`
}

// NewAppWithConfig creates an Fx app with the given configuration values
func NewAppWithConfig(dbPath, embedURL, project string) *fx.App {
	return fx.New(
		AppModule,
		fx.Supply(
			fx.Annotate(dbPath, fx.As(new(string)), fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate(embedURL, fx.As(new(string)), fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate(project, fx.As(new(string)), fx.ResultTags(`name:"project"`)),
		),
		fx.Invoke(func(lc fx.Lifecycle, mcpLifecycle *MCPLifecycle) {
			lc.Append(fx.Hook{
				OnStart: mcpLifecycle.Start,
				OnStop:  mcpLifecycle.Stop,
			})
		}),
	)
}

// NewApp creates an Fx app with default configuration
func NewApp() *fx.App {
	return fx.New(
		AppModule,
	)
}
