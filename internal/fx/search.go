package fx

import (
	"github.com/0x5457/ts-index/internal/embeddings"
	"github.com/0x5457/ts-index/internal/search"
	"github.com/0x5457/ts-index/internal/storage"
	"go.uber.org/fx"
)

// SearchParams represents dependencies for search service
type SearchParams struct {
	fx.In

	Embedder embeddings.Embedder
	VecStore storage.VectorStore
}

// NewSearchService creates a new search service instance
func NewSearchService(params SearchParams) *search.Service {
	return &search.Service{
		Embedder: params.Embedder,
		Vector:   params.VecStore,
	}
}

// SearchModule provides search components
var SearchModule = fx.Module("search",
	fx.Provide(NewSearchService),
)
