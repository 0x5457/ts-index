package storagefx

import (
	"fmt"

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
		return nil, fmt.Errorf("database path must be specified")
	}
	return sqlite.New(params.Config.DBPath)
}

// NewVectorStore creates a new vector store instance
func NewVectorStore(params Params) (storage.VectorStore, error) {
	if params.Config.DBPath == "" {
		return nil, fmt.Errorf("database path must be specified")
	}
	return sqlvec.New(params.Config.DBPath, params.Config.VectorDimension)
}

// Module provides storage components
var Module = fx.Module("storage",
	fx.Provide(
		NewSymbolStore,
		NewVectorStore,
	),
)
