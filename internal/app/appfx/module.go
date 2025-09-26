package appfx

import (
	"github.com/0x5457/ts-index/cmd/cmdsfx"
	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/embeddings/embeddingsfx"
	"github.com/0x5457/ts-index/internal/indexer/indexerfx"
	"github.com/0x5457/ts-index/internal/mcp/mcpfx"
	"github.com/0x5457/ts-index/internal/parser/parserfx"
	"github.com/0x5457/ts-index/internal/search/searchfx"
	"github.com/0x5457/ts-index/internal/storage/storagefx"
	"go.uber.org/fx"
)

// Module combines all application modules
var Module = fx.Options(
	configfx.Module,
	parserfx.Module,
	embeddingsfx.Module,
	storagefx.Module,
	searchfx.Module,
	indexerfx.Module,
	mcpfx.Module,
	cmdsfx.Module,
)

// NewAppWithConfig creates an Fx app with the given configuration values
func NewAppWithConfig(dbPath, embedURL, project string) *fx.App {
	return fx.New(
		Module,
		fx.Supply(
			fx.Annotate(dbPath, fx.ResultTags(`name:"dbPath"`)),
			fx.Annotate(embedURL, fx.ResultTags(`name:"embedURL"`)),
			fx.Annotate(project, fx.ResultTags(`name:"project"`)),
		),
		fx.Invoke(func(lc fx.Lifecycle, mcpLifecycle *mcpfx.Lifecycle) {
			lc.Append(fx.Hook{
				OnStart: mcpLifecycle.Start,
				OnStop:  mcpLifecycle.Stop,
			})
		}),
	)
}

// NewApp creates an Fx app with default configuration
func NewApp() *fx.App {
	return fx.New(Module)
}
