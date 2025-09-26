package fx

import (
	"fmt"

	"github.com/0x5457/ts-index/internal/storage"
	"github.com/0x5457/ts-index/internal/storage/sqlite"
	"github.com/0x5457/ts-index/internal/storage/sqlvec"
	"go.uber.org/fx"
)

// StorageParams represents dependencies for storage components
type StorageParams struct {
	fx.In

	Config *Config
}

// NewSymbolStore creates a new symbol store instance
func NewSymbolStore(params StorageParams) (storage.SymbolStore, error) {
	if params.Config.DBPath == "" {
		return nil, fmt.Errorf("database path must be specified")
	}
	return sqlite.New(params.Config.DBPath)
}

// NewVectorStore creates a new vector store instance
func NewVectorStore(params StorageParams) (storage.VectorStore, error) {
	if params.Config.DBPath == "" {
		return nil, fmt.Errorf("database path must be specified")
	}
	return sqlvec.New(params.Config.DBPath, params.Config.VectorDimension)
}

// StorageModule provides storage components
var StorageModule = fx.Module("storage",
	fx.Provide(
		NewSymbolStore,
		NewVectorStore,
	),
)
