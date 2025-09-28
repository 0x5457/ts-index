package storagefx

import (
	"github.com/0x5457/ts-index/internal/config/configfx"
	"github.com/0x5457/ts-index/internal/storage"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
	"go.uber.org/fx"
)

// Params represents dependencies for storage components
type Params struct {
	fx.In

	Config *configfx.Config
}

// NewSymbolStore creates a new symbol store instance
func NewSymbolStore(params Params) (storage.SymbolStore, error) {
	if params.Config.DBPath == "" {
		// Return nil when no database path is provided (e.g., in MCP client mode)
		return nil, nil
	}
	return sqlite.New(params.Config.DBPath)
}

// NewVectorStore creates a new vector store instance
func NewVectorStore(params Params) (storage.VectorStore, error) {
	if params.Config.DBPath == "" {
		// Return nil when no database path is provided (e.g., in MCP client mode)
		return nil, nil
	}
	return sqlvec.New(params.Config.DBPath, params.Config.VectorDimension)
}

// Module provides storage components
var Module = fx.Module("storage",
	fx.Provide(
		fx.Annotate(NewSymbolStore, fx.ResultTags(`optional:"true"`)),
		fx.Annotate(NewVectorStore, fx.ResultTags(`optional:"true"`)),
	),
)
