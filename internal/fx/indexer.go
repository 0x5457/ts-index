package fx

import (
	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/indexer"
	"github.com/0x5457/ts-index/internal/indexer/pipeline"
	"github.com/0x5457/ts-index/internal/parser"
	"github.com/0x5457/ts-index/internal/storage"
	"go.uber.org/fx"
)

// IndexerParams represents dependencies for indexer components
type IndexerParams struct {
	fx.In

	Parser   parser.Parser
	Embedder embeddings.Embedder
	SymStore storage.SymbolStore
	VecStore storage.VectorStore
}

// NewIndexer creates a new indexer instance
func NewIndexer(params IndexerParams) indexer.Indexer {
	return pipeline.New(
		params.Parser,
		params.Embedder,
		params.SymStore,
		params.VecStore,
		pipeline.Options{},
	)
}

// IndexerModule provides indexer components
var IndexerModule = fx.Module("indexer",
	fx.Provide(NewIndexer),
)
